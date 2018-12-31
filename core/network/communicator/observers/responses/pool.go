package responses

import (
	"geo-observers-blockchain/core/common/types/hash"
	"geo-observers-blockchain/core/network/communicator/observers/requests"
	"geo-observers-blockchain/core/utils"
)

type ResponsePoolInstanceBroadcastApprove struct {
	*response

	Hash *hash.SHA256Container
}

func NewResponsePoolInstanceBroadcastApprove(
	r requests.Request, observerNumber uint16, hash *hash.SHA256Container) *ResponsePoolInstanceBroadcastApprove {
	return &ResponsePoolInstanceBroadcastApprove{
		response: newResponse(r, observerNumber),
		Hash:     hash,
	}
}

func (r *ResponsePoolInstanceBroadcastApprove) MarshalBinary() (data []byte, err error) {
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

func (r *ResponsePoolInstanceBroadcastApprove) UnmarshalBinary(data []byte) (err error) {
	r.response = &response{}
	err = r.response.UnmarshalBinary(data[:2])
	if err != nil {
		return
	}

	r.Hash = &hash.SHA256Container{}
	return r.Hash.UnmarshalBinary(data[2:])
}

// --------------------------------------------------------------------------------------------------------------------

type ResponseClaimApprove struct {
	*ResponsePoolInstanceBroadcastApprove
}

func (r *ResponseClaimApprove) MarshalBinary() (data []byte, err error) {
	return r.ResponsePoolInstanceBroadcastApprove.MarshalBinary()
}

func (r *ResponseClaimApprove) UnmarshalBinary(data []byte) (err error) {
	r.ResponsePoolInstanceBroadcastApprove = &ResponsePoolInstanceBroadcastApprove{}
	return r.ResponsePoolInstanceBroadcastApprove.UnmarshalBinary(data)
}

// --------------------------------------------------------------------------------------------------------------------

type ResponseTSLApprove struct {
	*ResponsePoolInstanceBroadcastApprove
}

func (r *ResponseTSLApprove) MarshalBinary() (data []byte, err error) {
	return r.ResponsePoolInstanceBroadcastApprove.MarshalBinary()
}

func (r *ResponseTSLApprove) UnmarshalBinary(data []byte) (err error) {
	r.ResponsePoolInstanceBroadcastApprove = &ResponsePoolInstanceBroadcastApprove{}
	return r.ResponsePoolInstanceBroadcastApprove.UnmarshalBinary(data)
}
