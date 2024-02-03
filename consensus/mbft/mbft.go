// Package mbft implements PBFT consensus algorithm integration and bridge feature
package mbft

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"path/filepath"
	"time"

	"github.com/hashicorp/go-hclog"

	"github.com/DAO-Metaplayer/metachain/chain"
	"github.com/DAO-Metaplayer/metachain/consensus"
	"github.com/DAO-Metaplayer/metachain/consensus/mbft/contractsapi"
	bls "github.com/DAO-Metaplayer/metachain/consensus/mbft/signer"
	"github.com/DAO-Metaplayer/metachain/consensus/mbft/validator"
	"github.com/DAO-Metaplayer/metachain/consensus/mbft/wallet"
	"github.com/DAO-Metaplayer/metachain/contracts"
	"github.com/DAO-Metaplayer/metachain/forkmanager"
	"github.com/DAO-Metaplayer/metachain/helper/common"
	"github.com/DAO-Metaplayer/metachain/helper/progress"
	"github.com/DAO-Metaplayer/metachain/network"
	"github.com/DAO-Metaplayer/metachain/state"
	"github.com/DAO-Metaplayer/metachain/syncer"
	"github.com/DAO-Metaplayer/metachain/types"
)

const (
	minSyncPeers = 2
	pbftProto    = "/pbft/0.2"
	bridgeProto  = "/bridge/0.2"
)

var (
	errMissingBridgeConfig = errors.New("invalid genesis configuration, missing bridge configuration")
)

// mbftBackend is an interface defining mbft methods needed by fsm and sync tracker
type mbftBackend interface {
	// GetValidators retrieves validator set for the given block
	GetValidators(blockNumber uint64, parents []*types.Header) (validator.AccountSet, error)
}

// Factory is the factory function to create a discovery consensus
func Factory(params *consensus.Params) (consensus.Consensus, error) {
	logger := params.Logger.Named("mbft")

	setupHeaderHashFunc()

	mbft := &Mbft{
		config:  params,
		closeCh: make(chan struct{}),
		logger:  logger,
		txPool:  params.TxPool,
	}

	// initialize mbft consensus config
	customConfigJSON, err := json.Marshal(params.Config.Config)
	if err != nil {
		return nil, err
	}

	err = json.Unmarshal(customConfigJSON, &mbft.consensusConfig)
	if err != nil {
		return nil, err
	}

	return mbft, nil
}

type Mbft struct {
	// closeCh is used to signal that consensus protocol is stopped
	closeCh chan struct{}

	// metabft is the metabft engine
	metabft *METABFTConsensusWrapper

	// state is reference to the struct which encapsulates consensus data persistence logic
	state *State

	// consensus parameters
	config *consensus.Params

	// consensusConfig is genesis configuration for mbft consensus protocol
	consensusConfig *MBFTConfig

	// blockchain is a reference to the blockchain object
	blockchain blockchainBackend

	// runtime handles consensus runtime features like epoch, state and event management
	runtime *consensusRuntime

	// block time duration
	blockTime time.Duration

	// dataDir is the data directory to store the info
	dataDir string

	// reference to the syncer
	syncer syncer.Syncer

	// topic for consensus engine messages
	consensusTopic *network.Topic

	// topic for bridge messages
	bridgeTopic *network.Topic

	// key encapsulates ECDSA address and BLS signing logic
	key *wallet.Key

	// validatorsCache represents cache of validators snapshots
	validatorsCache *validatorsSnapshotCache

	// logger
	logger hclog.Logger

	// tx pool as interface
	txPool txPoolInterface
}

