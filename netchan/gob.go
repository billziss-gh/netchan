/*
 * gob.go
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
	"reflect"

	"github.com/billziss-gh/netgob/gob"
)

type gobMarshaler struct {
}

func newGobMarshaler() *gobMarshaler {
	return &gobMarshaler{}
}

func (self *gobMarshaler) RegisterType(val interface{}) {
	gob.Register(val)
}

func (self *gobMarshaler) Marshal(id string, vmsg reflect.Value) (buf []byte, err error) {
	defer func() {
		if r := recover(); nil != r {
			buf = nil
			err = ErrMarshalerPanic
		}
	}()

	wrt := &bytes.Buffer{}
	wrt.Write(make([]byte, 4))
	enc := gob.NewEncoder(wrt)
	enc.SetNetgobEncoder(self)

	err = enc.Encode(id)
	if nil != err {
		err = newErrMarshaler(err)
		return
	}

	msg := vmsg.Interface()
	err = enc.EncodeValue(reflect.ValueOf(&msg))
	if nil != err {
		err = newErrMarshaler(err)
		return
	}

	buf = wrt.Bytes()
	return
}

func (self *gobMarshaler) Unmarshal(buf []byte) (id string, vmsg reflect.Value, err error) {
	defer func() {
		if r := recover(); nil != r {
			id = ""
			vmsg = reflect.Value{}
			err = ErrMarshalerPanic
		}
	}()

	rdr := bytes.NewBuffer(buf)
	rdr.Read(make([]byte, 4))
	dec := gob.NewDecoder(rdr)
	dec.SetNetgobDecoder(self)

	err = dec.Decode(&id)
	if nil != err {
		err = newErrMarshaler(err)
		return
	}

	msg := interface{}(nil)
	err = dec.DecodeValue(reflect.ValueOf(&msg))
	if nil != err {
		id = ""
		vmsg = reflect.Value{}
		err = newErrMarshaler(err)
		return
	}

	vmsg = reflect.ValueOf(msg)
	return
}

func (self *gobMarshaler) NetgobEncode(i interface{}) ([]byte, error) {
	w := chanmap.weakref(i)
	if (weakref{}) == w {
		return nil, ErrMarshalerRef
	}

	return w[:], nil
}

func (self *gobMarshaler) NetgobDecode(i interface{}, buf []byte) error {
	v := reflect.ValueOf(i).Elem()

	var w weakref
	copy(w[:], buf)

	s := chanmap.strongref(w, func() interface{} {
		return reflect.MakeChan(v.Type(), 1).Interface()
	})
	if nil == s {
		return ErrMarshalerRef
	}

	v.Set(reflect.ValueOf(s))

	return nil
}

var _ Marshaler = (*gobMarshaler)(nil)
