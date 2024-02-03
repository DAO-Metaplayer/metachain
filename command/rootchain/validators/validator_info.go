package validators

import (
	"fmt"

	"github.com/DAO-Metaplayer/metachain/command"
	"github.com/DAO-Metaplayer/metachain/command/helper"
	"github.com/DAO-Metaplayer/metachain/command/mbftsecrets"
	rootHelper "github.com/DAO-Metaplayer/metachain/command/rootchain/helper"
	sidechainHelper "github.com/DAO-Metaplayer/metachain/command/sidechain"
	"github.com/DAO-Metaplayer/metachain/txrelayer"
	"github.com/DAO-Metaplayer/metachain/types"
	"github.com/spf13/cobra"
)

var (
	params validatorInfoParams
)

func GetCommand() *cobra.Command {
	validatorInfoCmd := &cobra.Command{
		Use:     "validator-info",
		Short:   "Gets validator info",
		PreRunE: runPreRun,
		RunE:    runCommand,
	}

	helper.RegisterJSONRPCFlag(validatorInfoCmd)
	setFlags(validatorInfoCmd)

	return validatorInfoCmd
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

	cmd.Flags().StringVar(
		&params.stakeManagerAddress,
		rootHelper.StakeManagerFlag,
		"",
		rootHelper.StakeManagerFlagDesc,
	)

	cmd.Flags().Int64Var(
		&params.chainID,
		mbftsecrets.ChainIDFlag,
		0,
		mbftsecrets.ChainIDFlagDesc,
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

	txRelayer, err := txrelayer.NewTxRelayer(txrelayer.WithIPAddress(params.jsonRPC))
	if err != nil {
		return err
	}

	validatorAddr := validatorAccount.Ecdsa.Address()
	supernetManagerAddr := types.StringToAddress(params.supernetManagerAddress)
	stakeManagerAddr := types.StringToAddress(params.stakeManagerAddress)

	validatorInfo, err := rootHelper.GetValidatorInfo(validatorAddr,
		supernetManagerAddr, stakeManagerAddr, params.chainID, txRelayer)
	if err != nil {
		return fmt.Errorf("failed to get validator info for %s: %w", validatorAddr, err)
	}

	outputter.WriteCommandResult(&validatorsInfoResult{
		Address:     validatorInfo.Address.String(),
		Stake:       validatorInfo.Stake.Uint64(),
		Active:      validatorInfo.IsActive,
		Whitelisted: validatorInfo.IsWhitelisted,
	})

	return nil
}
