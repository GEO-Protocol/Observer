package requests

import (
	"geo-observers-blockchain/core/common/types/transactions"
	"geo-observers-blockchain/core/network/communicator/geo/api/v0/common"
	"geo-observers-blockchain/core/network/communicator/geo/api/v0/requests"
	"geo-observers-blockchain/core/network/communicator/geo/api/v0/responses"
	testsCommon "geo-observers-blockchain/tests/network/geo"
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

func requestTSLIsPresent(t *testing.T, TxID *transactions.TxID) *responses.TSLIsPresent {
	conn := testsCommon.ConnectToObserver(t)
	defer conn.Close()

	request := requests.NewTSLIsPresent(TxID)
	testsCommon.SendRequest(t, request, conn)

	response := &responses.TSLIsPresent{}
	testsCommon.GetResponse(t, response, conn)
	return response
}
