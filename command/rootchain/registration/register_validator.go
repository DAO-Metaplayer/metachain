package registration

import (
	"encoding/hex"
	"errors"
	"fmt"

	"github.com/DAO-Metaplayer/metachain/command"
	"github.com/DAO-Metaplayer/metachain/command/helper"
	"github.com/DAO-Metaplayer/metachain/command/mbftsecrets"
	rootHelper "github.com/DAO-Metaplayer/metachain/command/rootchain/helper"
	"github.com/DAO-Metaplayer/metachain/consensus/mbft/contractsapi"
	bls "github.com/DAO-Metaplayer/metachain/consensus/mbft/signer"
	"github.com/DAO-Metaplayer/metachain/consensus/mbft/wallet"
	"github.com/DAO-Metaplayer/metachain/txrelayer"
	"github.com/DAO-Metaplayer/metachain/types"
	"github.com/spf13/cobra"
	"github.com/umbracle/ethgo"
)

var params registerParams

func GetCommand() *cobra.Command {
	registerCmd := &cobra.Command{
		Use:     "register-validator",
		Short:   "registers a whitelisted validator to supernet manager on rootchain",
		PreRunE: runPreRun,
		RunE:    runCommand,
	}

	setFlags(registerCmd)

	return registerCmd
}

func setFlags(cmd *cobra.Command) {
	cmd.Flags().StringVar(
		&params.accountDir,
		mbftsecrets.AccountDirFlag,
		"",
		mbftsecrets.AccountDirFlagDesc,
	)

	cmd.Flags().StringVar(
		&params.accountConfig,
		mbftsecrets.AccountConfigFlag,
		"",
		mbftsecrets.AccountConfigFlagDesc,
	)

	cmd.Flags().StringVar(
		&params.supernetManagerAddress,
		rootHelper.SupernetManagerFlag,
		"",
		rootHelper.SupernetManagerFlagDesc,
	)

	helper.RegisterJSONRPCFlag(cmd)
	cmd.MarkFlagsMutuallyExclusive(mbftsecrets.AccountConfigFlag, mbftsecrets.AccountDirFlag)
}

func runPreRun(cmd *cobra.Command, _ []string) error {
	params.jsonRPC = helper.GetJSONRPCAddress(cmd)

	return params.validateFlags()
}

func runCommand(cmd *cobra.Command, _ []string) error {
	outputter := command.InitializeOutputter(cmd)
	defer outputter.WriteOutput()

	secretsManager, err := mbftsecrets.GetSecretsManager(params.accountDir, params.accountConfig, true)
	if err != nil {
		return err
	}

	txRelayer, err := txrelayer.NewTxRelayer(txrelayer.WithIPAddress(params.jsonRPC))
	if err != nil {
		return err
	}

	rootChainID, err := txRelayer.Client().Eth().ChainID()
	if err != nil {
		return err
	}

	newValidatorAccount, err := wallet.NewAccountFromSecret(secretsManager)
	if err != nil {
		return err
	}

	koskSignature, err := bls.MakeKOSKSignature(
		newValidatorAccount.Bls, newValidatorAccount.Address(),
		rootChainID.Int64(), bls.DomainValidatorSet, types.StringToAddress(params.supernetManagerAddress))
	if err != nil {
		return err
	}

	receipt, err := registerValidator(txRelayer, newValidatorAccount, koskSignature)
	if err != nil {
		return err
	}

	if receipt.Status != uint64(types.ReceiptSuccess) {
		return errors.New("register validator transaction failed")
	}

	result := &registerResult{}
	foundLog := false

	var validatorRegisteredEvent contractsapi.ValidatorRegisteredEvent
	for _, log := range receipt.Logs {
		doesMatch, err := validatorRegisteredEvent.ParseLog(log)
		if err != nil {
			return err
		}

		if !doesMatch {
			continue
		}

		koskSignatureRaw, err := koskSignature.Marshal()
		if err != nil {
			return err
		}

		result.koskSignature = hex.EncodeToString(koskSignatureRaw)
		result.validatorAddress = validatorRegisteredEvent.Validator.String()

		foundLog = true

		break
	}

	if !foundLog {
		return fmt.Errorf("could not find an appropriate log in receipt that registration happened")
	}

	outputter.WriteCommandResult(result)

	return nil
}

func registerValidator(sender txrelayer.TxRelayer, account *wallet.Account,
	signature *bls.Signature) (*ethgo.Receipt, error) {
	sigMarshal, err := signature.ToBigInt()
	if err != nil {
		return nil, fmt.Errorf("register validator failed: %w", err)
	}

	registerFn := &contractsapi.RegisterCustomSupernetManagerFn{
		Signature: sigMarshal,
		Pubkey:    account.Bls.PublicKey().ToBigInt(),
	}

	input, err := registerFn.EncodeAbi()
	if err != nil {
		return nil, fmt.Errorf("register validator failed: %w", err)
	}

	supernetAddr := ethgo.Address(types.StringToAddress(params.supernetManagerAddress))
	txn := rootHelper.CreateTransaction(ethgo.ZeroAddress, &supernetAddr, input, nil, true)

	return sender.SendTransaction(txn, account.Ecdsa)
}
