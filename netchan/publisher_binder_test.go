/*
 * publisher_binder_test.go
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

func testPublisherBinder(t *testing.T, publisher Publisher, binder Binder) {
	pchan := make(chan string)
	cchan := make(chan string)
	echan := make(chan error)

	err := publisher.Publish("one", pchan)
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

	publisher.Unpublish("one", pchan)
}

func TestPublisherBinder(t *testing.T) {
	marshaler := NewGobMarshaler()
	transport := NewNetTransport(
		marshaler,
		&url.URL{
			Scheme: "tcp",
			Host:   ":25000",
		},
		nil)
	publisher := NewPublisher(transport)
	binder := NewBinder(transport)
	defer func() {
		transport.Close()
		time.Sleep(100 * time.Millisecond)
	}()

	testPublisherBinder(t, publisher, binder)
}

func TestDefaultPublisherBinder(t *testing.T) {
	testPublisherBinder(t, DefaultPublisher, DefaultBinder)
}

func testPublisherBinderUri(t *testing.T, publisher Publisher, binder Binder) {
	pchan := make(chan string)
	cchan := make(chan string)
	echan := make(chan error)

	err := publisher.Publish("one", pchan)
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

	publisher.Unpublish("one", pchan)
}

func TestPublisherBinderUri(t *testing.T) {
	marshaler := NewGobMarshaler()
	transport := NewNetTransport(
		marshaler,
		&url.URL{
			Scheme: "tcp",
			Host:   ":25000",
		},
		nil)
	publisher := NewPublisher(transport)
	binder := NewBinder(transport)
	defer func() {
		transport.Close()
		time.Sleep(100 * time.Millisecond)
	}()

	testPublisherBinderUri(t, publisher, binder)
}

func testPublisherBinderAnyAll(t *testing.T, publisher Publisher, binder Binder,
	anyall string) {
	pchan0 := make(chan string)
	pchan1 := make(chan string)
	pchan2 := make(chan string)
	cchan := make(chan string)
	echan := make(chan error)

	id := anyall + "one"

	err := publisher.Publish(id, pchan0)
	if nil != err {
		panic(err)
	}

	err = publisher.Publish(id, pchan1)
	if nil != err {
		panic(err)
	}

	err = publisher.Publish(id, pchan2)
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

	publisher.Unpublish(id, pchan0)
	publisher.Unpublish(id, pchan1)
	publisher.Unpublish(id, pchan2)
}

func TestPublisherBinderAnyAll(t *testing.T) {
	marshaler := NewGobMarshaler()
	transport := NewNetTransport(
		marshaler,
		&url.URL{
			Scheme: "tcp",
			Host:   ":25000",
		},
		nil)
	publisher := NewPublisher(transport)
	binder := NewBinder(transport)
	defer func() {
		transport.Close()
		time.Sleep(100 * time.Millisecond)
	}()

	testPublisherBinderAnyAll(t, publisher, binder, "")
	testPublisherBinderAnyAll(t, publisher, binder, "+")
}

func TestDefaultPublisherBinderAnyAll(t *testing.T) {
	testPublisherBinderAnyAll(t, DefaultPublisher, DefaultBinder, "")
	testPublisherBinderAnyAll(t, DefaultPublisher, DefaultBinder, "+")
}

func testPublisherBinderError(t *testing.T,
	transport Transport, publisher Publisher, binder Binder) {
	pchan := make(chan string)
	cchan := make(chan string)
	echan0 := make(chan error)
	echan1 := make(chan error)
	echan2 := make(chan error)

	err := publisher.Publish("one", pchan)
	if nil != err {
		panic(err)
	}

	err = publisher.Publish(IdErr, echan0)
	if nil != err {
		panic(err)
	}

	err = publisher.Publish(IdErr, echan1)
	if nil != err {
		panic(err)
	}

	err = publisher.Publish(IdErr, echan2)
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

	publisher.Unpublish("one", pchan)

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

func TestPublisherBinderError(t *testing.T) {
	marshaler := NewGobMarshaler()
	transport := NewNetTransport(
		marshaler,
		&url.URL{
			Scheme: "tcp",
			Host:   ":25000",
		},
		nil)
	publisher := NewPublisher(transport)
	binder := NewBinder(transport)
	defer func() {
	}()

	testPublisherBinderError(t, transport, publisher, binder)
}

func testPublisherBinderCloseRecv(t *testing.T, publisher Publisher, binder Binder) {
	pchan := make(chan string)
	cchan := make(chan string)
	echan := make(chan error)

	err := publisher.Publish("one", pchan)
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

	publisher.Unpublish("one", pchan)
}

func TestPublisherBinderCloseRecv(t *testing.T) {
	marshaler := NewGobMarshaler()
	transport := NewNetTransport(
		marshaler,
		&url.URL{
			Scheme: "tcp",
			Host:   ":25000",
		},
		nil)
	publisher := NewPublisher(transport)
	binder := NewBinder(transport)
	defer func() {
		transport.Close()
		time.Sleep(100 * time.Millisecond)
	}()

	testPublisherBinderCloseRecv(t, publisher, binder)
}

func testPublisherBinderInv(t *testing.T, publisher Publisher, binder Binder) {
	ichan := make(chan Message)
	pchan := make(chan string)
	cchan := make(chan int)
	echan := make(chan error)

	err := publisher.Publish(IdInv, ichan)
	if nil != err {
		panic(err)
	}

	err = publisher.Publish("one", pchan)
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

	publisher.Unpublish("one", pchan)
	publisher.Unpublish(IdInv, pchan)
}

func TestPublisherBinderInv(t *testing.T) {
	marshaler := NewGobMarshaler()
	transport := NewNetTransport(
		marshaler,
		&url.URL{
			Scheme: "tcp",
			Host:   ":25000",
		},
		nil)
	publisher := NewPublisher(transport)
	binder := NewBinder(transport)
	defer func() {
		transport.Close()
		time.Sleep(100 * time.Millisecond)
	}()

	testPublisherBinderInv(t, publisher, binder)
}

func testPublisherBinderMulti(t *testing.T, publisher Publisher, binder Binder) {
	pchan := make([]chan string, 100)
	cchan := make([]chan string, 100)
	echan := make(chan error)

	for i := range pchan {
		pchan[i] = make(chan string)

		err := publisher.Publish("chan"+strconv.Itoa(i), pchan[i])
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
		publisher.Unpublish("chan"+strconv.Itoa(i), pchan[i])
	}
}

func TestPublisherBinderMulti(t *testing.T) {
	marshaler := NewGobMarshaler()
	transport := NewNetTransport(
		marshaler,
		&url.URL{
			Scheme: "tcp",
			Host:   ":25000",
		},
		nil)
	publisher := NewPublisher(transport)
	binder := NewBinder(transport)
	defer func() {
		transport.Close()
		time.Sleep(100 * time.Millisecond)
	}()

	testPublisherBinderMulti(t, publisher, binder)
}

func TestDefaultPublisherBinderMulti(t *testing.T) {
	testPublisherBinderMulti(t, DefaultPublisher, DefaultBinder)
}

func testPublisherBinderMultiConcurrent(t *testing.T, publisher Publisher, binder Binder) {
	pchan := make([]chan string, 100)
	cchan := make([]chan string, 100)
	echan := make(chan error)

	for i := range pchan {
		pchan[i] = make(chan string)

		err := publisher.Publish("chan"+strconv.Itoa(i), pchan[i])
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
		publisher.Unpublish("chan"+strconv.Itoa(i), pchan[i])
	}
}

func TestPublisherBinderMultiConcurrent(t *testing.T) {
	marshaler := NewGobMarshaler()
	transport := NewNetTransport(
		marshaler,
		&url.URL{
			Scheme: "tcp",
			Host:   ":25000",
		},
		nil)
	publisher := NewPublisher(transport)
	binder := NewBinder(transport)
	defer func() {
		transport.Close()
		time.Sleep(100 * time.Millisecond)
	}()

	testPublisherBinderMultiConcurrent(t, publisher, binder)
}

func TestDefaultPublisherBinderMultiConcurrent(t *testing.T) {
	testPublisherBinderMultiConcurrent(t, DefaultPublisher, DefaultBinder)
}

func testPublisherBinderRoundtrip(t *testing.T, publisher Publisher, binder Binder) {
	pchan := make(chan chan string)
	cchan := make(chan chan string)
	echan := make(chan error)

	err := publisher.Publish("one", pchan)
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

	publisher.Unpublish("one", pchan)
}

func TestPublisherBinderRoundtrip(t *testing.T) {
	marshaler := NewGobMarshaler()
	transport := NewNetTransport(
		marshaler,
		&url.URL{
			Scheme: "tcp",
			Host:   ":25000",
		},
		nil)
	publisher := NewPublisher(transport)
	binder := NewBinder(transport)
	defer func() {
		transport.Close()
		time.Sleep(100 * time.Millisecond)
	}()

	testPublisherBinderRoundtrip(t, publisher, binder)
}

func TestDefaultPublisherBinderRoundtrip(t *testing.T) {
	testPublisherBinderRoundtrip(t, DefaultPublisher, DefaultBinder)
}

func testPublisherBinderMultiConcurrentRoundtrip(
	t *testing.T, publisher Publisher, binder Binder) {
	pchan := make([]chan chan string, 100)
	cchan := make([]chan chan string, 100)
	echan := make(chan error)

	for i := range pchan {
		pchan[i] = make(chan chan string)

		err := publisher.Publish("chan"+strconv.Itoa(i), pchan[i])
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
		publisher.Unpublish("chan"+strconv.Itoa(i), pchan[i])
	}
}

func TestPublisherBinderMultiConcurrentRoundtrip(t *testing.T) {
	marshaler := NewGobMarshaler()
	transport := NewNetTransport(
		marshaler,
		&url.URL{
			Scheme: "tcp",
			Host:   ":25000",
		},
		nil)
	publisher := NewPublisher(transport)
	binder := NewBinder(transport)
	defer func() {
		transport.Close()
		time.Sleep(100 * time.Millisecond)
	}()

	testPublisherBinderMultiConcurrentRoundtrip(t, publisher, binder)
}

func TestDefaultPublisherBinderMultiConcurrentRoundtrip(t *testing.T) {
	testPublisherBinderMultiConcurrentRoundtrip(t, DefaultPublisher, DefaultBinder)
}
