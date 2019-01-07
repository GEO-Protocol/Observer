package responses

import (
	"geo-observers-blockchain/core/common/types/hash"
	"geo-observers-blockchain/core/network/communicator/observers/requests"
	"geo-observers-blockchain/core/utils"
)

type PoolInstanceBroadcastApprove struct {
	*response

	Hash *hash.SHA256Container
}

func NewPoolInstanceBroadcastApprove(
	r requests.Request, observerNumber uint16, hash *hash.SHA256Container) *PoolInstanceBroadcastApprove {
	return &PoolInstanceBroadcastApprove{
		response: newResponse(r, observerNumber),
		Hash:     hash,
	}
}

func (r *PoolInstanceBroadcastApprove) MarshalBinary() (data []byte, err error) {
	responseData, err := r.response.MarshalBinary()
	if err != nil {
		return
	}

	hashData, err := r.Hash.MarshalBinary()
	if err != nil {
		return
	}

	return utils.ChainByteSlices(responseData, hashData), nil
}

func (r *PoolInstanceBroadcastApprove) UnmarshalBinary(data []byte) (err error) {
	r.response = &response{}
	err = r.response.UnmarshalBinary(data[:2])
	if err != nil {
		return
	}

	r.Hash = &hash.SHA256Container{}
	return r.Hash.UnmarshalBinary(data[2:])
}

// --------------------------------------------------------------------------------------------------------------------

type ClaimApprove struct {
	*PoolInstanceBroadcastApprove
}

func (r *ClaimApprove) MarshalBinary() (data []byte, err error) {
	return r.PoolInstanceBroadcastApprove.MarshalBinary()
}

func (r *ClaimApprove) UnmarshalBinary(data []byte) (err error) {
	r.PoolInstanceBroadcastApprove = &PoolInstanceBroadcastApprove{}
	return r.PoolInstanceBroadcastApprove.UnmarshalBinary(data)
}

// --------------------------------------------------------------------------------------------------------------------

type TSLApprove struct {
	*PoolInstanceBroadcastApprove
}

func (r *TSLApprove) MarshalBinary() (data []byte, err error) {
	return r.PoolInstanceBroadcastApprove.MarshalBinary()
}

func (r *TSLApprove) UnmarshalBinary(data []byte) (err error) {
	r.PoolInstanceBroadcastApprove = &PoolInstanceBroadcastApprove{}
	return r.PoolInstanceBroadcastApprove.UnmarshalBinary(data)
}
