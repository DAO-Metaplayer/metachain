package mbft

import (
	"github.com/DAO-Metaplayer/metachain/command/rootchain/registration"
	"github.com/DAO-Metaplayer/metachain/command/rootchain/staking"
	"github.com/DAO-Metaplayer/metachain/command/rootchain/supernet"
	"github.com/DAO-Metaplayer/metachain/command/rootchain/supernet/stakemanager"
	"github.com/DAO-Metaplayer/metachain/command/rootchain/validators"
	"github.com/DAO-Metaplayer/metachain/command/rootchain/whitelist"
	"github.com/DAO-Metaplayer/metachain/command/rootchain/withdraw"
	"github.com/DAO-Metaplayer/metachain/command/sidechain/rewards"
	"github.com/DAO-Metaplayer/metachain/command/sidechain/unstaking"
	sidechainWithdraw "github.com/DAO-Metaplayer/metachain/command/sidechain/withdraw"
	"github.com/spf13/cobra"
)

func GetCommand() *cobra.Command {
	mbftCmd := &cobra.Command{
		Use:   "mbft",
		Short: "Mbft command",
	}

	mbftCmd.AddCommand(
		// sidechain (validator set) command to unstake on child chain
		unstaking.GetCommand(),
		// sidechain (validator set) command to withdraw stake on child chain
		sidechainWithdraw.GetCommand(),
		// sidechain (reward pool) command to withdraw pending rewards
		rewards.GetCommand(),
		// rootchain (stake manager) command to withdraw stake
		withdraw.GetCommand(),
		// rootchain (supernet manager) command that queries validator info
		validators.GetCommand(),
		// rootchain (supernet manager) whitelist validator
		whitelist.GetCommand(),
		// rootchain (supernet manager) register validator
		registration.GetCommand(),
		// rootchain (stake manager) stake command
		staking.GetCommand(),
		// rootchain (supernet manager) command for finalizing genesis
		// validator set and enabling staking
		supernet.GetCommand(),
		// rootchain command for deploying stake manager
		stakemanager.GetCommand(),
	)

	return mbftCmd
}
