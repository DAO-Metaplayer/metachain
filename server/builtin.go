package server

import (
	"github.com/DAO-Metaplayer/metachain/chain"
	"github.com/DAO-Metaplayer/metachain/consensus"
	consensusMBFT "github.com/DAO-Metaplayer/metachain/consensus/mbft"
	"github.com/DAO-Metaplayer/metachain/forkmanager"
	"github.com/DAO-Metaplayer/metachain/secrets"
	"github.com/DAO-Metaplayer/metachain/secrets/awsssm"
	"github.com/DAO-Metaplayer/metachain/secrets/gcpssm"
	"github.com/DAO-Metaplayer/metachain/secrets/hashicorpvault"
	"github.com/DAO-Metaplayer/metachain/secrets/local"
	"github.com/DAO-Metaplayer/metachain/state"
)

type GenesisFactoryHook func(config *chain.Chain, engineName string) func(*state.Transition) error

type ConsensusType string

type ForkManagerFactory func(forks *chain.Forks) error

type ForkManagerInitialParamsFactory func(config *chain.Chain) (*forkmanager.ForkParams, error)

const (
	METABFTConsensus ConsensusType = "metabft"
	MBFTConsensus    ConsensusType = consensusMBFT.ConsensusName
)

var consensusBackends = map[ConsensusType]consensus.Factory{
	MBFTConsensus: consensusMBFT.Factory,
}

// secretsManagerBackends defines the SecretManager factories for different
// secret management solutions
var secretsManagerBackends = map[secrets.SecretsManagerType]secrets.SecretsManagerFactory{
	secrets.Local:          local.SecretsManagerFactory,
	secrets.HashicorpVault: hashicorpvault.SecretsManagerFactory,
	secrets.AWSSSM:         awsssm.SecretsManagerFactory,
	secrets.GCPSSM:         gcpssm.SecretsManagerFactory,
}

var genesisCreationFactory = map[ConsensusType]GenesisFactoryHook{
	MBFTConsensus: consensusMBFT.GenesisPostHookFactory,
}

var forkManagerFactory = map[ConsensusType]ForkManagerFactory{
	MBFTConsensus: consensusMBFT.ForkManagerFactory,
}

var forkManagerInitialParamsFactory = map[ConsensusType]ForkManagerInitialParamsFactory{
	MBFTConsensus: consensusMBFT.ForkManagerInitialParamsFactory,
}

func ConsensusSupported(value string) bool {
	_, ok := consensusBackends[ConsensusType(value)]

	return ok
}
