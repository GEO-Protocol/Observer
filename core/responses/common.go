package responses

import "geo-observers-blockchain/core/requests"

type Response interface {
	MarshalBinary() ([]byte, error)
	UnmarshalBinary([]byte) error
	Request() requests.Request
}
