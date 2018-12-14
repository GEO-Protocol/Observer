package requests

import "geo-observers-blockchain/core/utils"

type request struct {
	observerNumber uint16
}

func (r *request) ObserverNumber() uint16 {
	return r.observerNumber
}

func (r *request) SetObserverNumber(number uint16) {
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
	SetObserverNumber(number uint16)
	ObserverNumber() uint16
	MarshalBinary() ([]byte, error)
	UnmarshalBinary([]byte) error
}