func GenesisPostHookFactory(config *chain.Chain, engineName string) func(txn *state.Transition) error {
	return func(transition *state.Transition) error {
		mBFTConfig, err := GetMBFTConfig(config)
		if err != nil {
			return err
		}

		// calculate initial total supply of native erc20 token
		// we skip zero address, since its a special case address
		// that is used for minting and burning native token
		initialTotalSupply := big.NewInt(0)

		for addr, alloc := range config.Genesis.Alloc {
			if addr == types.ZeroAddress {
				continue
			}

			initialTotalSupply.Add(initialTotalSupply, alloc.Balance)
		}

		bridgeCfg := mBFTConfig.Bridge
		if bridgeCfg == nil {
			return errMissingBridgeConfig
		}

		proxyAddrMapping := contracts.GetProxyImplementationMapping()

		burnContractAddress, isBurnContractSet := getBurnContractAddress(config, mBFTConfig)
		if isBurnContractSet {
			proxyAddrMapping[contracts.DefaultBurnContract] = burnContractAddress
		}

		if _, ok := config.Genesis.Alloc[contracts.RewardTokenContract]; ok {
			proxyAddrMapping[contracts.RewardTokenContract] = contracts.RewardTokenContractV1
		}

		if err = initProxies(transition, mBFTConfig.ProxyContractsAdmin, proxyAddrMapping); err != nil {
			return err
		}

		// initialize ValidatorSet SC
		if err = initValidatorSet(mBFTConfig, transition); err != nil {
			return err
		}

		// approve reward pool
		if err = approveRewardPoolAsSpender(mBFTConfig, transition); err != nil {
			return err
		}

		// mint reward tokens to reward wallet
		if err = mintRewardTokensToWallet(mBFTConfig, transition); err != nil {
			return err
		}

		// initialize RewardPool SC
		if err = initRewardPool(mBFTConfig, transition); err != nil {
			return err
		}

		// check if there are Bridge Allow List Admins and Bridge Block List Admins
		// and if there are, get the first address as the Admin
		bridgeAllowListAdmin := types.ZeroAddress
		if config.Params.BridgeAllowList != nil && len(config.Params.BridgeAllowList.AdminAddresses) > 0 {
			bridgeAllowListAdmin = config.Params.BridgeAllowList.AdminAddresses[0]
		}

		bridgeBlockListAdmin := types.ZeroAddress
		if config.Params.BridgeBlockList != nil && len(config.Params.BridgeBlockList.AdminAddresses) > 0 {
			bridgeBlockListAdmin = config.Params.BridgeBlockList.AdminAddresses[0]
		}

		// initialize Predicate SCs
		if bridgeAllowListAdmin != types.ZeroAddress || bridgeBlockListAdmin != types.ZeroAddress {
			// The owner of the contract will be the allow list admin or the block list admin, if any of them is set.
			owner := contracts.SystemCaller
			useBridgeAllowList := bridgeAllowListAdmin != types.ZeroAddress
			useBridgeBlockList := bridgeBlockListAdmin != types.ZeroAddress

			if bridgeAllowListAdmin != types.ZeroAddress {
				owner = bridgeAllowListAdmin
			} else if bridgeBlockListAdmin != types.ZeroAddress {
				owner = bridgeBlockListAdmin
			}

			// initialize ChildERC20PredicateAccessList SC
			input, err := getInitERC20PredicateACLInput(mBFTConfig.Bridge, owner,
				useBridgeAllowList, useBridgeBlockList, false)
			if err != nil {
				return err
			}

			if err = callContract(contracts.SystemCaller, contracts.ChildERC20PredicateContract, input,
				"ChildERC20PredicateAccessList", transition); err != nil {
				return err
			}

			// initialize ChildERC721PredicateAccessList SC
			input, err = getInitERC721PredicateACLInput(mBFTConfig.Bridge, owner,
				useBridgeAllowList, useBridgeBlockList, false)
			if err != nil {
				return err
			}

			if err = callContract(contracts.SystemCaller, contracts.ChildERC721PredicateContract, input,
				"ChildERC721PredicateAccessList", transition); err != nil {
				return err
			}

			// initialize ChildERC1155PredicateAccessList SC
			input, err = getInitERC1155PredicateACLInput(mBFTConfig.Bridge, owner,
				useBridgeAllowList, useBridgeBlockList, false)
			if err != nil {
				return err
			}

			if err = callContract(contracts.SystemCaller, contracts.ChildERC1155PredicateContract, input,
				"ChildERC1155PredicateAccessList", transition); err != nil {
				return err
			}

			// initialize RootMintableERC20PredicateAccessList SC
			input, err = getInitERC20PredicateACLInput(mBFTConfig.Bridge, owner,
				useBridgeAllowList, useBridgeBlockList, true)
			if err != nil {
				return err
			}

			if err = callContract(contracts.SystemCaller, contracts.RootMintableERC20PredicateContract, input,
				"RootMintableERC20PredicateAccessList", transition); err != nil {
				return err
			}

			// initialize RootMintableERC721PredicateAccessList SC
			input, err = getInitERC721PredicateACLInput(mBFTConfig.Bridge, owner,
				useBridgeAllowList, useBridgeBlockList, true)
			if err != nil {
				return err
			}

			if err = callContract(contracts.SystemCaller, contracts.RootMintableERC721PredicateContract, input,
				"RootMintableERC721PredicateAccessList", transition); err != nil {
				return err
			}

			// initialize RootMintableERC1155PredicateAccessList SC
			input, err = getInitERC1155PredicateACLInput(mBFTConfig.Bridge, owner,
				useBridgeAllowList, useBridgeBlockList, true)
			if err != nil {
				return err
			}

			if err = callContract(contracts.SystemCaller, contracts.RootMintableERC1155PredicateContract, input,
				"RootMintableERC1155PredicateAccessList", transition); err != nil {
				return err
			}
		} else {
			// initialize ChildERC20Predicate SC
			input, err := getInitERC20PredicateInput(bridgeCfg, false)
			if err != nil {
				return err
			}

			if err = callContract(contracts.SystemCaller, contracts.ChildERC20PredicateContract, input,
				"ChildERC20Predicate", transition); err != nil {
				return err
			}

			// initialize ChildERC721Predicate SC
			input, err = getInitERC721PredicateInput(bridgeCfg, false)
			if err != nil {
				return err
			}

			if err = callContract(contracts.SystemCaller, contracts.ChildERC721PredicateContract, input,
				"ChildERC721Predicate", transition); err != nil {
				return err
			}

			// initialize ChildERC1155Predicate SC
			input, err = getInitERC1155PredicateInput(bridgeCfg, false)
			if err != nil {
				return err
			}

			if err = callContract(contracts.SystemCaller, contracts.ChildERC1155PredicateContract, input,
				"ChildERC1155Predicate", transition); err != nil {
				return err
			}

			// initialize RootMintableERC20Predicate SC
			input, err = getInitERC20PredicateInput(bridgeCfg, true)
			if err != nil {
				return err
			}

			if err = callContract(contracts.SystemCaller, contracts.RootMintableERC20PredicateContract, input,
				"RootMintableERC20Predicate", transition); err != nil {
				return err
			}

			// initialize RootMintableERC721Predicate SC
			input, err = getInitERC721PredicateInput(bridgeCfg, true)
			if err != nil {
				return err
			}

			if err = callContract(contracts.SystemCaller, contracts.RootMintableERC721PredicateContract, input,
				"RootMintableERC721Predicate", transition); err != nil {
				return err
			}

			// initialize RootMintableERC1155Predicate SC
			input, err = getInitERC1155PredicateInput(bridgeCfg, true)
			if err != nil {
				return err
			}

			if err = callContract(contracts.SystemCaller, contracts.RootMintableERC1155PredicateContract, input,
				"RootMintableERC1155Predicate", transition); err != nil {
				return err
			}
		}

		if mBFTConfig.NativeTokenConfig.IsMintable {
			// initialize NativeERC20Mintable SC
			params := &contractsapi.InitializeNativeERC20MintableFn{
				Predicate_:   contracts.ChildERC20PredicateContract,
				Owner_:       mBFTConfig.NativeTokenConfig.Owner,
				RootToken_:   types.ZeroAddress, // in case native mintable token is used, it is always root token
				Name_:        mBFTConfig.NativeTokenConfig.Name,
				Symbol_:      mBFTConfig.NativeTokenConfig.Symbol,
				Decimals_:    mBFTConfig.NativeTokenConfig.Decimals,
				TokenSupply_: initialTotalSupply,
			}

			input, err := params.EncodeAbi()
			if err != nil {
				return err
			}

			if err = callContract(contracts.SystemCaller,
				contracts.NativeERC20TokenContract, input, "NativeERC20Mintable", transition); err != nil {
				return err
			}
		} else {
			// initialize NativeERC20 SC
			params := &contractsapi.InitializeNativeERC20Fn{
				Name_:        mBFTConfig.NativeTokenConfig.Name,
				Symbol_:      mBFTConfig.NativeTokenConfig.Symbol,
				Decimals_:    mBFTConfig.NativeTokenConfig.Decimals,
				RootToken_:   mBFTConfig.Bridge.RootNativeERC20Addr,
				Predicate_:   contracts.ChildERC20PredicateContract,
				TokenSupply_: initialTotalSupply,
			}

			input, err := params.EncodeAbi()
			if err != nil {
				return err
			}

			if err = callContract(contracts.SystemCaller,
				contracts.NativeERC20TokenContract, input, "NativeERC20", transition); err != nil {
				return err
			}

			// initialize EIP1559Burn SC
			if isBurnContractSet {
				burnParams := &contractsapi.InitializeEIP1559BurnFn{
					NewChildERC20Predicate: contracts.ChildERC20PredicateContract,
					NewBurnDestination:     config.Params.BurnContractDestinationAddress,
				}

				input, err = burnParams.EncodeAbi()
				if err != nil {
					return err
				}

				if err = callContract(contracts.SystemCaller,
					burnContractAddress,
					input, "EIP1559Burn", transition); err != nil {
					return err
				}
			}
		}

		return nil
	}
}

