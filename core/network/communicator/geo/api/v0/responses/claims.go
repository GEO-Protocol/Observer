package responses

import (
	"geo-observers-blockchain/core/common"
	"geo-observers-blockchain/core/common/errors"
	"geo-observers-blockchain/core/utils"
)

type ClaimIsPresent struct {
	PresentInPool  bool
	PresentInBlock uint64
}

func (response *ClaimIsPresent) MarshalBinary() (data []byte, err error) {
	data = make([]byte, 1)
	if response.PresentInPool {
		data[0] = 1
	} else {
		data[0] = 0
	}

	presentInBlockBinary := utils.MarshalUint64(response.PresentInBlock)
	return utils.ChainByteSlices(data, presentInBlockBinary), nil
}

func (response *ClaimIsPresent) UnmarshalBinary(data []byte) (err error) {
	if len(data) < common.Uint64ByteSize+1 {
		return errors.InvalidDataFormat
	}

	response.PresentInPool = true
	if data[0] == 0 {
		response.PresentInPool = false
	}

	response.PresentInBlock, err = utils.UnmarshalUint64(data[1:])
	return
}
