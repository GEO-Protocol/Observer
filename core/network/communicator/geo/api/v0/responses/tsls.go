package responses

import (
	"geo-observers-blockchain/core/common"
	"geo-observers-blockchain/core/common/errors"
	"geo-observers-blockchain/core/geo"
	"geo-observers-blockchain/core/utils"
)

type TSLIsPresent struct {
	PresentInPool  bool
	PresentInBlock uint64
}

func (response *TSLIsPresent) MarshalBinary() (data []byte, err error) {
	data = make([]byte, 1)
	if response.PresentInPool {
		data[0] = 1
	} else {
		data[0] = 0
	}

	presentInBlockBinary := utils.MarshalUint64(response.PresentInBlock)
	return utils.ChainByteSlices(data, presentInBlockBinary), nil
}

func (response *TSLIsPresent) UnmarshalBinary(data []byte) (err error) {
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

// --------------------------------------------------------------------------------------------------------------------

const (
	TSLGetMinBinarySize = geo.TSLMinBinarySize + 1
)

type TSLGet struct {
	IsPresent bool
	TSL       *geo.TSL
}

func (response *TSLGet) MarshalBinary() (data []byte, err error) {
	data = make([]byte, 1)
	if response.IsPresent {
		data[0] = 1
	} else {
		data[0] = 0
	}

	tslBinary, err := response.TSL.MarshalBinary()
	if err != nil {
		return
	}

	return utils.ChainByteSlices(data, tslBinary), nil
}

func (response *TSLGet) UnmarshalBinary(data []byte) (err error) {
	if len(data) < TSLGetMinBinarySize {
		return errors.InvalidDataFormat
	}

	response.IsPresent = true
	if data[0] == 0 {
		response.IsPresent = false
	}

	response.TSL = geo.NewTSL()
	err = response.TSL.UnmarshalBinary(data[1:])
	return
}
