/*
 * publisher_test.go
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

func testPublisher(t *testing.T, publisher Publisher) {
	ichan := make(chan bool)

	err := publisher.Publish("one", ichan)
	if nil != err {
		panic(err)
	}

	err = publisher.Publish("one", ichan)
	if nil != err {
		panic(err)
	}

	err = publisher.Publish("two", ichan)
	if nil != err {
		panic(err)
	}

	publisher.Unpublish("one", ichan)
	publisher.Unpublish("two", ichan)
}

func TestPublisher(t *testing.T) {
	marshaler := NewGobMarshaler()
	transport := NewNetTransport(
		marshaler,
		&url.URL{
			Scheme: "tcp",
			Host:   ":25000",
		},
		nil)
	publisher := NewPublisher(transport)
	NewConnector(transport)
	defer func() {
		transport.Close()
		time.Sleep(100 * time.Millisecond)
	}()

	testPublisher(t, publisher)
}

func TestDefaultPublisher(t *testing.T) {
	testPublisher(t, DefaultPublisher)
}

func TestDefaultPublisherHideImpl(t *testing.T) {
	_, okR := DefaultPublisher.(Recver)
	_, okC := DefaultPublisher.(ChanEncoder)

	if okR {
		t.Errorf("DefaultPulisher SHOULD NOT implement TransportRecver")
	}

	if okC {
		t.Errorf("DefaultPulisher SHOULD NOT implement ChanEncoder")
	}
}
