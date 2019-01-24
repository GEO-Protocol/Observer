package geo

import (
	"geo-observers-blockchain/core/common/types/transactions"
	"geo-observers-blockchain/core/network/communicator/geo/api/v0/common"
	"geo-observers-blockchain/core/network/communicator/geo/api/v0/requests"
	"geo-observers-blockchain/core/network/communicator/geo/api/v0/responses"
	"testing"
)

const (
	TSLGetRequestID = 68
)

func TestTSLGetRequestID(t *testing.T) {
	if //noinspection GoBoolExpressions
	TSLGetRequestID != common.ReqTSLGet {
		t.Fatal()
	}
}

func TestTSLGet(t *testing.T) {
	{
		// TSL that is ABSENT in chain.
		response := requestTSLGet(t, transactions.NewTransactionUUID())
		if response.IsPresent {
			t.Error()
		}

		if response.TSL != nil {
			t.Error()
		}
	}

	{
		// TSL that is present in chain
		// todo: implement
		// todo: add observers clusters
	}
}

func requestTSLGet(t *testing.T, TxID *transactions.TransactionUUID) *responses.TSLGet {
	conn := connectToObserver(t)
	defer conn.Close()

	request := requests.NewTSLGet(TxID)
	sendRequest(t, request, conn)

	response := &responses.TSLGet{}
	getResponse(t, response, conn)
	return response
}