func ForkManagerFactory(forks *chain.Forks) error {
	// place fork manager handler registration here
	return nil
}

// Initialize initializes the consensus (e.g. setup data)
func (p *Mbft) Initialize() error {
	p.logger.Info("initializing mbft...")

	// read account
	account, err := wallet.NewAccountFromSecret(p.config.SecretsManager)
	if err != nil {
		return fmt.Errorf("failed to read account data. Error: %w", err)
	}

	// set key
	p.key = wallet.NewKey(account)

	// create and set syncer
	p.syncer = syncer.NewSyncer(
		p.config.Logger.Named("syncer"),
		p.config.Network,
		p.config.Blockchain,
		time.Duration(p.config.BlockTime)*3*time.Second,
	)

	// set blockchain backend
	p.blockchain = &blockchainWrapper{
		blockchain: p.config.Blockchain,
		executor:   p.config.Executor,
	}

	// create bridge and consensus topics
	if err = p.createTopics(); err != nil {
		return fmt.Errorf("cannot create topics: %w", err)
	}

	// set block time
	p.blockTime = time.Duration(p.config.BlockTime)

	// initialize mbft consensus data directory
	p.dataDir = filepath.Join(p.config.Config.Path, "mbft")
	// create the data dir if not exists
	if err = common.CreateDirSafe(p.dataDir, 0750); err != nil {
		return fmt.Errorf("failed to create data directory. Error: %w", err)
	}

	stt, err := newState(filepath.Join(p.dataDir, stateFileName), p.logger, p.closeCh)
	if err != nil {
		return fmt.Errorf("failed to create state instance. Error: %w", err)
	}

	p.state = stt
	p.validatorsCache = newValidatorsSnapshotCache(p.config.Logger, stt, p.blockchain)

	// create runtime
	if err := p.initRuntime(); err != nil {
		return err
	}

	p.metabft = newMETABFTConsensusWrapper(p.logger, p.runtime, p)

	if err = p.subscribeTometabftTopic(); err != nil {
		return fmt.Errorf("METABFT topic subscription failed: %w", err)
	}

	return nil
}

