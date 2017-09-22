/*
 * binder_test.go
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

func testBinder(t *testing.T, binder Binder) {
	ichan := make(chan bool)

	err := binder.Bind(
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

	err = binder.Bind(
		&url.URL{
			Scheme: "tcp",
			Host:   "127.0.0.2",
			Path:   "one",
		},
		ichan,
		nil)
	if ErrBinderChanBound != err {
		t.Errorf("incorrect error: expect %v, got %v", ErrBinderChanBound, err)
	}

	err = binder.Bind(
		nil,
		ichan,
		make(chan error))
	if nil != err {
		panic(err)
	}

	err = binder.Bind(
		nil,
		make(chan int),
		make(chan error))
	if ErrBinderChanNotBound != err {
		t.Errorf("incorrect error: expect %v, got %v", ErrBinderChanNotBound, err)
	}

	close(ichan)
}

func TestBinder(t *testing.T) {
	marshaler := NewGobMarshaler()
	transport := NewNetTransport(
		marshaler,
		&url.URL{
			Scheme: "tcp",
			Host:   ":25000",
		},
		nil)
	NewExposer(transport)
	binder := NewBinder(transport)
	defer func() {
		transport.Close()
		time.Sleep(100 * time.Millisecond)
	}()

	testBinder(t, binder)
}

func TestDefaultBinder(t *testing.T) {
	testBinder(t, DefaultBinder)
}

func TestDefaultBinderHideImpl(t *testing.T) {
	_, okR := DefaultBinder.(Sender)
	_, okC := DefaultBinder.(ChanDecoder)

	if okR {
		t.Errorf("DefaultBinder SHOULD NOT implement TransportSender")
	}

	if okC {
		t.Errorf("DefaultBinder SHOULD NOT implement ChanDecoder")
	}
}
