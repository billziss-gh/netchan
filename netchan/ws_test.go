// +build websocket

/*
 * ws_test.go
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

func TestWsTransport(t *testing.T) {
	marshaler := NewJsonMarshaler()
	transport := NewWsTransport(
		marshaler,
		&url.URL{
			Scheme: "ws",
			Host:   ":25000",
		},
		nil,
		nil)
	defer func() {
		transport.Close()
		time.Sleep(100 * time.Millisecond)
	}()

	testTransport(t, transport, "ws")
}