func ForkManagerInitialParamsFactory(config *chain.Chain) (*forkmanager.ForkParams, error) {
	pbftConfig, err := GetMBFTConfig(config)
	if err != nil {
		return nil, err
	}

	return &forkmanager.ForkParams{
		MaxValidatorSetSize: &pbftConfig.MaxValidatorSetSize,
		EpochSize:           &pbftConfig.EpochSize,
		SprintSize:          &pbftConfig.SprintSize,
		BlockTime:           &pbftConfig.BlockTime,
		BlockTimeDrift:      &pbftConfig.BlockTimeDrift,
	}, nil
}

// Start starts the consensus and servers
func (p *Mbft) Start() error {
	p.logger.Info("starting mbft consensus", "signer", p.key.String())

	// start syncer (also initializes peer map)
	if err := p.syncer.Start(); err != nil {
		return fmt.Errorf("failed to start syncer. Error: %w", err)
	}

	// sync concurrently, retrying indefinitely
	go common.RetryForever(context.Background(), time.Second, func(context.Context) error {
		blockHandler := func(b *types.FullBlock) bool {
			p.runtime.OnBlockInserted(b)

			return false
		}
		if err := p.syncer.Sync(blockHandler); err != nil {
			p.logger.Error("blocks synchronization failed", "error", err)

			return err
		}

		return nil
	})

	// start consensus runtime
	if err := p.startRuntime(); err != nil {
		return fmt.Errorf("consensus runtime start failed: %w", err)
	}

	// start state DB process
	go p.state.startStatsReleasing()

	// mbft rootchain metrics
	go p.publishRootchainMetrics(p.logger.Named("rootchain_metrics"))

	return nil
}

