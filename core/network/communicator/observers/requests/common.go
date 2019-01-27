package requests

import (
	"geo-observers-blockchain/core/settings"
	"geo-observers-blockchain/core/utils"
)

type request struct {
	observerNumber uint16

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

func (r *request) ObserverIndex() uint16 {
	return r.observerNumber
}

func (r *request) DestinationObservers() []uint16 {
	return r.destinationObservers
}

func (r *request) SetObserverIndex(number uint16) {
	r.observerNumber = number
}

func (r *request) MarshalBinary() ([]byte, error) {
	return utils.MarshalUint16(r.observerNumber), nil
}

func (r *request) UnmarshalBinary(data []byte) (err error) {
	r.observerNumber, err = utils.UnmarshalUint16(data[0:2])
	return
}

// --------------------------------------------------------------------------------------------------------------------

type Request interface {
	SetObserverIndex(number uint16)
	ObserverIndex() uint16
	MarshalBinary() ([]byte, error)
	UnmarshalBinary([]byte) error
}
