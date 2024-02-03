package mbft

import (
	"context"

	"github.com/DAO-Metaplayer/metabft/core"
)

// METABFTConsensusWrapper is a convenience wrapper for the metabft package
type METABFTConsensusWrapper struct {
	*core.METABFT
}

func newMETABFTConsensusWrapper(
	logger core.Logger,
	backend core.Backend,
	transport core.Transport,
) *METABFTConsensusWrapper {
	return &METABFTConsensusWrapper{
		METABFT: core.NewMETABFT(logger, backend, transport),
	}
}

// runSequence starts the underlying consensus mechanism for the given height.
// It may be called by a single thread at any given time
// It returns channel which will be closed after c.METABFT.RunSequence is done
// and stopSequence function which can be used to halt c.METABFT.RunSequence routine from outside
func (c *METABFTConsensusWrapper) runSequence(height uint64) (<-chan struct{}, func()) {
	sequenceDone := make(chan struct{})
	ctx, cancelSequence := context.WithCancel(context.Background())

	go func() {
		c.METABFT.RunSequence(ctx, height)
		cancelSequence()
		close(sequenceDone)
	}()

	return sequenceDone, func() {
		// stopSequence terminates the running METABFT sequence gracefully and waits for it to return
		cancelSequence()
		<-sequenceDone // waits until c.METABFT.RunSequenc routine finishes
	}
}
