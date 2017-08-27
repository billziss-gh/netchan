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
	"testing"
)

func TestPublisher(t *testing.T) {
	ichan := make(chan bool)
	echan := make(chan error)

	err := DefaultPublisher.Publish("one", ichan, echan)
	if nil != err {
		panic(err)
	}

	err = DefaultPublisher.Publish("one", ichan, echan)
	if nil != err {
		panic(err)
	}

	err = DefaultPublisher.Publish("two", ichan, echan)
	if nil != err {
		panic(err)
	}

	DefaultPublisher.Unpublish("one", ichan)
	DefaultPublisher.Unpublish("two", ichan)
}
