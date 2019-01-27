package requests

import (
	"geo-observers-blockchain/core/common/types/transactions"
	"geo-observers-blockchain/core/network/communicator/geo/api/v0/common"
	"geo-observers-blockchain/core/utils"
)

type TxsStates struct {
	*common.RequestWithResponse
	TxIDs *transactions.TransactionIDs
}

func NewTxStates(TxIDs []*transactions.TxID) *TxsStates {
	return &TxsStates{
		TxIDs:               transactions.NewTransactionIDs(TxIDs),
		RequestWithResponse: common.NewRequestWithResponse(),
	}
}

func (request *TxsStates) MarshalBinary() (data []byte, err error) {
	typeID := []byte{common.ReqTxStates}
	txIDsBinary, err := request.TxIDs.MarshalBinary()
	return utils.ChainByteSlices(typeID, txIDsBinary), nil
}

func (request *TxsStates) UnmarshalBinary(data []byte) (err error) {
	request.RequestWithResponse = common.NewRequestWithResponse()
	request.TxIDs = &transactions.TransactionIDs{}
	return request.TxIDs.UnmarshalBinary(data)
}
