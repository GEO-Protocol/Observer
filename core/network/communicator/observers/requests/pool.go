package requests

import (
	"geo-observers-blockchain/core/common/errors"
	"geo-observers-blockchain/core/geo"
	"geo-observers-blockchain/core/network/communicator/observers/constants"
	"geo-observers-blockchain/core/utils"
)

type PoolInstanceBroadcast struct {
	request

	Instance interface{}
}

func NewPoolInstanceBroadcast(
	destinationObservers []uint16, instance interface{}) *PoolInstanceBroadcast {
	return &PoolInstanceBroadcast{
		request:  newRequest(destinationObservers),
		Instance: instance,
	}
}

func (r *PoolInstanceBroadcast) MarshalBinary() (data []byte, err error) {
	var (
		streamType   []byte
		instanceData []byte
	)

	switch r.Instance.(type) {
	case *geo.TSL:
		{
			streamType = constants.StreamTypeRequestTSLBroadcast
			instanceData, err = r.Instance.(*geo.TSL).MarshalBinary()
		}

	case *geo.Claim:
		{
			streamType = constants.StreamTypeRequestClaimBroadcast
			instanceData, err = r.Instance.(*geo.Claim).MarshalBinary()
		}

	default:
		err = errors.InvalidDataFormat
	}

	if err != nil {
		return
	}

	requestData, err := r.request.MarshalBinary()
	if err != nil {
		return
	}

	data = utils.ChainByteSlices(streamType, requestData, instanceData)
	return
}

func (r *PoolInstanceBroadcast) UnmarshalBinary(data []byte) (err error) {
	err = r.request.UnmarshalBinary(data[1:3])
	if err != nil {
		return
	}

	switch data[0] {
	case constants.DataTypeRequestTSLBroadcast:
		{
			r.Instance = geo.NewTSL()
			err = r.Instance.(*geo.TSL).UnmarshalBinary(data[3:])
		}

	case constants.DataTypeRequestClaimBroadcast:
		{
			r.Instance = geo.NewClaim()
			err = r.Instance.(*geo.Claim).UnmarshalBinary(data[3:])
		}

	default:
		err = errors.InvalidDataFormat
	}

	return
}
