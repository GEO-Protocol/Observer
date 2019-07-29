package responses

import (
	"geo-observers-blockchain/core/requests"
	"geo-observers-blockchain/core/utils"
)

const ResponseDefaultBytesLength = 4

type Response interface {
	MarshalBinary() ([]byte, error)
	UnmarshalBinary([]byte) error
	Request() requests.Request
	ObserverIndex() uint16

	SetSenderIndex(index uint16)
	SenderIndex() uint16
}

type response struct {
	request       requests.Request
	receiverIndex uint16
	senderIndex   uint16
}

func newResponse(req requests.Request, observerNumber uint16) *response {
	return &response{
		request:       req,
		receiverIndex: observerNumber,
	}
}

func (r *response) Request() requests.Request {
	return r.request
}

func (r *response) ObserverIndex() uint16 {
	return r.receiverIndex
}

func (r *response) SenderIndex() uint16 {
	return r.senderIndex
}

func (r *response) SetSenderIndex(index uint16) {
	r.senderIndex = index
}

func (r *response) MarshalBinary() ([]byte, error) {
	return utils.ChainByteSlices(
		utils.MarshalUint16(r.senderIndex),
		utils.MarshalUint16(r.receiverIndex)), nil
}

func (r *response) UnmarshalBinary(data []byte) (err error) {
	r.senderIndex, err = utils.UnmarshalUint16(data[:2])
	if err != nil {
		return
	}

	r.receiverIndex, err = utils.UnmarshalUint16(data[2:4])
	return
}
