package command

import (
	"fmt"

	"github.com/umbracle/ethgo"

	"github.com/DAO-Metaplayer/metachain/chain"
	"github.com/DAO-Metaplayer/metachain/server"
)

const (
	DefaultGenesisFileName           = "genesis.json"
	DefaultChainName                 = "metachain"
	DefaultChainID                   = 100
	DefaultConsensus                 = server.MBFTConsensus
	DefaultGenesisGasUsed            = 458752  // 0x70000
	DefaultGenesisGasLimit           = 5242880 // 0x500000
	DefaultGenesisBaseFeeEM          = chain.GenesisBaseFeeEM
	DefaultGenesisBaseFeeChangeDenom = chain.BaseFeeChangeDenom
)

var (
	DefaultStake                = ethgo.Ether(1e6)
	DefaultPremineBalance       = ethgo.Ether(1e6)
	DefaultGenesisBaseFee       = chain.GenesisBaseFee
	DefaultGenesisBaseFeeConfig = fmt.Sprintf(
		"%d:%d:%d",
		DefaultGenesisBaseFee,
		DefaultGenesisBaseFeeEM,
		DefaultGenesisBaseFeeChangeDenom,
	)
)

const (
	JSONOutputFlag  = "json"
	GRPCAddressFlag = "grpc-address"
	JSONRPCFlag     = "jsonrpc"
)

// GRPCAddressFlagLEGACY Legacy flag that needs to be present to preserve backwards
// compatibility with running clients
const (
	GRPCAddressFlagLEGACY = "grpc"
)
