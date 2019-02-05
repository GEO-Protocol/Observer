package requests

import (
	"geo-observers-blockchain/core/common/types/transactions"
	"geo-observers-blockchain/core/network/communicator/geo/api/v0/common"
	"geo-observers-blockchain/core/network/communicator/geo/api/v0/requests"
	"geo-observers-blockchain/core/network/communicator/geo/api/v0/responses"
	testsCommon "geo-observers-blockchain/tests/network/geo"
	"testing"
	"time"
)

const (
	TxStatesRequestID = 192
)

func TestTxStatesRequestID(t *testing.T) {
	if //noinspection GoBoolExpressions
	TxStatesRequestID != common.ReqTxStates {
		t.Fatal()
	}
}

func TestTxStateUnknownTransactions(t *testing.T) {
	{
		// Positive: one transaction ID requested.
		// Expected result: response with state == "No Info".

		txID, _ := transactions.NewRandomTxID(1)
		response := requestTxStates(t, []*transactions.TxID{txID})

		if len(response.States.At) != 1 {
			t.Error("Expected one state, got ", len(response.States.At))
		}

		if response.States.At[0].State != transactions.TxStateNoInfo {
			t.Error("Expected state 'TxStateNoInfo', got ", response.States.At[0].State)
		}
	}

	{
		// Positive: several transaction IDs requested.
		// Expected result: the same count of responses each with s == "No Info".

		tx1ID, _ := transactions.NewRandomTxID(1)
		tx2ID, _ := transactions.NewRandomTxID(1)
		tx3ID, _ := transactions.NewRandomTxID(1)
		tx4ID, _ := transactions.NewRandomTxID(1)
		txIDs := []*transactions.TxID{tx1ID, tx2ID, tx3ID, tx4ID}

		response := requestTxStates(t, txIDs)

		if len(response.States.At) != len(txIDs) {
			t.Error("Expected ", len(txIDs), " states, got ", len(response.States.At))
		}

		for _, s := range response.States.At {
			if s.State != transactions.TxStateNoInfo {
				t.Error("Expected s 'TxStateNoInfo', got ", s)
			}
		}
	}

	{
		// Negative: no transaction IDs are present in request.
		// Expected result: connection drop.

		conn := testsCommon.ConnectToObserver(t, 0)
		defer conn.Close()

		request := requests.NewTxStates([]*transactions.TxID{})
		testsCommon.SendRequest(t, request, conn)

		_ = conn.SetReadDeadline(time.Now())
		one := []byte{0}
		_, err := conn.Read(one)
		if err == nil {
			t.Error("Expected connection to be closed, but it seems that not.")
		}
	}
}

func TestTxStateClaimIsPresentInPool(t *testing.T) {
	// todo: implement
}

func TestTxStateClaimIsPresentInChain(t *testing.T) {
	// todo: implement
}

func TestTxStateTSLIsPresentInChain(t *testing.T) {
	// todo: implement
}

func requestTxStates(t *testing.T, TxIDs []*transactions.TxID) *responses.TxStates {
	conn := testsCommon.ConnectToObserver(t, 0)
	defer conn.Close()

	request := requests.NewTxStates(TxIDs)
	testsCommon.SendRequest(t, request, conn)

	response := &responses.TxStates{}
	testsCommon.GetResponse(t, response, conn)
	return response
}
