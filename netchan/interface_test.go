/*
 * interface_test.go
 *
 * Copyright 2017 Bill Zissimopoulos
 */
/*
 * This file is part of netchan.
 *
 * It is licensed under the MIT license. The full license text can be found
 * in the License.txt file at the root of this project.
 */

package netchan

import (
	"errors"
	"fmt"
	"sync"
	"time"
)

func ping(wg *sync.WaitGroup, count int) {
	defer wg.Done()

	pingch := make(chan chan struct{})
	errch := make(chan error, 1)
	err := Bind("tcp://127.0.0.1/pingpong", pingch, errch)
	if nil != err {
		panic(err)
	}

	for i := 0; count > i; i++ {
		// send a new pong (response) channel
		pongch := make(chan struct{})
		pingch <- pongch

		fmt.Println("ping")

		// wait for pong response, error or timeout
		select {
		case <-pongch:
		case err = <-errch:
			panic(err)
		case <-time.After(10 * time.Second):
			err = errors.New("timeout")
			panic(err)
		}
	}

	pingch <- nil

	close(pingch)
}

func pong(wg *sync.WaitGroup, exposed chan struct{}) {
	defer wg.Done()

	pingch := make(chan chan struct{})
	err := Expose("pingpong", pingch)
	if nil != err {
		panic(err)
	}

	close(exposed)

	for {
		// receive the pong (response) channel
		pongch := <-pingch
		if nil == pongch {
			fmt.Println("END")
			break
		}

		fmt.Println("pong")

		// send the pong response
		pongch <- struct{}{}
	}

	Unexpose("pingpong", pingch)
}

func Example() {
	wg := &sync.WaitGroup{}

	exposed := make(chan struct{})
	wg.Add(1)
	go pong(wg, exposed)
	<-exposed

	wg.Add(1)
	go ping(wg, 10)

	wg.Wait()

	// Output:
	// ping
	// pong
	// ping
	// pong
	// ping
	// pong
	// ping
	// pong
	// ping
	// pong
	// ping
	// pong
	// ping
	// pong
	// ping
	// pong
	// ping
	// pong
	// ping
	// pong
	// END
}
