package requests

import (
	"geo-observers-blockchain/core/common"
	"geo-observers-blockchain/core/utils"
)

// ChainTop is used to request other observers hash of the last block in their chain,
// along with hash of the block, that is supposed to be highest (from current observer's point of view).
type ChainTop struct {
	request

	LastBlockIndex uint64
}

func NewChainTop(currentChainHeight uint64) *ChainTop {
	return &ChainTop{
		request: newRequest(nil),

		LastBlockIndex: currentChainHeight,
	}
}

func (r *ChainTop) MarshalBinary() (data []byte, err error) {
	requestBinary, err := r.request.MarshalBinary()
	if err != nil {
		return
	}

	lastBlockIndexBinary := utils.MarshalUint64(r.LastBlockIndex)
	return utils.ChainByteSlices(requestBinary, lastBlockIndexBinary), nil
}

func (r *ChainTop) UnmarshalBinary(data []byte) (err error) {
	err = r.request.UnmarshalBinary(data[0:common.Uint16ByteSize])
	if err != nil {
		return
	}

	r.LastBlockIndex, err = utils.UnmarshalUint64(data[common.Uint16ByteSize:])
	return
}
