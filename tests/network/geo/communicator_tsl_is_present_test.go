package geo

import (
	"geo-observers-blockchain/core/common/types/transactions"
	"geo-observers-blockchain/core/network/communicator/geo/api/v0/common"
	"geo-observers-blockchain/core/network/communicator/geo/api/v0/requests"
	"geo-observers-blockchain/core/network/communicator/geo/api/v0/responses"
	"testing"
)

const (
	TSLIsPresentRequestID = 66
)

func TestTSLIsPresentRequestID(t *testing.T) {
	if //noinspection GoBoolExpressions
	TSLIsPresentRequestID != common.ReqTSLIsPresent {
		t.Fatal()
	}
}

func requestTSLIsPresent(t *testing.T, TxID *transactions.TransactionUUID) *responses.TSLIsPresent {
	conn := ConnectToObserver(t)
	defer conn.Close()

	request := requests.NewTSLIsPresent(TxID)
	SendRequest(t, request, conn)

	response := &responses.TSLIsPresent{}
	GetResponse(t, response, conn)
	return response
}
