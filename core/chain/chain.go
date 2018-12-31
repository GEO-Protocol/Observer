package chain

import (
	"geo-observers-blockchain/core/storage"
	"geo-observers-blockchain/core/utils"
	"time"
)

var (
	ErrBlocksCollision = utils.Error("chain", "blocks collision")
)

// todo: consider dropping claims and TSLsHashes that are not needed any more.
type Chain struct {
	// Time when last block was generated/inserted.
	// todo: move this into block info
	//  each one block must contains date time of it creation
	LastBlockTimestamp time.Time

	storage *storage.AppendOnlyStorage
}

func NewChain() (chain *Chain, err error) {
	storageHandler, err := storage.Open("data/chain.dat")
	if err != nil {
		return
	}

	chain = &Chain{storage: storageHandler}
	chain.resetLastBlockTimestamp()
	return
}

func (chain *Chain) Height() uint64 {
	recordsCount, err := chain.storage.Count()
	if err != nil {
		// improve: return error as well
		return 0
	}

	return recordsCount
}

func (chain *Chain) Append(bs *BlockSigned) (err error) {
	// todo: add hashes check for collision detection

	if chain.storage.HasIndex(bs.Data.Height) {
		return ErrBlocksCollision
	}

	data, err := bs.MarshalBinary()
	if err != nil {
		return
	}

	return chain.storage.Append(data[:])
}

// todo: use this method instead of method in block producer
func (chain *Chain) Validate(block *ProposedBlockData) error {
	return nil
}

func (chain *Chain) resetLastBlockTimestamp() {
	if chain.Height() == 0 {
		chain.LastBlockTimestamp = time.Now()

	} else {
		// todo: read from last block
	}
}
