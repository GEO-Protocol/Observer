package geo

import (
	"geo-observers-blockchain/core/common/types/transactions"
	"geo-observers-blockchain/core/geo"
	"geo-observers-blockchain/core/network/communicator/geo/api/v0/common"
	"geo-observers-blockchain/core/network/communicator/geo/api/v0/requests"
	"testing"
)

const (
	AppendTSLRequestID = 64
)

func TestTSLAppendRequestID(t *testing.T) {
	if //noinspection GoBoolExpressions
	AppendTSLRequestID != common.ReqTSLAppend {
		t.Fatal()
	}
}

func TestTSLAppendPoolOnly(t *testing.T) {
	{
		// TSL with one signature.
		// (check in pool only).
		tsl := createEmptyTSL(1)
		requestTSLAppend(t, tsl)

		response := requestTSLIsPresent(t, tsl.TxUUID)
		if !response.PresentInPool {
			t.Error()
		}
	}

	{
		// TSL with one signature.
		// (check in block only, observers cluster is needed).
		// todo: add implementation.
	}

	{
		// TSL with several signatures
	}
}

func TestTSLAppendToChain(t *testing.T) {
	// todo: implement

	{
		// TSL with one signature.
	}

	{
		// TSL with several signatures
	}
}

func TestInvalidTSL(t *testing.T) {
	// todo: implement

	{
		// Invalid data: no signatures
	}

	{
		// Invalid data: invalid signatures
	}
}

func requestTSLAppend(t *testing.T, tsl *geo.TSL) {
	conn := connectToObserver(t)
	defer conn.Close()

	request := requests.TSLAppend{TSL: tsl}
	requestBinary, err := request.MarshalBinary()
	if err != nil {
		t.Error()
	}

	sendData(t, conn, requestBinary)
}

func createEmptyTSL(membersCount int) (tsl *geo.TSL) {
	txID, _ := transactions.NewRandomTransactionUUID()
	members := &transactions.Members{}

	for i := 0; i < membersCount; i++ {
		_ = members.Add(transactions.NewMember(uint16(i)))
	}

	tsl = &geo.TSL{
		TxUUID:  txID,
		Members: members,
	}
	return
}
