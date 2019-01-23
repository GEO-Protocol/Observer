package requests

import "encoding"

type Request interface {
	// Returns channel from which the response should be fetched,
	// in case if Request has paired response and client should wait for it (and keep up the connection).
	// Otherwise - returns nil (some requests to the Observer does not assumes any response).
	ResponseChannel() chan encoding.BinaryMarshaler
	UnmarshalBinary(data []byte) (err error)
}

// --------------------------------------------------------------------------------------------------------------------

type requestWithResponse struct {
	response chan encoding.BinaryMarshaler
}

func newRequestWithResponse() *requestWithResponse {
	return &requestWithResponse{
		response: make(chan encoding.BinaryMarshaler, 1),
	}
}

func (r *requestWithResponse) ResponseChannel() chan encoding.BinaryMarshaler {
	return r.response
}

// --------------------------------------------------------------------------------------------------------------------

type requestWithoutResponse struct{}

func (r *requestWithoutResponse) ResponseChannel() chan encoding.BinaryMarshaler {
	return nil
}
