package requests

import (
	"geo-observers-blockchain/core/common/types/transactions"
	"geo-observers-blockchain/core/geo"
)

type ClaimAppend struct {
	*requestWithoutResponse
	TxID  *transactions.TransactionUUID
	Claim *geo.Claim
}

func (r *ClaimAppend) UnmarshalBinary(data []byte) (err error) {
	panic("")
}

// --------------------------------------------------------------------------------------------------------------------

type ClaimGet struct {
	*requestWithResponse
	TxID *transactions.TransactionUUID
}

func (r *ClaimGet) UnmarshalBinary(data []byte) (err error) {
	r.requestWithResponse = newRequestWithResponse()
	return r.TxID.UnmarshalBinary(data)
}

// --------------------------------------------------------------------------------------------------------------------

type ClaimIsPresent struct {
	*requestWithResponse
	TxID *transactions.TransactionUUID
}

func (r *ClaimIsPresent) UnmarshalBinary(data []byte) (err error) {
	r.requestWithResponse = newRequestWithResponse()
	return r.TxID.UnmarshalBinary(data)
}
