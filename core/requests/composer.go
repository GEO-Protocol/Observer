package requests

import (
	"geo-observers-blockchain/core/utils"
)

type ChainInfo struct {
	request

	// Index of last block of current chain.
	LastBlockIndex uint64
}

func NewChainInfoRequest(currentChainHeight uint64) *ChainInfo {
	return &ChainInfo{
		request:        newRequestForAllObservers(),
		LastBlockIndex: currentChainHeight,
	}
}

func (r *ChainInfo) MarshalBinary() (data []byte, err error) {
	requestBinary, err := r.request.MarshalBinary()
	if err != nil {
		return
	}

	lastBlockIndexBinary := utils.MarshalUint64(r.LastBlockIndex)
	return utils.ChainByteSlices(requestBinary, lastBlockIndexBinary), nil
}

func (r *ChainInfo) UnmarshalBinary(data []byte) (err error) {
	err = r.request.UnmarshalBinary(data[0:RequestDefaultBytesLength])
	if err != nil {
		return
	}

	r.LastBlockIndex, err = utils.UnmarshalUint64(data[RequestDefaultBytesLength:])
	return
}
