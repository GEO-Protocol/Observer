package chain

import "time"

// todo: consider dropping claims and TSLs that are not needed any more.
type Chain struct {
	// Time when last block was generated/inserted.
	LastBlockTimestamp time.Time
}

func NewChain() *Chain {
	chain := &Chain{}

	chain.loadFromDisk()
	chain.resetLastBlockTimestamp()
	return chain
}

// todo: implement
func (c *Chain) Height() uint64 {
	return 0
}

func (c *Chain) TryAppend(block *BlockProposal) error {
	return nil
}

func (c *Chain) validate(block *BlockProposal) error {
	return nil
}

func (c *Chain) loadFromDisk() {

}

func (c *Chain) resetLastBlockTimestamp() {
	if c.Height() == 0 {
		c.LastBlockTimestamp = time.Now()

	} else {
		// todo: read from last block
	}
}
