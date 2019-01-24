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
	ClaimIsPresentRequestID = 130
)

func TestClaimIsRequestID(t *testing.T) {
	if //noinspection GoBoolExpressions
	ClaimIsPresentRequestID != common.ReqClaimIsPresent {
		t.Fatal()
	}
}

func requestClaimIsPresent(t *testing.T, TxID *transactions.TransactionUUID) *responses.ClaimIsPresent {
	conn := testsCommon.ConnectToObserver(t)
	defer conn.Close()

	request := requests.NewClaimIsPresent(TxID)
	testsCommon.SendRequest(t, request, conn)

	response := &responses.ClaimIsPresent{}
	testsCommon.GetResponse(t, response, conn)
	return response
}
