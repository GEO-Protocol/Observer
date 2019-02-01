package config

import (
	"geo-observers-blockchain/tests"
	"os/exec"
	"strings"
	"testing"
	"time"
)

func TestDebugClusterMode(t *testing.T) {
	_ = tests.TerminateObservers()

	cmd := exec.Command("/tmp/observer_tests/observer", "--mode=debug-cluster")
	cmd.Dir = "/tmp/observer_tests/"

	go func() {
		time.Sleep(time.Second)
		_ = tests.TerminateObservers()
	}()

	out, _ := cmd.Output()
	log := string(out)
	if !strings.Contains(log, "DebugCluster") {
		t.Error()
	}
}
