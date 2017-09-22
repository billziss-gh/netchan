/*
 * exposer_binder_test.go
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

func testExposerBinder(t *testing.T, exposer Exposer, binder Binder) {
	pchan := make(chan string)
	cchan := make(chan string)
	echan := make(chan error)

	err := exposer.Expose("one", pchan)
	if nil != err {
		panic(err)
	}

	err = binder.Bind(
		&url.URL{
			Scheme: "tcp",
			Host:   "127.0.0.1",
			Path:   "one",
		},
		cchan,
		echan)
	if nil != err {
		panic(err)
	}

	cchan <- "fortytwo"

	close(cchan)

	s := <-pchan
	if "fortytwo" != s {
		t.Errorf("incorrect msg: expect %v, got %v", "fortytwo", s)
	}

	exposer.Unexpose("one", pchan)
}

func TestExposerBinder(t *testing.T) {
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

	testExposerBinder(t, exposer, binder)
}

func TestDefaultExposerBinder(t *testing.T) {
	testExposerBinder(t, DefaultExposer, DefaultBinder)
}

func testExposerBinderUri(t *testing.T, exposer Exposer, binder Binder) {
	pchan := make(chan string)
	cchan := make(chan string)
	echan := make(chan error)

	err := exposer.Expose("one", pchan)
	if nil != err {
		panic(err)
	}

	err = binder.Bind("tcp://127.0.0.1/one", cchan, echan)
	if nil != err {
		panic(err)
	}

	cchan <- "fortytwo"

	close(cchan)

	s := <-pchan
	if "fortytwo" != s {
		t.Errorf("incorrect msg: expect %v, got %v", "fortytwo", s)
	}

	exposer.Unexpose("one", pchan)
}

func TestExposerBinderUri(t *testing.T) {
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

	testExposerBinderUri(t, exposer, binder)
}

func testExposerBinderAnyAll(t *testing.T, exposer Exposer, binder Binder,
	anyall string) {
	pchan0 := make(chan string)
	pchan1 := make(chan string)
	pchan2 := make(chan string)
	cchan := make(chan string)
	echan := make(chan error)

	id := anyall + "one"

	err := exposer.Expose(id, pchan0)
	if nil != err {
		panic(err)
	}

	err = exposer.Expose(id, pchan1)
	if nil != err {
		panic(err)
	}

	err = exposer.Expose(id, pchan2)
	if nil != err {
		panic(err)
	}

	err = binder.Bind(
		&url.URL{
			Scheme: "tcp",
			Host:   "127.0.0.1",
			Path:   id,
		},
		cchan,
		echan)
	if nil != err {
		panic(err)
	}

	cchan <- "fortytwo"

	close(cchan)

	cnt := 1
	if "+" == anyall {
		cnt = 3
	}

	sum := 0
	for i := 0; cnt > i; i++ {
		s := ""
		select {
		case s = <-pchan0:
			sum |= 1
		case s = <-pchan1:
			sum |= 2
		case s = <-pchan2:
			sum |= 4
		}
		if "fortytwo" != s {
			t.Errorf("incorrect msg: expect %v, got %v", "fortytwo", s)
		}
	}

	if "+" == anyall {
		if 7 != sum {
			t.Errorf("incorrect sum: expect %v, got %v", 7, sum)
		}
	}

	exposer.Unexpose(id, pchan0)
	exposer.Unexpose(id, pchan1)
	exposer.Unexpose(id, pchan2)
}

func TestExposerBinderAnyAll(t *testing.T) {
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

	testExposerBinderAnyAll(t, exposer, binder, "")
	testExposerBinderAnyAll(t, exposer, binder, "+")
}

func TestDefaultExposerBinderAnyAll(t *testing.T) {
	testExposerBinderAnyAll(t, DefaultExposer, DefaultBinder, "")
	testExposerBinderAnyAll(t, DefaultExposer, DefaultBinder, "+")
}

func testExposerBinderError(t *testing.T,
	transport Transport, exposer Exposer, binder Binder) {
	pchan := make(chan string)
	cchan := make(chan string)
	echan0 := make(chan error)
	echan1 := make(chan error)
	echan2 := make(chan error)

	err := exposer.Expose("one", pchan)
	if nil != err {
		panic(err)
	}

	err = exposer.Expose(IdErr, echan0)
	if nil != err {
		panic(err)
	}

	err = exposer.Expose(IdErr, echan1)
	if nil != err {
		panic(err)
	}

	err = exposer.Expose(IdErr, echan2)
	if nil != err {
		panic(err)
	}

	err = binder.Bind(
		&url.URL{
			Scheme: "tcp",
			Host:   "127.0.0.1",
			Path:   IdErr,
		},
		cchan,
		nil)
	if ErrArgumentInvalid != err {
		t.Error()
	}

	err = binder.Bind(
		&url.URL{
			Scheme: "tcp",
			Host:   "127.0.0.1",
			Path:   "one",
		},
		cchan,
		nil)
	if nil != err {
		panic(err)
	}

	cchan <- "fortytwo"

	close(cchan)

	s := <-pchan
	if "fortytwo" != s {
		t.Errorf("incorrect msg: expect %v, got %v", "fortytwo", s)
	}

	exposer.Unexpose("one", pchan)

	transport.Close()
	time.Sleep(100 * time.Millisecond)

	sum := 0
	for 7 != sum {
		select {
		case <-echan0:
			sum |= 1
		case <-echan1:
			sum |= 2
		case <-echan2:
			sum |= 4
		}
	}
}

func TestExposerBinderError(t *testing.T) {
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
	}()

	testExposerBinderError(t, transport, exposer, binder)
}

func testExposerBinderCloseRecv(t *testing.T, exposer Exposer, binder Binder) {
	pchan := make(chan string)
	cchan := make(chan string)
	echan := make(chan error)

	err := exposer.Expose("one", pchan)
	if nil != err {
		panic(err)
	}

	err = binder.Bind(
		&url.URL{
			Scheme: "tcp",
			Host:   "127.0.0.1",
			Path:   "one",
		},
		cchan,
		echan)
	if nil != err {
		panic(err)
	}

	cchan <- "fortytwo"

	close(cchan)
	close(pchan)

	time.Sleep(100 * time.Millisecond)

	exposer.Unexpose("one", pchan)
}

func TestExposerBinderCloseRecv(t *testing.T) {
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

	testExposerBinderCloseRecv(t, exposer, binder)
}

func testExposerBinderInv(t *testing.T, exposer Exposer, binder Binder) {
	ichan := make(chan Message)
	pchan := make(chan string)
	cchan := make(chan int)
	echan := make(chan error)

	err := exposer.Expose(IdInv, ichan)
	if nil != err {
		panic(err)
	}

	err = exposer.Expose("one", pchan)
	if nil != err {
		panic(err)
	}

	err = binder.Bind(
		&url.URL{
			Scheme: "tcp",
			Host:   "127.0.0.1",
			Path:   "one",
		},
		cchan,
		echan)
	if nil != err {
		panic(err)
	}

	cchan <- 42

	close(cchan)

	m := <-ichan
	if "one" != m.Id || 42 != m.Value.Interface() {
		t.Errorf("incorrect msg: expect invalid message, got %v", m)
	}

	exposer.Unexpose("one", pchan)
	exposer.Unexpose(IdInv, pchan)
}

func TestExposerBinderInv(t *testing.T) {
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

	testExposerBinderInv(t, exposer, binder)
}

func testExposerBinderMulti(t *testing.T, exposer Exposer, binder Binder) {
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

	for i := range cchan {
		cchan[i] <- "msg" + strconv.Itoa(i)
		s := <-pchan[i]
		if "msg"+strconv.Itoa(i) != s {
			t.Errorf("incorrect msg: expect %v, got %v", "msg"+strconv.Itoa(i), s)
		}
	}

	for i := range cchan {
		close(cchan[i])
	}

	for i := range pchan {
		exposer.Unexpose("chan"+strconv.Itoa(i), pchan[i])
	}
}

func TestExposerBinderMulti(t *testing.T) {
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

	testExposerBinderMulti(t, exposer, binder)
}

func TestDefaultExposerBinderMulti(t *testing.T) {
	testExposerBinderMulti(t, DefaultExposer, DefaultBinder)
}

func testExposerBinderMultiConcurrent(t *testing.T, exposer Exposer, binder Binder) {
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
}

func TestExposerBinderMultiConcurrent(t *testing.T) {
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

	testExposerBinderMultiConcurrent(t, exposer, binder)
}

func TestDefaultExposerBinderMultiConcurrent(t *testing.T) {
	testExposerBinderMultiConcurrent(t, DefaultExposer, DefaultBinder)
}

func testExposerBinderRoundtrip(t *testing.T, exposer Exposer, binder Binder) {
	pchan := make(chan chan string)
	cchan := make(chan chan string)
	echan := make(chan error)

	err := exposer.Expose("one", pchan)
	if nil != err {
		panic(err)
	}

	err = binder.Bind(
		&url.URL{
			Scheme: "tcp",
			Host:   "127.0.0.1",
			Path:   "one",
		},
		cchan,
		echan)
	if nil != err {
		panic(err)
	}

	cch := make(chan string)
	cchan <- cch

	close(cchan)

	pch := <-pchan
	pch <- "fortytwo"

	close(pch)

	s := <-cch
	if "fortytwo" != s {
		t.Errorf("incorrect msg: expect %v, got %v", "fortytwo", s)
	}

	exposer.Unexpose("one", pchan)
}

func TestExposerBinderRoundtrip(t *testing.T) {
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

	testExposerBinderRoundtrip(t, exposer, binder)
}

func TestDefaultExposerBinderRoundtrip(t *testing.T) {
	testExposerBinderRoundtrip(t, DefaultExposer, DefaultBinder)
}

func testExposerBinderMultiConcurrentRoundtrip(
	t *testing.T, exposer Exposer, binder Binder) {
	pchan := make([]chan chan string, 100)
	cchan := make([]chan chan string, 100)
	echan := make(chan error)

	for i := range pchan {
		pchan[i] = make(chan chan string)

		err := exposer.Expose("chan"+strconv.Itoa(i), pchan[i])
		if nil != err {
			panic(err)
		}
	}

	for i := range cchan {
		cchan[i] = make(chan chan string)

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
			ch := make(chan string)
			cchan[i] <- ch
			s := <-ch
			if "msg"+strconv.Itoa(i) != s {
				t.Errorf("incorrect msg: expect %v, got %v", "msg"+strconv.Itoa(i), s)
			}
			wg.Done()
		}()
	}

	for i := range pchan {
		i := i
		wg.Add(1)
		go func() {
			ch := <-pchan[i]
			ch <- "msg" + strconv.Itoa(i)
			close(ch)
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
}

func TestExposerBinderMultiConcurrentRoundtrip(t *testing.T) {
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

	testExposerBinderMultiConcurrentRoundtrip(t, exposer, binder)
}

func TestDefaultExposerBinderMultiConcurrentRoundtrip(t *testing.T) {
	testExposerBinderMultiConcurrentRoundtrip(t, DefaultExposer, DefaultBinder)
}
