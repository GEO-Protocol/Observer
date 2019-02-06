package requests

import (
	"geo-observers-blockchain/core/common/types/transactions"
	"geo-observers-blockchain/core/geo"
	"geo-observers-blockchain/core/network/communicator/geo/api/v0/common"
	"geo-observers-blockchain/core/utils"
)

type TSLAppend struct {
	common.RequestWithoutResponse
	TSL *geo.TSL
}

func (request *TSLAppend) MarshalBinary() (data []byte, err error) {
	typeID := []byte{common.ReqTSLAppend}
	tslBinary, err := request.TSL.MarshalBinary()
	return utils.ChainByteSlices(typeID, tslBinary), nil
}

func (request *TSLAppend) UnmarshalBinary(data []byte) (err error) {
	request.TSL = geo.NewTSL()
	return request.TSL.UnmarshalBinary(data)
}

// --------------------------------------------------------------------------------------------------------------------

type TSLGet struct {
	common.RequestWithResponse
	TxID *transactions.TxID
}

func NewTSLGet(TxID *transactions.TxID) *TSLGet {
	return &TSLGet{
		RequestWithResponse: *(common.NewRequestWithResponse()),
		TxID:                TxID,
	}
}

func (request *TSLGet) MarshalBinary() (data []byte, err error) {
	typeID := []byte{common.ReqTSLGet}
	txIDBinary, err := request.TxID.MarshalBinary()
	return utils.ChainByteSlices(typeID, txIDBinary), nil
}

func (request *TSLGet) UnmarshalBinary(data []byte) (err error) {
	request.RequestWithResponse = *(common.NewRequestWithResponse())
	request.TxID = &transactions.TxID{}
	return request.TxID.UnmarshalBinary(data)
}

// --------------------------------------------------------------------------------------------------------------------

type TSLIsPresent struct {
	common.RequestWithResponse
	TxID *transactions.TxID
}

func NewTSLIsPresent(TxID *transactions.TxID) *TSLIsPresent {
	return &TSLIsPresent{
		RequestWithResponse: *(common.NewRequestWithResponse()),
		TxID:                TxID,
	}
}

func (request *TSLIsPresent) MarshalBinary() (data []byte, err error) {
	typeID := []byte{common.ReqTSLIsPresent}
	txIDBinary, err := request.TxID.MarshalBinary()
	return utils.ChainByteSlices(typeID, txIDBinary), nil
}

func (request *TSLIsPresent) UnmarshalBinary(data []byte) (err error) {
	request.RequestWithResponse = *(common.NewRequestWithResponse())
	request.TxID = &transactions.TxID{}
	return request.TxID.UnmarshalBinary(data)
}
