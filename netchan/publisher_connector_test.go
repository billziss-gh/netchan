/*
 * publisher_connector_test.go
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

func testPublisherConnector(t *testing.T, publisher Publisher, connector Connector) {
	pchan := make(chan string)
	cchan := make(chan string)
	echan := make(chan error)

	err := publisher.Publish("one", pchan)
	if nil != err {
		panic(err)
	}

	err = connector.Connect(
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

func TestPublisherConnector(t *testing.T) {
	marshaler := newGobMarshaler()
	transport := newNetTransport(
		marshaler,
		&url.URL{
			Scheme: "tcp",
			Host:   ":25000",
		})
	publisher := newPublisher(transport)
	connector := newConnector(transport)
	defer func() {
		time.Sleep(100 * time.Millisecond)
		transport.Close()
		time.Sleep(100 * time.Millisecond)
	}()

	testPublisherConnector(t, publisher, connector)
}

func testPublisherConnectorUri(t *testing.T, publisher Publisher, connector Connector) {
	pchan := make(chan string)
	cchan := make(chan string)
	echan := make(chan error)

	err := publisher.Publish("one", pchan)
	if nil != err {
		panic(err)
	}

	err = connector.Connect("tcp://127.0.0.1/one", cchan, echan)
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

func TestPublisherConnectorUri(t *testing.T) {
	marshaler := newGobMarshaler()
	transport := newNetTransport(
		marshaler,
		&url.URL{
			Scheme: "tcp",
			Host:   ":25000",
		})
	publisher := newPublisher(transport)
	connector := newConnector(transport)
	defer func() {
		time.Sleep(100 * time.Millisecond)
		transport.Close()
		time.Sleep(100 * time.Millisecond)
	}()

	testPublisherConnectorUri(t, publisher, connector)
}

func TestDefaultPublisherConnector(t *testing.T) {
	testPublisherConnector(t, DefaultPublisher, DefaultConnector)
	time.Sleep(100 * time.Millisecond)
}

func testPublisherConnectorAnyAll(t *testing.T, publisher Publisher, connector Connector,
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

	err = connector.Connect(
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

func TestPublisherConnectorAnyAll(t *testing.T) {
	marshaler := newGobMarshaler()
	transport := newNetTransport(
		marshaler,
		&url.URL{
			Scheme: "tcp",
			Host:   ":25000",
		})
	publisher := newPublisher(transport)
	connector := newConnector(transport)
	defer func() {
		time.Sleep(100 * time.Millisecond)
		transport.Close()
		time.Sleep(100 * time.Millisecond)
	}()

	testPublisherConnectorAnyAll(t, publisher, connector, "")
	testPublisherConnectorAnyAll(t, publisher, connector, "+")
}

func TestDefaultPublisherConnectorAnyAll(t *testing.T) {
	testPublisherConnectorAnyAll(t, DefaultPublisher, DefaultConnector, "")
	testPublisherConnectorAnyAll(t, DefaultPublisher, DefaultConnector, "+")
	time.Sleep(100 * time.Millisecond)
}

func testPublisherConnectorError(t *testing.T,
	transport Transport, publisher Publisher, connector Connector) {
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

	err = connector.Connect(
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

	err = connector.Connect(
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

	time.Sleep(100 * time.Millisecond)
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

func TestPublisherConnectorError(t *testing.T) {
	marshaler := newGobMarshaler()
	transport := newNetTransport(
		marshaler,
		&url.URL{
			Scheme: "tcp",
			Host:   ":25000",
		})
	publisher := newPublisher(transport)
	connector := newConnector(transport)
	defer func() {
	}()

	testPublisherConnectorError(t, transport, publisher, connector)
}

func testPublisherConnectorMulti(t *testing.T, publisher Publisher, connector Connector) {
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

		err := connector.Connect(
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

func TestPublisherConnectorMulti(t *testing.T) {
	marshaler := newGobMarshaler()
	transport := newNetTransport(
		marshaler,
		&url.URL{
			Scheme: "tcp",
			Host:   ":25000",
		})
	publisher := newPublisher(transport)
	connector := newConnector(transport)
	defer func() {
		time.Sleep(100 * time.Millisecond)
		transport.Close()
		time.Sleep(100 * time.Millisecond)
	}()

	testPublisherConnectorMulti(t, publisher, connector)
}

func TestDefaultPublisherConnectorMulti(t *testing.T) {
	testPublisherConnectorMulti(t, DefaultPublisher, DefaultConnector)
	time.Sleep(100 * time.Millisecond)
}

func testPublisherConnectorMultiConcurrent(t *testing.T, publisher Publisher, connector Connector) {
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

		err := connector.Connect(
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

func TestPublisherConnectorMultiConcurrent(t *testing.T) {
	marshaler := newGobMarshaler()
	transport := newNetTransport(
		marshaler,
		&url.URL{
			Scheme: "tcp",
			Host:   ":25000",
		})
	publisher := newPublisher(transport)
	connector := newConnector(transport)
	defer func() {
		time.Sleep(100 * time.Millisecond)
		transport.Close()
		time.Sleep(100 * time.Millisecond)
	}()

	testPublisherConnectorMultiConcurrent(t, publisher, connector)
}

func TestDefaultPublisherConnectorMultiConcurrent(t *testing.T) {
	testPublisherConnectorMultiConcurrent(t, DefaultPublisher, DefaultConnector)
	time.Sleep(100 * time.Millisecond)
}
