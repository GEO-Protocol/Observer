package requests

import (
	"geo-observers-blockchain/core/settings"
	"geo-observers-blockchain/core/utils"
)

const RequestDefaultBytesLength = 4

type request struct {
	senderIndex   uint16
	receiverIndex uint16

	// If nil - request must be sent to all observers.
	// Otherwise - contains positional number ob observers,
	// that should receive the request.
	destinationObservers []uint16
}

func newRequest(destinationObservers []uint16) request {
	if len(destinationObservers) == settings.ObserversMaxCount {
		return request{}
	}

	return request{
		destinationObservers: destinationObservers,
	}
}

func newRequestForAllObservers() request {
	return request{}
}

func (r *request) ObserverIndex() uint16 {
	return r.receiverIndex
}

func (r *request) SetObserverIndex(index uint16) {
	r.receiverIndex = index
}

func (r *request) SenderIndex() uint16 {
	return r.senderIndex
}

func (r *request) SetSenderIndex(index uint16) {
	r.senderIndex = index
}

func (r *request) DestinationObservers() []uint16 {
	return r.destinationObservers
}

func (r *request) MarshalBinary() ([]byte, error) {
	return utils.ChainByteSlices(
		utils.MarshalUint16(r.senderIndex),
		utils.MarshalUint16(r.receiverIndex)), nil
}

func (r *request) UnmarshalBinary(data []byte) (err error) {
	r.senderIndex, err = utils.UnmarshalUint16(data[:2])
	if err != nil {
		return
	}

	r.receiverIndex, err = utils.UnmarshalUint16(data[2:4])
	return
}

// --------------------------------------------------------------------------------------------------------------------

type Request interface {
	SetObserverIndex(number uint16)
	ObserverIndex() uint16
	SenderIndex() uint16
	SetSenderIndex(index uint16)
	MarshalBinary() ([]byte, error)
	UnmarshalBinary([]byte) error
}
