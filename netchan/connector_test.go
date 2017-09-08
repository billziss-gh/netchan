/*
 * connector_test.go
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
	"testing"
	"time"
)

func testConnector(t *testing.T, connector Connector) {
	ichan := make(chan bool)

	err := connector.Connect(
		&url.URL{
			Scheme: "tcp",
			Host:   "127.0.0.1",
			Path:   "one",
		},
		ichan,
		nil)
	if nil != err {
		panic(err)
	}

	err = connector.Connect(
		&url.URL{
			Scheme: "tcp",
			Host:   "127.0.0.2",
			Path:   "one",
		},
		ichan,
		nil)
	if ErrArgumentConnected != err {
		t.Errorf("incorrect error: expect %v, got %v", ErrArgumentConnected, err)
	}

	err = connector.Connect(
		nil,
		ichan,
		make(chan error))
	if nil != err {
		panic(err)
	}

	err = connector.Connect(
		nil,
		make(chan int),
		make(chan error))
	if ErrArgumentNotConnected != err {
		t.Errorf("incorrect error: expect %v, got %v", ErrArgumentNotConnected, err)
	}

	close(ichan)
}

func TestConnector(t *testing.T) {
	marshaler := newGobMarshaler()
	transport := newNetTransport(
		marshaler,
		&url.URL{
			Scheme: "tcp",
			Host:   ":25000",
		})
	newPublisher(transport)
	connector := newConnector(transport)
	defer func() {
		transport.Close()
		time.Sleep(100 * time.Millisecond)
	}()

	testConnector(t, connector)
}

func TestDefaultConnector(t *testing.T) {
	testConnector(t, DefaultConnector)
}
