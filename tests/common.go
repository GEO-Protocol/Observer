package tests

import (
	"fmt"
	"os/exec"
	"time"
)

const Debug = false

// LaunchTestObserver spawns internal observer process for testing purposes.
func LaunchTestObserver() (err error) {

	// Dropping previous chain data
	_ = exec.Command("rm", "/tmp/observer_tests/data/chain.dat").Run()

	if !Debug {
		_ = TerminateObserver()
		cmd := exec.Command("/tmp/observer_tests/observer")
		cmd.Dir = "/tmp/observer_tests/"

		println("looking for the observer executable at ", cmd.Dir)
		err = cmd.Start()
		if err != nil {
			fmt.Println(err)
			return
		}

		// Timout is needed for the observer to init it's interfaces.
		time.Sleep(time.Second * 3)
	}

	return
}

func TerminateObserver() (err error) {
	if !Debug {
		return exec.Command("killall", "observer", "-q").Run()
	}

	return
}
