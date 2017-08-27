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
)

func testPublisher(t *testing.T, publisher Publisher) {
	ichan := make(chan bool)
	echan := make(chan error)

	err := publisher.Publish("one", ichan, echan)
	if nil != err {
		panic(err)
	}

	err = publisher.Publish("one", ichan, echan)
	if nil != err {
		panic(err)
	}

	err = publisher.Publish("two", ichan, echan)
	if nil != err {
		panic(err)
	}

	publisher.Unpublish("one", ichan)
	publisher.Unpublish("two", ichan)
}

func TestPublisher(t *testing.T) {
	marshaler := newGobMarshaler()
	transport := newNetTransport(
		marshaler,
		&url.URL{
			Scheme: "tcp",
			Host:   ":25000",
		})
	publisher := newPublisher(transport)
	newConnector(transport)
	defer transport.Close()

	testPublisher(t, publisher)
}
