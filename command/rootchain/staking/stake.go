package staking

import (
	"fmt"
	"math/big"
	"time"

	"github.com/DAO-Metaplayer/metachain/command"
	"github.com/DAO-Metaplayer/metachain/command/helper"
	"github.com/DAO-Metaplayer/metachain/command/mbftsecrets"
	rootHelper "github.com/DAO-Metaplayer/metachain/command/rootchain/helper"
	sidechainHelper "github.com/DAO-Metaplayer/metachain/command/sidechain"
	"github.com/DAO-Metaplayer/metachain/consensus/mbft/contractsapi"
	"github.com/DAO-Metaplayer/metachain/txrelayer"
	"github.com/DAO-Metaplayer/metachain/types"
	"github.com/spf13/cobra"
	"github.com/umbracle/ethgo"
)

var (
	params stakeParams
)

func GetCommand() *cobra.Command {
	stakeCmd := &cobra.Command{
		Use:     "stake",
		Short:   "Stakes the amount sent for validator on rootchain",
		PreRunE: runPreRun,
		RunE:    runCommand,
	}

	helper.RegisterJSONRPCFlag(stakeCmd)
	setFlags(stakeCmd)

	return stakeCmd
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
		&params.stakeManagerAddr,
		rootHelper.StakeManagerFlag,
		"",
		rootHelper.StakeManagerFlagDesc,
	)

	cmd.Flags().StringVar(
		&params.amount,
		sidechainHelper.AmountFlag,
		"",
		"amount to stake",
	)

	cmd.Flags().Int64Var(
		&params.supernetID,
		supernetIDFlag,
		0,
		"ID of supernet provided by stake manager on supernet registration",
	)

	cmd.Flags().StringVar(
		&params.stakeTokenAddr,
		rootHelper.StakeTokenFlag,
		"",
		rootHelper.StakeTokenFlagDesc,
	)

	cmd.MarkFlagsMutuallyExclusive(mbftsecrets.AccountDirFlag, mbftsecrets.AccountConfigFlag)
}

func runPreRun(cmd *cobra.Command, _ []string) error {
	params.jsonRPC = helper.GetJSONRPCAddress(cmd)

	return params.validateFlags()
}

func runCommand(cmd *cobra.Command, _ []string) error {
	outputter := command.InitializeOutputter(cmd)
	defer outputter.WriteOutput()

	validatorAccount, err := sidechainHelper.GetAccount(params.accountDir, params.accountConfig)
	if err != nil {
		return err
	}

	txRelayer, err := txrelayer.NewTxRelayer(txrelayer.WithIPAddress(params.jsonRPC),
		txrelayer.WithReceiptTimeout(150*time.Millisecond))
	if err != nil {
		return err
	}

	approveTxn, err := rootHelper.CreateApproveERC20Txn(params.amountValue,
		types.StringToAddress(params.stakeManagerAddr), types.StringToAddress(params.stakeTokenAddr), true)
	if err != nil {
		return err
	}

	receipt, err := txRelayer.SendTransaction(approveTxn, validatorAccount.Ecdsa)
	if err != nil {
		return err
	}

	if receipt.Status == uint64(types.ReceiptFailed) {
		return fmt.Errorf("approve transaction failed on block %d", receipt.BlockNumber)
	}

	stakeFn := contractsapi.StakeForStakeManagerFn{
		ID:     new(big.Int).SetInt64(params.supernetID),
		Amount: params.amountValue,
	}

	encoded, err := stakeFn.EncodeAbi()
	if err != nil {
		return err
	}

	stakeManagerAddr := ethgo.Address(types.StringToAddress(params.stakeManagerAddr))

	txn := rootHelper.CreateTransaction(validatorAccount.Ecdsa.Address(), &stakeManagerAddr, encoded, nil, true)

	receipt, err = txRelayer.SendTransaction(txn, validatorAccount.Ecdsa)
	if err != nil {
		return err
	}

	if receipt.Status == uint64(types.ReceiptFailed) {
		return fmt.Errorf("staking transaction failed on block %d", receipt.BlockNumber)
	}

	result := &stakeResult{
		ValidatorAddress: validatorAccount.Ecdsa.Address().String(),
	}

	var (
		stakeAddedEvent contractsapi.StakeAddedEvent
		foundLog        bool
	)

	// check the logs to check for the result
	for _, log := range receipt.Logs {
		doesMatch, err := stakeAddedEvent.ParseLog(log)
		if err != nil {
			return err
		}

		if !doesMatch {
			continue
		}

		result.Amount = stakeAddedEvent.Amount
		result.ValidatorAddress = stakeAddedEvent.Validator.String()
		foundLog = true

		break
	}

	if !foundLog {
		return fmt.Errorf("could not find an appropriate log in receipt that stake happened")
	}

	outputter.WriteCommandResult(result)

	return nil
}
