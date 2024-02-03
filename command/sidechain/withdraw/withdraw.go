package withdraw

import (
	"fmt"
	"time"

	"github.com/spf13/cobra"
	"github.com/umbracle/ethgo"

	"github.com/DAO-Metaplayer/metachain/command"
	"github.com/DAO-Metaplayer/metachain/command/bridge/common"
	"github.com/DAO-Metaplayer/metachain/command/helper"
	"github.com/DAO-Metaplayer/metachain/command/mbftsecrets"
	rootHelper "github.com/DAO-Metaplayer/metachain/command/rootchain/helper"
	sidechainHelper "github.com/DAO-Metaplayer/metachain/command/sidechain"
	"github.com/DAO-Metaplayer/metachain/consensus/mbft/contractsapi"
	"github.com/DAO-Metaplayer/metachain/contracts"
	"github.com/DAO-Metaplayer/metachain/txrelayer"
	"github.com/DAO-Metaplayer/metachain/types"
)

var params withdrawParams

func GetCommand() *cobra.Command {
	unstakeCmd := &cobra.Command{
		Use:     "withdraw-child",
		Short:   "Withdraws pending withdrawals on child chain for given validator",
		PreRunE: runPreRun,
		RunE:    runCommand,
	}

	helper.RegisterJSONRPCFlag(unstakeCmd)
	setFlags(unstakeCmd)

	return unstakeCmd
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

	encoded, err := contractsapi.ValidatorSet.Abi.Methods["withdraw"].Encode([]interface{}{})
	if err != nil {
		return err
	}

	receiver := (*ethgo.Address)(&contracts.ValidatorSetContract)
	txn := rootHelper.CreateTransaction(validatorAccount.Ecdsa.Address(), receiver, encoded, nil, false)

	receipt, err := txRelayer.SendTransaction(txn, validatorAccount.Ecdsa)
	if err != nil {
		return err
	}

	if receipt.Status != uint64(types.ReceiptSuccess) {
		return fmt.Errorf("withdraw transaction failed on block: %d", receipt.BlockNumber)
	}

	var (
		withdrawalEvent contractsapi.WithdrawalEvent
		foundLog        bool
	)

	// check the logs to check for the result
	for _, log := range receipt.Logs {
		doesMatch, err := withdrawalEvent.ParseLog(log)
		if err != nil {
			return err
		}

		if doesMatch {
			foundLog = true

			break
		}
	}

	if !foundLog {
		return fmt.Errorf("could not find an appropriate log in receipt that withdraw happened on ValidatorSet")
	}

	exitEventIDs, err := common.ExtractExitEventIDs(receipt)
	if err != nil {
		return fmt.Errorf("withdrawal failed: %w", err)
	}

	outputter.WriteCommandResult(
		&withdrawResult{
			ValidatorAddress: validatorAccount.Ecdsa.Address().String(),
			Amount:           withdrawalEvent.Amount,
			ExitEventIDs:     exitEventIDs,
			BlockNumber:      receipt.BlockNumber,
		})

	return nil
}
