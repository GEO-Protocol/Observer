package requests

import (
	"geo-observers-blockchain/core/common/types/transactions"
	"geo-observers-blockchain/core/geo"
)

type TSLAppend struct {
	*requestWithoutResponse
	TSL *geo.TSL
}

func (r *TSLAppend) UnmarshalBinary(data []byte) (err error) {
	r.TSL = geo.NewTSL()
	return r.TSL.UnmarshalBinary(data)
}

// --------------------------------------------------------------------------------------------------------------------

type TSLGet struct {
	*requestWithResponse
	TxID *transactions.TransactionUUID
}

func (r *TSLGet) UnmarshalBinary(data []byte) (err error) {
	r.requestWithResponse = newRequestWithResponse()
	r.TxID = &transactions.TransactionUUID{}
	return r.TxID.UnmarshalBinary(data)
}

// --------------------------------------------------------------------------------------------------------------------

type TSLIsPresent struct {
	*requestWithResponse
	TxID *transactions.TransactionUUID
}

func (r *TSLIsPresent) UnmarshalBinary(data []byte) (err error) {
	r.requestWithResponse = newRequestWithResponse()
	r.TxID = &transactions.TransactionUUID{}
	return r.TxID.UnmarshalBinary(data)
}
