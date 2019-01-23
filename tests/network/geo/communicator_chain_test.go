package geo

import (
	"fmt"
	"geo-observers-blockchain/core/common"
	"geo-observers-blockchain/core/utils"
	"geo-observers-blockchain/tests"
	"os"
	"testing"
)

const (
	LastBlockHeightRequestID = 32
)

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

	sendData(t, conn, []byte{LastBlockHeightRequestID})
	response := getResponse(t, conn)

	const totalExpectedResponseLength = common.Uint32ByteSize * 2
	if len(response) != totalExpectedResponseLength {
		t.Error()
	}

	lastBlockHeight, err := utils.UnmarshalUint64(response)
	if err != nil {
		t.Error(err)
	}

	fmt.Println(lastBlockHeight)
	if lastBlockHeight > 10 {
		t.Error()
	}
}
