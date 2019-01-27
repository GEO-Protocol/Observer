package requests

import "geo-observers-blockchain/core/network/communicator/geo/api/v0/common"

type LastBlockNumber struct {
	*common.RequestWithResponse
}

func (request *LastBlockNumber) MarshalBinary() (data []byte, err error) {
	typeID := []byte{common.ReqChainLastBlockNumber}
	return typeID, nil
}

func (request *LastBlockNumber) UnmarshalBinary(data []byte) (err error) {
	request.RequestWithResponse = common.NewRequestWithResponse()
	return
}
