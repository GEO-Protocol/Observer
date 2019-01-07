package chain

import (
	"geo-observers-blockchain/core/chain/block"
	"geo-observers-blockchain/core/chain/signatures"
	"geo-observers-blockchain/core/common"
	"geo-observers-blockchain/core/geo"
	"geo-observers-blockchain/core/storage"
	"geo-observers-blockchain/core/utils"
)

var (
	ErrBlocksCollision = utils.Error("chain", "blocks collision")
)

// todo: consider dropping claims and TSLsHashes that are not needed any more.
// improve: write 1-2 bytes for empty block, now empty block contains 68B.
type Chain struct {
	storage *storage.AppendOnlyStorage
}

func NewChain() (chain *Chain, err error) {
	storageHandler, err := storage.Open("data/chain.dat")
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

func (chain *Chain) At(index uint64) (b *block.Signed, err error) {
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

func (chain *Chain) ensureGenesisBlockPresence() (err error) {
	if chain.Height() == 0 {
		b := &block.Signed{
			Body: &block.Body{
				Index:               0,
				ExternalChainHeight: 0,
				AuthorObserverIndex: 0,
				Claims:              &geo.Claims{},
				TSLs:                &geo.TransactionSignaturesLists{},
			},
			Signatures: signatures.NewIndexedObserversSignatures(common.ObserversMaxCount),
		}

		err = chain.Append(b)
	}

	return
}

//
//func (chain *Chain) resetLastBlockTimestamp() {
//	if chain.Index() == 0 {
//		chain.LastBlockTimestamp = time.Now()
//
//	} else {
//		// todo: read from last block
//	}
//}
