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
	"errors"
	"reflect"
	"testing"
)

type testMarshalerCoder struct {
	chanmap *weakmap
}

func (self *testMarshalerCoder) ChanEncode(link Link,
	v reflect.Value, accum map[interface{}]reflect.Value) ([]byte, error) {
	w := self.chanmap.weakref(v.Interface())
	if (weakref{}) == w {
		return nil, errors.New("marshaler ref is invalid")
	}

	return w[:], nil
}

func (self *testMarshalerCoder) ChanEncodeAccum(link Link,
	accum map[interface{}]reflect.Value) error {
	return nil
}

func (self *testMarshalerCoder) ChanDecode(link Link,
	v reflect.Value, buf []byte, accum map[interface{}]reflect.Value) error {
	v = v.Elem()

	var w weakref
	copy(w[:], buf)

	s := self.chanmap.strongref(w, nil)
	if nil == s {
		return errors.New("marshaler ref is invalid")
	}

	v.Set(reflect.ValueOf(s))

	return nil
}

func (self *testMarshalerCoder) ChanDecodeAccum(link Link,
	accum map[interface{}]reflect.Value) error {
	return nil
}

func testMarshalerRoundtrip(t *testing.T, marshaler Marshaler, id0 string, msg0 interface{}) {
	vmsg0 := reflect.ValueOf(msg0)
	buf, err := marshaler.Marshal(nil, id0, vmsg0, netMsgHdrLen)
	if nil != err {
		panic(err)
	}

	id, vmsg, err := marshaler.Unmarshal(nil, buf, netMsgHdrLen)
	if nil != err {
		panic(err)
	}
	msg := vmsg.Interface()

	if id0 != id {
		t.Errorf("incorrect id: expect %v, got %v", id0, id)
	}

	if reflect.Ptr == vmsg0.Kind() {
		msg0 = vmsg0.Elem().Interface()
		msg = vmsg.Elem().Interface()
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

type testRefData struct {
	I int
	S string
	C chan string
}

func TestGobMarshaler(t *testing.T) {
	coder := &testMarshalerCoder{newWeakmap()}

	marshaler := NewGobMarshaler()
	marshaler.SetChanEncoder(coder)
	marshaler.SetChanDecoder(coder)

	marshaler.RegisterType(testData{})
	marshaler.RegisterType(&testRefData{})

	testMarshalerRoundtrip(t, marshaler, "42", "fortytwo")

	testMarshalerRoundtrip(t, marshaler, "ichan", make(chan struct{}))
	testMarshalerRoundtrip(t, marshaler, "ichan", make(chan error))

	td := testData{10, "ten", make(chan string)}
	testMarshalerRoundtrip(t, marshaler, "10ten", td)

	trd := testRefData{10, "ten", make(chan string)}
	testMarshalerRoundtrip(t, marshaler, "10ten", &trd)
}

func TestJsonMarshaler(t *testing.T) {
	coder := &testMarshalerCoder{newWeakmap()}

	marshaler := NewJsonMarshaler()
	marshaler.SetChanEncoder(coder)
	marshaler.SetChanDecoder(coder)

	marshaler.RegisterType(testData{})
	marshaler.RegisterType(&testRefData{})

	testMarshalerRoundtrip(t, marshaler, "42", "fortytwo")

	testMarshalerRoundtrip(t, marshaler, "ichan", make(chan struct{}))
	testMarshalerRoundtrip(t, marshaler, "ichan", make(chan error))

	td := testData{10, "ten", make(chan string)}
	testMarshalerRoundtrip(t, marshaler, "10ten", td)

	trd := testRefData{10, "ten", make(chan string)}
	testMarshalerRoundtrip(t, marshaler, "10ten", &trd)
}

func TestRefEncodeDecode(t *testing.T) {
	w0 := weakref{42, 43, 44}
	s := refEncode(w0)

	w, ok := refDecode(s)
	if !ok || w0 != w {
		t.Errorf("incorrect ref: expect %v, got %v", w0, w)
	}

	w, ok = refDecode(s[1:])
	if ok || w0 == w {
		t.Errorf("incorrect ref: expect !ok")
	}
}
