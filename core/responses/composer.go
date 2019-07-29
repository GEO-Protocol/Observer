package responses

import (
	"geo-observers-blockchain/core/common/types/hash"
	"geo-observers-blockchain/core/requests"
	"geo-observers-blockchain/core/utils"
)

type ChainInfo struct {
	*response

	LastBlockHash      *hash.SHA256Container
	RequestedBlockHash *hash.SHA256Container
}

func NewChainTop(r requests.Request, observerNumber uint16,
	topBlockHash, requestedBlockHash *hash.SHA256Container) *ChainInfo {
	return &ChainInfo{
		response:           newResponse(r, observerNumber),
		LastBlockHash:      topBlockHash,
		RequestedBlockHash: requestedBlockHash,
	}
}

func (r *ChainInfo) MarshalBinary() (data []byte, err error) {
	responseData, err := r.response.MarshalBinary()
	if err != nil {
		return
	}

	topBlockHashData, err := r.LastBlockHash.MarshalBinary()
	if err != nil {
		return
	}

	requestedBlockHashData, err := r.RequestedBlockHash.MarshalBinary()
	if err != nil {
		return
	}

	return utils.ChainByteSlices(responseData, topBlockHashData, requestedBlockHashData), nil
}

func (r *ChainInfo) UnmarshalBinary(data []byte) (err error) {
	r.response = &response{}
	err = r.response.UnmarshalBinary(data[:ResponseDefaultBytesLength])
	if err != nil {
		return
	}

	r.LastBlockHash = &hash.SHA256Container{}
	err = r.LastBlockHash.UnmarshalBinary(data[ResponseDefaultBytesLength:])
	if err != nil {
		return
	}

	r.RequestedBlockHash = &hash.SHA256Container{}
	return r.RequestedBlockHash.UnmarshalBinary(data[ResponseDefaultBytesLength+hash.BytesSize:])
}
