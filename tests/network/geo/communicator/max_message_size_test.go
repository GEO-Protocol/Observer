package communicator

import (
	communicator "geo-observers-blockchain/core/network/communicator/geo"
	testsCommon "geo-observers-blockchain/tests/network/geo"
	"testing"
)

const (
	MaxMessageSize = 1024 * 1024 * 32 // 32 MB
	MinMessageSize = 1
)

func TestCommunicatorMaxMessageSizeConst(t *testing.T) {
	if //noinspection GoBoolExpressions
	communicator.MaxMessageSize != MaxMessageSize {
		t.Fatal()
	}
}

func TestCommunicatorMinMessageSizeConst(t *testing.T) {
	if //noinspection GoBoolExpressions
	communicator.MinMessageSize != MinMessageSize {
		t.Fatal()
	}

	if //noinspection GoBoolExpressions
	communicator.MinMessageSize < 1 {
		t.Fatal()
	}
}

func TestTooBigMessageMessage(t *testing.T) {
	message := make([]byte, MaxMessageSize+1024)

	conn := testsCommon.ConnectToObserver(t)
	err := testsCommon.SendDataOrReportError(conn, message)
	if err == nil {
		t.Error("Connection must be dropped by the observer, but transfer was successful")
	}

	err = testsCommon.SendDataOrReportError(conn, message)
	if err == nil {
		t.Error("Connection must be closed already, but transfer was successful")
	}
}
