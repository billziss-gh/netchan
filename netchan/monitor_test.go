/*
 * monitor_test.go
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
	"net/url"
	"strconv"
	"sync"
	"testing"
)

func TestMonitor(t *testing.T) {
	pchan := make([]chan string, 100)
	cchan := make([]chan string, 100)
	echan := make(chan error)

	for i := range pchan {
		pchan[i] = make(chan string)

		err := Publish("chan"+strconv.Itoa(i), pchan[i])
		if nil != err {
			panic(err)
		}
	}

	for i := range cchan {
		cchan[i] = make(chan string)

		err := Connect(
			&url.URL{
				Scheme: "tcp",
				Host:   "127.0.0.1",
				Path:   "chan" + strconv.Itoa(i),
			},
			cchan[i],
			echan)
		if nil != err {
			panic(err)
		}
	}

	wg := sync.WaitGroup{}

	for i := range cchan {
		i := i
		wg.Add(1)
		go func() {
			cchan[i] <- "msg" + strconv.Itoa(i)
			wg.Done()
		}()
	}

	for i := range pchan {
		i := i
		wg.Add(1)
		go func() {
			s := <-pchan[i]
			if "msg"+strconv.Itoa(i) != s {
				t.Errorf("incorrect msg: expect %v, got %v", "msg"+strconv.Itoa(i), s)
			}
			wg.Done()
		}()
	}

	wg.Wait()

	for i := range cchan {
		close(cchan[i])
	}

	for i := range pchan {
		Unpublish("chan"+strconv.Itoa(i), pchan[i])
	}

	pubmon := DefaultPublisher.(Monitor)
	for _, name := range pubmon.StatNames() {
		switch name {
		case "Recv":
			if pubmon.Stat(name) != 100 {
				t.Errorf("unexpected %v stat", name)
			}
		default:
			if pubmon.Stat(name) != 0 {
				t.Errorf("unexpected %v stat", name)
			}
		}
	}

	conmon := DefaultConnector.(Monitor)
	for _, name := range conmon.StatNames() {
		switch name {
		case "Send":
			if conmon.Stat(name) != 100 {
				t.Errorf("unexpected %v stat", name)
			}
		default:
			if conmon.Stat(name) != 0 {
				t.Errorf("unexpected %v stat", name)
			}
		}
	}
}
