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
	_ = error(MakeErrArgument()).(Err)
	_ = error(MakeErrPublisher()).(Err)
	_ = error(MakeErrBinder()).(Err)
	_ = error(MakeErrTransport()).(Err)
	_ = error(MakeErrMarshaler()).(Err)

	msg0 := ErrArgumentInvalid.(*ErrArgument).message
	if "netchan: argument: invalid" != msg0 {
		t.Errorf("incorrect error message: expect %v, got %v", "netchan: argument: invalid", msg0)
	}

	err := ErrArgumentInvalid
	msg := err.Error()
	if msg0 != msg {
		t.Errorf("incorrect error message: expect %v, got %v", msg0, msg)
	}

	err = MakeErrArgument(ErrArgumentInvalid)
	if ErrArgumentInvalid != err {
		t.Errorf("incorrect error: expect %#v, got %#v", ErrArgumentInvalid, err)
	}

	err = MakeErrTransport("hello")
	msg = err.Error()
	if "hello" != msg {
		t.Errorf("incorrect error message: expect %v, got %v", "hello", msg)
	}

	err = MakeErrTransport(ErrArgumentInvalid)
	msg = err.Error()
	if msg0 != msg {
		t.Errorf("incorrect error message: expect %v, got %v", msg0, msg)
	}

	err = MakeErrTransport("hello", ErrArgumentInvalid)
	msg = err.Error()
	if "hello" != msg {
		t.Errorf("incorrect error message: expect %v, got %v", "hello", msg)
	}
}
