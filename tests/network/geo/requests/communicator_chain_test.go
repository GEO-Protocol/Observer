package requests

import (
	"geo-observers-blockchain/core/network/communicator/geo/api/v0/common"
	"geo-observers-blockchain/core/network/communicator/geo/api/v0/requests"
	"geo-observers-blockchain/core/network/communicator/geo/api/v0/responses"
	testsCommon "geo-observers-blockchain/tests/network/geo"
	"testing"
)

const (
	LastBlockNumberRequestID = 32
)

func TestLastBlockNumberRequestID(t *testing.T) {
	if //noinspection GoBoolExpressions
	LastBlockNumberRequestID != common.ReqChainLastBlockNumber {
		t.Fatal()
	}
}

func TestLastBlockHeight(t *testing.T) {
	conn := testsCommon.ConnectToObserver(t)
	defer conn.Close()

	request := &requests.LastBlockNumber{}
	testsCommon.SendRequest(t, request, conn)

	response := &responses.LastBlockHeight{}
	testsCommon.GetResponse(t, response, conn)

	if response.Height > 10 {
		// Observers was started recently, chain height must be relatively small.
		t.Error()
	}

	if response.Height < 1 {
		// Chain height can't be 0 or less.
		t.Error()
	}
}
