package responses

import (
	"geo-observers-blockchain/core/network/communicator/observers/requests"
	"geo-observers-blockchain/core/utils"
	"time"
)

type ResponseTimeFrame struct {
	*response

	FrameIndex      uint16
	NanosecondsLeft uint64
	Received        time.Time

	// todo: add observers configuration hash
	// todo: add observer signature to prevent data obfuscation
}

func NewResponseTimeFrame(r requests.Request, observerIndex, index uint16, nanosecondsLeft uint64) *ResponseTimeFrame {
	return &ResponseTimeFrame{
		response:        newResponse(r, observerIndex),
		FrameIndex:      index,
		NanosecondsLeft: nanosecondsLeft,
	}
}

func (r *ResponseTimeFrame) Request() requests.Request {
	return r.request
}

func (r *ResponseTimeFrame) MarshalBinary() ([]byte, error) {
	return utils.ChainByteSlices(
		utils.MarshalUint16(r.FrameIndex),
		utils.MarshalUint64(r.NanosecondsLeft)), nil
}

func (r *ResponseTimeFrame) UnmarshalBinary(data []byte) (err error) {
	r.FrameIndex, err = utils.UnmarshalUint16(data[0:2])
	if err != nil {
		return
	}

	r.NanosecondsLeft, err = utils.UnmarshalUint64(data[2:10])
	if err != nil {
		return
	}

	return nil
}