// initRuntime creates consensus runtime
func (p *Mbft) initRuntime() error {
	runtimeConfig := &runtimeConfig{
		MBFTConfig:            p.consensusConfig,
		Key:                   p.key,
		DataDir:               p.dataDir,
		State:                 p.state,
		blockchain:            p.blockchain,
		mbftBackend:           p,
		txPool:                p.txPool,
		bridgeTopic:           p.bridgeTopic,
		numBlockConfirmations: p.config.NumBlockConfirmations,
	}

	runtime, err := newConsensusRuntime(p.logger, runtimeConfig)
	if err != nil {
		return err
	}

	p.runtime = runtime

	return nil
}

// startRuntime starts consensus runtime
func (p *Mbft) startRuntime() error {
	go p.startConsensusProtocol()

	return nil
}

func (p *Mbft) startConsensusProtocol() {
	// wait to have at least n peers connected. The 2 is just an initial heuristic value
	// Most likely we will parametrize this in the future.
	if !p.waitForNPeers() {
		return
	}

	p.logger.Debug("peers connected")

	newBlockSub := p.blockchain.SubscribeEvents()
	defer newBlockSub.Close()

	syncerBlockCh := make(chan struct{})

	go func() {
		eventCh := newBlockSub.GetEventCh()

		for {
			select {
			case <-p.closeCh:
				return
			case ev := <-eventCh:
				// The blockchain notification system can eventually deliver
				// stale block notifications. These should be ignored
				if ev.Source == "syncer" && ev.NewChain[0].Number >= p.blockchain.CurrentHeader().Number {
					p.logger.Info("sync block notification received", "block height", ev.NewChain[0].Number,
						"current height", p.blockchain.CurrentHeader().Number)
					syncerBlockCh <- struct{}{}
				}
			}
		}
	}()

	var (
		sequenceCh   <-chan struct{}
		stopSequence func()
	)

	for {
		latestHeader := p.blockchain.CurrentHeader()

		currentValidators, err := p.GetValidators(latestHeader.Number, nil)
		if err != nil {
			p.logger.Error("failed to query current validator set", "block number", latestHeader.Number, "error", err)
		}

		isValidator := currentValidators.ContainsNodeID(p.key.String())
		p.runtime.setIsActiveValidator(isValidator)

		p.txPool.SetSealing(isValidator) // update tx pool

		if isValidator {
			// initialize FSM as a stateless metabft backend via runtime as an adapter
			err = p.runtime.FSM()
			if err != nil {
				p.logger.Error("failed to create fsm", "block number", latestHeader.Number, "error", err)

				continue
			}

			sequenceCh, stopSequence = p.metabft.runSequence(latestHeader.Number + 1)
		}

		now := time.Now().UTC()

		select {
		case <-syncerBlockCh:
			if isValidator {
				stopSequence()
				p.logger.Info("canceled sequence", "sequence", latestHeader.Number+1)
			}
		case <-sequenceCh:
		case <-p.closeCh:
			if isValidator {
				stopSequence()
			}

			return
		}

		p.logger.Debug("time to run the sequence", "seconds", time.Since(now))
	}
}

func (p *Mbft) waitForNPeers() bool {
	for {
		select {
		case <-p.closeCh:
			return false
		case <-time.After(2 * time.Second):
		}

		if len(p.config.Network.Peers()) >= minSyncPeers {
			break
		}
	}

	return true
}

// Close closes the connection
func (p *Mbft) Close() error {
	if p.syncer != nil {
		if err := p.syncer.Close(); err != nil {
			return err
		}
	}

	close(p.closeCh)
	p.runtime.close()

	return nil
}

// GetSyncProgression retrieves the current sync progression, if any
func (p *Mbft) GetSyncProgression() *progress.Progression {
	return p.syncer.GetSyncProgression()
}

// VerifyHeader implements consensus.Engine and checks whether a header conforms to the consensus rules
func (p *Mbft) VerifyHeader(header *types.Header) error {
	// Short circuit if the header is known
	if _, ok := p.blockchain.GetHeaderByHash(header.Hash); ok {
		return nil
	}

	parent, ok := p.blockchain.GetHeaderByHash(header.ParentHash)
	if !ok {
		return fmt.Errorf(
			"unable to get parent header by hash for block number %d",
			header.Number,
		)
	}

	return p.verifyHeaderImpl(parent, header, p.consensusConfig.BlockTimeDrift, nil)
}

