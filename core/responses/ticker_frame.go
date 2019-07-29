package responses

import (
	"geo-observers-blockchain/core/common"
	"geo-observers-blockchain/core/requests"
	"geo-observers-blockchain/core/utils"
	"time"
)

type TimeFrame struct {
	*response

	FrameIndex      uint16
	NanosecondsLeft uint64
	Received        time.Time

	// todo: add observers configuration hash
	// todo: add observer signature to prevent data obfuscation
}

func NewTimeFrame(r requests.Request, observerIndex, index uint16, nanosecondsLeft uint64) *TimeFrame {
	return &TimeFrame{
		response:        newResponse(r, observerIndex),
		FrameIndex:      index,
		NanosecondsLeft: nanosecondsLeft,
	}
}

func (r *TimeFrame) Request() requests.Request {
	return r.request
}

func (r *TimeFrame) MarshalBinary() (data []byte, err error) {
	responseData, err := r.response.MarshalBinary()
	if err != nil {
		return
	}

	return utils.ChainByteSlices(
		responseData,
		utils.MarshalUint16(r.FrameIndex),
		utils.MarshalUint64(r.NanosecondsLeft)), nil
}

func (r *TimeFrame) UnmarshalBinary(data []byte) (err error) {
	r.response = &response{}
	err = r.response.UnmarshalBinary(data[:ResponseDefaultBytesLength])
	if err != nil {
		return
	}

	r.FrameIndex, err = utils.UnmarshalUint16(data[ResponseDefaultBytesLength : ResponseDefaultBytesLength+2])
	if err != nil {
		return
	}

	const nanosecondsOffset = ResponseDefaultBytesLength + common.Uint16ByteSize
	r.NanosecondsLeft, err = utils.UnmarshalUint64(data[nanosecondsOffset : nanosecondsOffset+common.Uint64ByteSize])
	if err != nil {
		return
	}

	return nil
}
