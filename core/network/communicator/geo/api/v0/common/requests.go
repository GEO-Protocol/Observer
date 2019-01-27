package common

import "encoding"

type Request interface {
	// Returns channel from which the response should be fetched,
	// in case if Request has paired response and client should wait for it (and keep up the connection).
	// Otherwise - returns nil (some requests to the Observer does not assumes any response).
	ResponseChannel() chan encoding.BinaryMarshaler
	ErrorsChannel() chan error
	UnmarshalBinary(data []byte) (err error)
}

// --------------------------------------------------------------------------------------------------------------------

type RequestWithResponse struct {
	response chan encoding.BinaryMarshaler
	errors   chan error
}

func NewRequestWithResponse() *RequestWithResponse {
	return &RequestWithResponse{
		response: make(chan encoding.BinaryMarshaler, 1),
	}
}

func (r *RequestWithResponse) ResponseChannel() chan encoding.BinaryMarshaler {
	return r.response
}

func (r *RequestWithResponse) ErrorsChannel() chan error {
	return r.errors
}

// --------------------------------------------------------------------------------------------------------------------

type RequestWithoutResponse struct{}

func (r *RequestWithoutResponse) ResponseChannel() chan encoding.BinaryMarshaler {
	return nil
}

func (r *RequestWithoutResponse) ErrorsChannel() chan error {
	return nil
}
