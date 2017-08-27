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

func TestConnector(t *testing.T) {
	ichan := make(chan bool)
	echan := make(chan error)

	err := DefaultConnector.Connect(
		&url.URL{
			Scheme: "tcp",
			Host:   "127.0.0.1",
			Path:   "one",
		},
		ichan,
		echan)
	if nil != err {
		panic(err)
	}

	close(ichan)

	time.Sleep(300 * time.Millisecond)
}
