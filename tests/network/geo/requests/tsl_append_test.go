package requests

import (
	"geo-observers-blockchain/core/network/communicator/geo/api/v0/common"
	testsCommon "geo-observers-blockchain/tests/network/geo"
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
		// Positive: TSL with one signature.
		// Expected result: received response with
		//          approve that TSL with exact TxID is present in pool.

		tsl := testsCommon.CreateEmptyTSL(1)
		testsCommon.RequestTSLAppend(t, tsl, 0)

		response := testsCommon.RequestTSLIsPresent(t, tsl.TxUUID, 0)
		if !response.PresentInPool {
			t.Error()
		}
	}

	{
		// Positive: TSL with 10 signatures.
		// Expected result: received response with
		//          approve that TSL with exact TxID is present in pool.

		tsl := testsCommon.CreateEmptyTSL(10)
		testsCommon.RequestTSLAppend(t, tsl, 0)

		response := testsCommon.RequestTSLIsPresent(t, tsl.TxUUID, 0)
		if !response.PresentInPool {
			t.Error()
		}
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
