package responses

import (
	"geo-observers-blockchain/core/common"
	"geo-observers-blockchain/core/common/errors"
	"geo-observers-blockchain/core/utils"
)

type LastBlockHeight struct {
	Height uint64
}

func (response *LastBlockHeight) MarshalBinary() (data []byte, err error) {
	return utils.MarshalUint64(response.Height), nil
}

func (response *LastBlockHeight) UnmarshalBinary(data []byte) (err error) {
	if len(data) < common.Uint64ByteSize {
		return errors.InvalidDataFormat
	}

	response.Height, err = utils.UnmarshalUint64(data)
	return
}
