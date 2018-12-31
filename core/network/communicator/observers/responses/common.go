package responses

import (
	"geo-observers-blockchain/core/network/communicator/observers/requests"
	"geo-observers-blockchain/core/utils"
)

type Response interface {
	MarshalBinary() ([]byte, error)
	UnmarshalBinary([]byte) error
	Request() requests.Request
	ObserverNumber() uint16
}

type response struct {
	request        requests.Request
	observerNumber uint16
}

func newResponse(req requests.Request, observerNumber uint16) *response {
	return &response{
		request:        req,
		observerNumber: observerNumber,
	}
}

func (r *response) Request() requests.Request {
	return r.request
}

func (r *response) ObserverNumber() uint16 {
	return r.observerNumber
}

func (r *response) MarshalBinary() ([]byte, error) {
	return utils.MarshalUint16(r.observerNumber), nil
}

func (r *response) UnmarshalBinary(data []byte) (err error) {
	r.observerNumber, err = utils.UnmarshalUint16(data[:2])
	return
}
