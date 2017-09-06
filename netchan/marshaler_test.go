/*
 * marshaler_test.go
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
	"reflect"
	"testing"
)

func testMarshalerRoundtrip(t *testing.T, marshaler Marshaler, id0 string, msg0 interface{}) {
	vmsg0 := reflect.ValueOf(msg0)
	buf, err := marshaler.Marshal(nil, id0, vmsg0)
	if nil != err {
		panic(err)
	}

	id, vmsg, err := marshaler.Unmarshal(nil, buf)
	if nil != err {
		panic(err)
	}
	msg := vmsg.Interface()

	if id0 != id {
		t.Errorf("incorrect id: expect %v, got %v", id0, id)
	}

	if msg0 != msg {
		t.Errorf("incorrect msg: expect %v, got %v", msg0, msg)
	}
}

type testData struct {
	I int
	S string
	C chan string
}

func TestGobMarshaler(t *testing.T) {
	marshaler := newGobMarshaler()

	marshaler.RegisterType(testData{})

	testMarshalerRoundtrip(t, marshaler, "42", "fortytwo")

	testMarshalerRoundtrip(t, marshaler, "ichan", make(chan struct{}))

	td := testData{10, "ten", make(chan string)}
	testMarshalerRoundtrip(t, marshaler, "10ten", td)
}
