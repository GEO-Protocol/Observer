package requests_cluster

import (
	"fmt"
	"geo-observers-blockchain/tests"
	"os"
	"testing"
)

func TestMain(m *testing.M) {
	err := tests.LaunchTestObserversCluster()
	if err != nil {
		fmt.Println(err)
		return
	}

	code := m.Run()

	err = tests.TerminateObservers()
	if err != nil {
		fmt.Println(err)
		return
	}

	os.Exit(code)
}
