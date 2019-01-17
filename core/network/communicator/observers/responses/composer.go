package responses

import (
	"geo-observers-blockchain/core/common/types/hash"
	"geo-observers-blockchain/core/network/communicator/observers/requests"
	"geo-observers-blockchain/core/utils"
)

type ChainTop struct {
	*response

	TopBlockHash       *hash.SHA256Container
	RequestedBlockHash *hash.SHA256Container
}

func NewChainTop(r requests.Request, observerNumber uint16,
	topBlockHash, requestedBlockHash *hash.SHA256Container) *ChainTop {
	return &ChainTop{
		response:           newResponse(r, observerNumber),
		TopBlockHash:       topBlockHash,
		RequestedBlockHash: requestedBlockHash,
	}
}

func (r *ChainTop) MarshalBinary() (data []byte, err error) {
	responseData, err := r.response.MarshalBinary()
	if err != nil {
		return
	}

	topBlockHashData, err := r.TopBlockHash.MarshalBinary()
	if err != nil {
		return
	}

	requestedBlockHashData, err := r.RequestedBlockHash.MarshalBinary()
	if err != nil {
		return
	}

	return utils.ChainByteSlices(responseData, topBlockHashData, requestedBlockHashData), nil
}

func (r *ChainTop) UnmarshalBinary(data []byte) (err error) {
	r.response = &response{}
	err = r.response.UnmarshalBinary(data[:2])
	if err != nil {
		return
	}

	r.TopBlockHash = &hash.SHA256Container{}
	err = r.TopBlockHash.UnmarshalBinary(data[2:])
	if err != nil {
		return
	}

	r.RequestedBlockHash = &hash.SHA256Container{}
	return r.RequestedBlockHash.UnmarshalBinary(data[2+hash.BytesSize:])
}
