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
	"bytes"
	"encoding/gob"
	"reflect"
	"testing"
)

func testMarshalerRoundtrip(t *testing.T, marshaler Marshaler, id0 string, msg0 interface{}) {
	vmsg0 := reflect.ValueOf(msg0)
	buf, err := marshaler.Marshal(id0, vmsg0)
	if nil != err {
		panic(err)
	}

	id, vmsg, err := marshaler.Unmarshal(buf)
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
}

type ichan chan int

var ichanInst ichan = make(chan int)

func (self ichan) GobEncode() ([]byte, error) {
	return []byte{42}, nil
}

func (self *ichan) GobDecode([]byte) error {
	*self = ichanInst
	return nil
}

func TestGobMarshaler(t *testing.T) {
	marshaler := newGobMarshaler()

	gob.Register(testData{})
	gob.Register(ichan(make(chan int)))

	testMarshalerRoundtrip(t, marshaler, "42", "fortytwo")

	td := testData{10, "ten"}
	testMarshalerRoundtrip(t, marshaler, "10ten", td)

	testMarshalerRoundtrip(t, marshaler, "ichan", ichanInst)
}

func TestRefMarshal(t *testing.T) {
	c0 := make(chan struct{})
	c1 := make(chan error)

	buf0, err := RefMarshal(c0)
	if nil != err {
		panic(err)
	}

	buf1, err := RefMarshal(c0)
	if nil != err {
		panic(err)
	}

	if !bytes.Equal(buf0, buf1) {
		t.Errorf("incorrect non-equal bytes")
	}

	buf1, err = RefMarshal(c1)
	if nil != err {
		panic(err)
	}

	d0, err := RefUnmarshal(buf0)
	if nil != err {
		panic(err)
	}

	d1, err := RefUnmarshal(buf1)
	if nil != err {
		panic(err)
	}

	if c0 != d0 {
		t.Errorf("incorrect marshal: expect %v, got %v", c0, d0)
	}

	if c1 != d1 {
		t.Errorf("incorrect marshal: expect %v, got %v", c1, d1)
	}
}
