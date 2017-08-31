/*
 * errors_test.go
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

func TestErr(t *testing.T) {
	_ = newErrArgument().(Err)
	_ = newErrTransport().(Err)
	_ = newErrMarshaler().(Err)

	_ = newErrArgument().(*ErrArgument)
	_ = newErrTransport().(*ErrTransport)
	_ = newErrMarshaler().(*ErrMarshaler)

	msg0 := ErrArgumentInvalid.(*ErrArgument).message
	if "argument is invalid" != msg0 {
		t.Errorf("incorrect error message: expect %v, got %v", "argument is invalid", msg0)
	}

	err := ErrArgumentInvalid
	msg := err.Error()
	if msg0 != msg {
		t.Errorf("incorrect error message: expect %v, got %v", msg0, msg)
	}

	err = newErrTransport("hello")
	msg = err.Error()
	if "hello" != msg {
		t.Errorf("incorrect error message: expect %v, got %v", "hello", msg)
	}

	err = newErrTransport(ErrArgumentInvalid)
	msg = err.Error()
	if msg0 != msg {
		t.Errorf("incorrect error message: expect %v, got %v", msg0, msg)
	}

	err = newErrTransport("hello", ErrArgumentInvalid)
	msg = err.Error()
	if "hello" != msg {
		t.Errorf("incorrect error message: expect %v, got %v", "hello", msg)
	}
}
