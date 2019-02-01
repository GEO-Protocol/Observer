package requests_cluster

import (
	testsCommon "geo-observers-blockchain/tests/network/geo"
	"testing"
)

func TestTSLDebugClusterFlow(t *testing.T) {

	// Positive: tsl propagation to internal observers pool requested.
	// Expected result: tsl is propagated to 4 observers.

	tsl := testsCommon.CreateEmptyTSL(1)
	testsCommon.RequestTSLAppend(t, tsl, 0)

	//// Wait for propagation to be performed.
	//time.Sleep(time.Second * 2)
	//
	//for i:=0; i<4; i++ {
	//	response := testsCommon.RequestTSLIsPresent(t, tsl.TxUUID, i)
	//	if !response.PresentInPool {
	//		t.Error()
	//	}
	//}
}
