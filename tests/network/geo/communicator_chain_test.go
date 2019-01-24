package geo

import (
	"fmt"
	"geo-observers-blockchain/core/network/communicator/geo/api/v0/common"
	"geo-observers-blockchain/core/network/communicator/geo/api/v0/requests"
	"geo-observers-blockchain/core/network/communicator/geo/api/v0/responses"
	"geo-observers-blockchain/tests"
	"os"
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

func TestMain(m *testing.M) {
	err := tests.LaunchTestObserver()
	if err != nil {
		fmt.Println(err)
		return
	}

	code := m.Run()

	err = tests.TerminateObserver()
	if err != nil {
		fmt.Println(err)
		return
	}

	os.Exit(code)
}

func TestLastBlockHeight(t *testing.T) {
	conn := connectToObserver(t)
	defer conn.Close()

	request := &requests.LastBlockNumber{}
	sendRequest(t, request, conn)

	response := &responses.LastBlockHeight{}
	getResponse(t, response, conn)

	if response.Height > 10 {
		// Observers was started recently, chain height must be relatively small.
		t.Error()
	}

	if response.Height < 1 {
		// Chain height can't be 0 or less.
		t.Error()
	}
}
