package requests

import (
	"geo-observers-blockchain/core/network/communicator/geo/api/v0/common"
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
