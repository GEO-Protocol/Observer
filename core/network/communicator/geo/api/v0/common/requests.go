package common

import "encoding"

type Request interface {
	// Returns channel from which the response should be fetched,
	// in case if Request has paired response and client should wait for it (and keep up the connection).
	// Otherwise - returns nil (some requests to the Observer does not assumes any response).
	ResponseChannel() chan encoding.BinaryMarshaler
	UnmarshalBinary(data []byte) (err error)
}

// --------------------------------------------------------------------------------------------------------------------

type RequestWithResponse struct {
	response chan encoding.BinaryMarshaler
}

func NewRequestWithResponse() *RequestWithResponse {
	return &RequestWithResponse{
		response: make(chan encoding.BinaryMarshaler, 1),
	}
}

func (r *RequestWithResponse) ResponseChannel() chan encoding.BinaryMarshaler {
	return r.response
}

// --------------------------------------------------------------------------------------------------------------------

type RequestWithoutResponse struct{}

func (r *RequestWithoutResponse) ResponseChannel() chan encoding.BinaryMarshaler {
	return nil
}
