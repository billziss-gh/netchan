/*
 * stats_test.go
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
	"time"
)

func testStats(t *testing.T, exposer Exposer, binder Binder) {
	pchan := make([]chan string, 100)
	cchan := make([]chan string, 100)
	echan := make(chan error)

	for i := range pchan {
		pchan[i] = make(chan string)

		err := exposer.Expose("chan"+strconv.Itoa(i), pchan[i])
		if nil != err {
			panic(err)
		}
	}

	for i := range cchan {
		cchan[i] = make(chan string)

		err := binder.Bind(
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
		exposer.Unexpose("chan"+strconv.Itoa(i), pchan[i])
	}

	pubstats := exposer.(Stats)
	for _, name := range pubstats.StatNames() {
		switch name {
		case "Recv":
			if pubstats.Stat(name) != 100 {
				t.Errorf("unexpected %v stat", name)
			}
		default:
			if pubstats.Stat(name) != 0 {
				t.Errorf("unexpected %v stat", name)
			}
		}
	}

	constats := binder.(Stats)
	for _, name := range constats.StatNames() {
		switch name {
		case "Send":
			if constats.Stat(name) != 100 {
				t.Errorf("unexpected %v stat", name)
			}
		default:
			if constats.Stat(name) != 0 {
				t.Errorf("unexpected %v stat", name)
			}
		}
	}
}

func TestStats(t *testing.T) {
	marshaler := NewGobMarshaler()
	transport := NewNetTransport(
		marshaler,
		&url.URL{
			Scheme: "tcp",
			Host:   ":25000",
		},
		nil)
	exposer := NewExposer(transport)
	binder := NewBinder(transport)
	defer func() {
		transport.Close()
		time.Sleep(100 * time.Millisecond)
	}()

	testStats(t, exposer, binder)
}
