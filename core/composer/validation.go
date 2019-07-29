package composer

import (
	"geo-observers-blockchain/core/chain/chain"
	"geo-observers-blockchain/core/common/errors"
	"geo-observers-blockchain/core/common/types/hash"
)

func (c *Composer) validateFullyRewrittenChain(tmpCopyFilename string) (e errors.E) {
	chainCandidate, err := chain.NewChain(tmpCopyFilename)
	if err != nil {
		e = errors.AppendStackTrace(err)
		return
	}

	return c.checkBlocksSequence(chainCandidate, 1)
}

func (c *Composer) validateIncrementallySynchronisedChain(
	tmpCopyFilename string, currentChain *chain.Chain) (e errors.E) {

	chainCandidate, err := chain.NewChain(tmpCopyFilename)
	if err != nil {
		e = errors.AppendStackTrace(err)
		return
	}

	return c.checkBlocksSequence(chainCandidate, currentChain.Height())
}

func (c *Composer) validateLastBlock(
	tmpCopyFilename string, currentChain *chain.Chain) (e errors.E) {

	chainCandidate, err := chain.NewChain(tmpCopyFilename)
	if err != nil {
		e = errors.AppendStackTrace(err)
		return
	}

	lastBlock, e := chainCandidate.LastBlock()
	if e != nil {
		return
	}

	if lastBlock.Body.Index == 0 {
		// Chain contains no blocks.
		// Genesis block should not be validated.
		return
	}

	return c.checkBlocksSequence(chainCandidate, lastBlock.Body.Index)
}

func (c *Composer) checkBlocksSequence(candidate *chain.Chain, offset uint64) (e errors.E) {
	var previousBlockHash hash.SHA256Container
	if offset == 0 {
		e = errors.AppendStackTrace(errors.InvalidParameter)
		return

	} else if offset == 1 {
		previousBlockHash = candidate.GenerateGenesisBlock().Body.Hash

	} else {
		previousBlock, err := candidate.BlockAt(offset - 1)
		if err != nil {
			e = errors.AppendStackTrace(errors.ValidationFailed)
			return
		}

		previousBlockHash = previousBlock.Body.Hash
	}

	totalBlocksCount := candidate.Height()
	for ; offset < totalBlocksCount; offset++ {
		nextBlock, err := candidate.BlockAt(offset)
		if err != nil {
			e = errors.AppendStackTrace(errors.ValidationFailed)
			return
		}

		reportedHash := nextBlock.Body.Hash

		e = nextBlock.Body.SortInternalSequences()
		if e != nil {
			return
		}

		e = nextBlock.Body.UpdateHash(previousBlockHash)
		if e != nil {
			return
		}

		if nextBlock.Body.Hash.Compare(&reportedHash) == false {
			e = errors.AppendStackTrace(errors.ValidationFailed)
		}

		previousBlockHash = nextBlock.Body.Hash
	}

	return
}
