package deploy

import (
	"context"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/umbracle/ethgo"
	"github.com/umbracle/ethgo/jsonrpc"
	"github.com/umbracle/ethgo/testutil"

	"github.com/DAO-Metaplayer/metachain/command"
	"github.com/DAO-Metaplayer/metachain/command/rootchain/helper"
	"github.com/DAO-Metaplayer/metachain/consensus/mbft"
	"github.com/DAO-Metaplayer/metachain/consensus/mbft/contractsapi"
	"github.com/DAO-Metaplayer/metachain/consensus/mbft/validator"
	"github.com/DAO-Metaplayer/metachain/types"
)

func TestDeployContracts_NoPanics(t *testing.T) {
	t.Parallel()

	server := testutil.DeployTestServer(t, nil)
	t.Cleanup(func() {
		if err := os.RemoveAll(params.genesisPath); err != nil {
			t.Fatal(err)
		}
	})

	client, err := jsonrpc.NewClient(server.HTTPAddr())
	require.NoError(t, err)

	testKey, err := helper.DecodePrivateKey("")
	require.NoError(t, err)

	receipt, err := server.Fund(testKey.Address())
	require.NoError(t, err)
	require.Equal(t, uint64(types.ReceiptSuccess), receipt.Status)

	txn := &ethgo.Transaction{
		To:    nil, // contract deployment
		Input: contractsapi.StakeManager.Bytecode,
	}

	receipt, err = server.SendTxn(txn)
	require.NoError(t, err)
	require.Equal(t, uint64(types.ReceiptSuccess), receipt.Status)

	outputter := command.InitializeOutputter(GetCommand())
	params.stakeManagerAddr = receipt.ContractAddress.String()
	params.stakeTokenAddr = types.StringToAddress("0x123456789").String()
	params.proxyContractsAdmin = "0x5aaeb6053f3e94c9b9a09f33669435e7ef1beaed"
	consensusCfg = mbft.MBFTConfig{
		NativeTokenConfig: &mbft.TokenConfig{
			Name:       "Test",
			Symbol:     "TST",
			Decimals:   18,
			IsMintable: false,
		},
	}

	require.NotPanics(t, func() {
		_, err = deployContracts(outputter, client, 1, []*validator.GenesisValidator{}, context.Background())
	})
	require.NoError(t, err)
}
