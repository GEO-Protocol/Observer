package chain

import (
	"geo-observers-blockchain/core/chain/block"
	"geo-observers-blockchain/core/chain/signatures"
	"geo-observers-blockchain/core/common"
	"geo-observers-blockchain/core/common/errors"
	"geo-observers-blockchain/core/geo"
	"geo-observers-blockchain/core/storage"
	"geo-observers-blockchain/core/utils"
)

const (
	DataPath     = "data/"
	DataFilePath = DataPath + "chain.dat"
)

var (
	// todo: move it to the common errors
	ErrBlocksCollision = utils.Error("chain", "blocks collision")
)

// todo: consider dropping claims and TSLsHashes that are not needed any more.
// improve: write 1-2 bytes for empty block, now empty block contains 68B.
type Chain struct {
	storage *storage.AppendOnlyStorage
}

func NewChain(datFilePath string) (chain *Chain, err error) {
	storageHandler, err := storage.Open(datFilePath)
	if err != nil {
		return
	}

	chain = &Chain{storage: storageHandler}
	err = chain.ensureGenesisBlockPresence()
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

func (chain *Chain) Append(bs *block.Signed) (err error) {
	// todo: add hashes check for collision detection

	if chain.storage.HasIndex(bs.Body.Index) {
		return ErrBlocksCollision
	}

	data, err := bs.MarshalBinary()
	if err != nil {
		return
	}

	return chain.storage.Append(data[:])
}

func (chain *Chain) BlockAt(index uint64) (b *block.Signed, err error) {
	data, err := chain.storage.Get(index)
	if err != nil {
		return
	}

	b = &block.Signed{}
	err = b.UnmarshalBinary(data)
	if err != nil {
		return
	}

	return
}

// LastBlock returns highest block of the chain.
// todo: tests needed.
func (chain *Chain) LastBlock() (b *block.Signed, e errors.E) {
	topIndex := chain.Height() - 1
	if topIndex < 0 {
		e = errors.AppendStackTrace(errors.InvalidChainHeight)
		return
	}

	b, err := chain.BlockAt(topIndex)
	if err != nil {
		e = errors.AppendStackTrace(err)
		return
	}

	return
}

func (chain *Chain) GenerateGenesisBlock() (b *block.Signed) {
	return &block.Signed{
		Body: &block.Body{
			Index:               0,
			ExternalChainHeight: 0,
			AuthorObserverIndex: 0,
			Claims:              &geo.Claims{},
			TSLs:                &geo.TSLs{},
		},
		Signatures: signatures.NewIndexedObserversSignatures(common.ObserversMaxCount),
	}
}

func (chain *Chain) ensureGenesisBlockPresence() (err error) {
	if chain.Height() == 0 {
		err = chain.Append(
			chain.GenerateGenesisBlock())
	}
	return
}