func (p *Mbft) verifyHeaderImpl(parent, header *types.Header, blockTimeDrift uint64, parents []*types.Header) error {
	// validate header fields
	if err := validateHeaderFields(parent, header, blockTimeDrift); err != nil {
		return fmt.Errorf("failed to validate header for block %d. error = %w", header.Number, err)
	}

	// decode the extra data
	extra, err := GetmetabftExtra(header.ExtraData)
	if err != nil {
		return fmt.Errorf("failed to verify header for block %d. get extra error = %w", header.Number, err)
	}

	// validate extra data
	return extra.ValidateFinalizedData(
		header, parent, parents, p.blockchain.GetChainID(), p, bls.DomainCheckpointManager, p.logger)
}

func (p *Mbft) GetValidators(blockNumber uint64, parents []*types.Header) (validator.AccountSet, error) {
	return p.validatorsCache.GetSnapshot(blockNumber, parents)
}

// ProcessHeaders updates the snapshot based on the verified headers
func (p *Mbft) ProcessHeaders(_ []*types.Header) error {
	// Not required
	return nil
}

// GetBlockCreator retrieves the block creator (or signer) given the block header
func (p *Mbft) GetBlockCreator(h *types.Header) (types.Address, error) {
	return types.BytesToAddress(h.Miner), nil
}

// PreCommitState a hook to be called before finalizing state transition on inserting block
func (p *Mbft) PreCommitState(block *types.Block, _ *state.Transition) error {
	commitmentTxExists := false

	validators, err := p.GetValidators(block.Number()-1, nil)
	if err != nil {
		return err
	}

	// validate commitment state transactions
	for _, tx := range block.Transactions {
		if tx.Type != types.StateTx {
			continue
		}

		decodedStateTx, err := decodeStateTransaction(tx.Input)
		if err != nil {
			return fmt.Errorf("unknown state transaction: tx=%v, error: %w", tx.Hash, err)
		}

		if signedCommitment, ok := decodedStateTx.(*CommitmentMessageSigned); ok {
			if commitmentTxExists {
				return fmt.Errorf("only one commitment state tx is allowed per block: %v", tx.Hash)
			}

			commitmentTxExists = true

			if err := verifyBridgeCommitmentTx(
				block.Number(),
				tx.Hash,
				signedCommitment,
				validator.NewValidatorSet(validators, p.logger)); err != nil {
				return err
			}
		}
	}

	return nil
}

// GetBridgeProvider is an implementation of Consensus interface
// Returns an instance of BridgeDataProvider
func (p *Mbft) GetBridgeProvider() consensus.BridgeDataProvider {
	return p.runtime
}

// FilterExtra is an implementation of Consensus interface
func (p *Mbft) FilterExtra(extra []byte) ([]byte, error) {
	return GetmetabftExtraClean(extra)
}

// initProxies initializes proxy contracts, that allow upgradeability of contracts implementation
func initProxies(transition *state.Transition, admin types.Address,
	proxyToImplMap map[types.Address]types.Address) error {
	for proxyAddress, implAddress := range proxyToImplMap {
		protectSetupProxyFn := &contractsapi.ProtectSetUpProxyGenesisProxyFn{Initiator: contracts.SystemCaller}

		proxyInput, err := protectSetupProxyFn.EncodeAbi()
		if err != nil {
			return fmt.Errorf("GenesisProxy.protectSetUpProxy params encoding failed: %w", err)
		}

		err = callContract(contracts.SystemCaller, proxyAddress, proxyInput, "GenesisProxy.protectSetUpProxy", transition)
		if err != nil {
			return err
		}

		setUpproxyFn := &contractsapi.SetUpProxyGenesisProxyFn{
			Logic: implAddress,
			Admin: admin,
			Data:  []byte{},
		}

		proxyInput, err = setUpproxyFn.EncodeAbi()
		if err != nil {
			return fmt.Errorf("GenesisProxy.setUpProxy params encoding failed: %w", err)
		}

		err = callContract(contracts.SystemCaller, proxyAddress, proxyInput, "GenesisProxy.setUpProxy", transition)
		if err != nil {
			return err
		}
	}

	return nil
}

func getBurnContractAddress(config *chain.Chain, mBFTConfig MBFTConfig) (types.Address, bool) {
	if config.Params.BurnContract != nil &&
		len(config.Params.BurnContract) == 1 &&
		!mBFTConfig.NativeTokenConfig.IsMintable {
		for _, address := range config.Params.BurnContract {
			if _, ok := config.Genesis.Alloc[address]; ok {
				return address, true
			}
		}
	}

	return types.ZeroAddress, false
}
