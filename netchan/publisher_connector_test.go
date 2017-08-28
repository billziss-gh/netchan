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
	"testing"
	"time"
)

func testPublisherConnector(t *testing.T, publisher Publisher, connector Connector) {
	pchan := make(chan string)
	cchan := make(chan string)
	echan := make(chan error)

	err := publisher.Publish("one", pchan, echan)
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

func testPublisherConnectorMulti(t *testing.T, publisher Publisher, connector Connector) {
	pchan := make([]chan string, 100)
	cchan := make([]chan string, 100)
	echan := make(chan error)

	for i := range pchan {
		pchan[i] = make(chan string)

		err := publisher.Publish("chan"+strconv.Itoa(i), pchan[i], echan)
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
			t.Errorf("incorrect msg: expect %v, got %v", "fortytwo", s)
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
