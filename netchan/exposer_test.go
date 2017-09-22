/*
 * exposer_test.go
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

func testExposer(t *testing.T, exposer Exposer) {
	ichan := make(chan bool)

	err := exposer.Expose("one", ichan)
	if nil != err {
		panic(err)
	}

	err = exposer.Expose("one", ichan)
	if nil != err {
		panic(err)
	}

	err = exposer.Expose("two", ichan)
	if nil != err {
		panic(err)
	}

	exposer.Unexpose("one", ichan)
	exposer.Unexpose("two", ichan)
}

func TestExposer(t *testing.T) {
	marshaler := NewGobMarshaler()
	transport := NewNetTransport(
		marshaler,
		&url.URL{
			Scheme: "tcp",
			Host:   ":25000",
		},
		nil)
	exposer := NewExposer(transport)
	NewBinder(transport)
	defer func() {
		transport.Close()
		time.Sleep(100 * time.Millisecond)
	}()

	testExposer(t, exposer)
}

func TestDefaultExposer(t *testing.T) {
	testExposer(t, DefaultExposer)
}

func TestDefaultExposerHideImpl(t *testing.T) {
	_, okR := DefaultExposer.(Recver)
	_, okC := DefaultExposer.(ChanEncoder)

	if okR {
		t.Errorf("DefaultPulisher SHOULD NOT implement TransportRecver")
	}

	if okC {
		t.Errorf("DefaultPulisher SHOULD NOT implement ChanEncoder")
	}
}
