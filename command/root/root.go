package root

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/DAO-Metaplayer/metachain/command/backup"
	"github.com/DAO-Metaplayer/metachain/command/bridge"
	"github.com/DAO-Metaplayer/metachain/command/genesis"
	"github.com/DAO-Metaplayer/metachain/command/helper"
	"github.com/DAO-Metaplayer/metachain/command/license"
	"github.com/DAO-Metaplayer/metachain/command/mbft"
	"github.com/DAO-Metaplayer/metachain/command/mbftsecrets"
	"github.com/DAO-Metaplayer/metachain/command/monitor"
	"github.com/DAO-Metaplayer/metachain/command/peers"
	"github.com/DAO-Metaplayer/metachain/command/regenesis"
	"github.com/DAO-Metaplayer/metachain/command/rootchain"
	"github.com/DAO-Metaplayer/metachain/command/secrets"
	"github.com/DAO-Metaplayer/metachain/command/server"
	"github.com/DAO-Metaplayer/metachain/command/status"
	"github.com/DAO-Metaplayer/metachain/command/txpool"
	"github.com/DAO-Metaplayer/metachain/command/version"
)

type RootCommand struct {
	baseCmd *cobra.Command
}

func NewRootCommand() *RootCommand {
	rootCommand := &RootCommand{
		baseCmd: &cobra.Command{
			Short: "Metachain is a framework for building Ethereum-compatible Blockchain networks",
		},
	}

	helper.RegisterJSONOutputFlag(rootCommand.baseCmd)

	rootCommand.registerSubCommands()

	return rootCommand
}

func (rc *RootCommand) registerSubCommands() {
	rc.baseCmd.AddCommand(
		version.GetCommand(),
		txpool.GetCommand(),
		status.GetCommand(),
		secrets.GetCommand(),
		peers.GetCommand(),
		rootchain.GetCommand(),
		monitor.GetCommand(),
		backup.GetCommand(),
		genesis.GetCommand(),
		server.GetCommand(),
		license.GetCommand(),
		mbftsecrets.GetCommand(),
		mbft.GetCommand(),
		bridge.GetCommand(),
		regenesis.GetCommand(),
	)
}

func (rc *RootCommand) Execute() {
	if err := rc.baseCmd.Execute(); err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)

		os.Exit(1)
	}
}
