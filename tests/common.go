package tests

import (
	"fmt"
	"os/exec"
	"time"
)

const Debug = true

// LaunchTestObserver spawns internal observer process for testing purposes.
func LaunchTestObserver() (err error) {
	if !Debug {
		_ = TerminateObservers()

		// Dropping previous chain data
		_ = exec.Command("rm", "/tmp/observer_tests/data/chain.dat").Run()

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

func LaunchTestObserversCluster() (err error) {
	//if !Debug {

	_ = TerminateObservers()

	// Dropping previous chain data
	for i := 0; i < 4; i++ {
		_ = exec.Command("rm", fmt.Sprint("/tmp/observers_test_cluster/", i, "/data/chain.dat")).Run()
	}

	errors := make(chan error, 4)
	for i := 0; i < 4; i++ {
		go func(observerIndex int) {
			cmd := exec.Command(fmt.Sprint("/tmp/observers_test_cluster/", observerIndex, "/observer"), "--mode=debug-cluster")
			cmd.Dir = fmt.Sprint("/tmp/observers_test_cluster/", observerIndex, "/")
			err = cmd.Start()
			if err != nil {
				errors <- err
			}

			errors <- nil
		}(i)
	}

	for i := 0; i < 4; i++ {
		err := <-errors
		if err != nil {
			return err
		}
	}

	time.Sleep(time.Second * 2)
	return
	//}
	//
	//return
}

func TerminateObservers() (err error) {
	if !Debug {
		return exec.Command("killall", "observer", "-q").Run()
	}

	return
}
