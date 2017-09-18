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
	chanEnc ChanEncoder
	chanDec ChanDecoder
}

// NewGobMarshaler creates a new Marshaler that uses the netgob format
// for encoding/decoding.
func NewGobMarshaler() Marshaler {
	return &gobMarshaler{}
}

func (self *gobMarshaler) RegisterType(val interface{}) {
	gob.Register(val)
}

func (self *gobMarshaler) SetChanEncoder(chanEnc ChanEncoder) {
	self.chanEnc = chanEnc
}

func (self *gobMarshaler) SetChanDecoder(chanDec ChanDecoder) {
	self.chanDec = chanDec
}

func (self *gobMarshaler) Marshal(
	link Link, id string, vmsg reflect.Value, hdrlen int) (buf []byte, err error) {
	defer func() {
		if r := recover(); nil != r {
			buf = nil
			if e, ok := r.(error); ok {
				err = MakeErrMarshaler(e)
			} else {
				err = ErrMarshalerPanic
			}
		}
	}()

	wrt := &bytes.Buffer{}
	wrt.Write(make([]byte, hdrlen))
	enc := gob.NewEncoder(wrt)
	enc.SetNetgobEncoder(&gobMarshalerNetgobEncoder{self.chanEnc, link})

	err = enc.Encode(id)
	if nil != err {
		err = MakeErrMarshaler(err)
		return
	}

	msg := vmsg.Interface()
	err = enc.EncodeValue(reflect.ValueOf(&msg))
	if nil != err {
		err = MakeErrMarshaler(err)
		return
	}

	buf = wrt.Bytes()
	return
}

func (self *gobMarshaler) Unmarshal(
	link Link, buf []byte, hdrlen int) (id string, vmsg reflect.Value, err error) {
	defer func() {
		if r := recover(); nil != r {
			id = ""
			vmsg = reflect.Value{}
			if e, ok := r.(error); ok {
				err = MakeErrMarshaler(e)
			} else {
				err = ErrMarshalerPanic
			}
		}
	}()

	rdr := bytes.NewBuffer(buf[hdrlen:])
	dec := gob.NewDecoder(rdr)
	dec.SetNetgobDecoder(&gobMarshalerNetgobDecoder{self.chanDec, link})

	err = dec.Decode(&id)
	if nil != err {
		err = MakeErrMarshaler(err)
		return
	}

	msg := interface{}(nil)
	err = dec.DecodeValue(reflect.ValueOf(&msg))
	if nil != err {
		id = ""
		vmsg = reflect.Value{}
		err = MakeErrMarshaler(err)
		return
	}

	vmsg = reflect.ValueOf(msg)
	return
}

type gobMarshalerNetgobEncoder struct {
	chanEnc ChanEncoder
	link    Link
}

func (self *gobMarshalerNetgobEncoder) NetgobEncode(i interface{}) ([]byte, error) {
	if nil == self.chanEnc {
		return nil, ErrMarshalerNoChanEncoder
	}
	return self.chanEnc.ChanEncode(self.link, i)
}

type gobMarshalerNetgobDecoder struct {
	chanDec ChanDecoder
	link    Link
}

func (self *gobMarshalerNetgobDecoder) NetgobDecode(i interface{}, buf []byte) error {
	if nil == self.chanDec {
		return ErrMarshalerNoChanDecoder
	}
	return self.chanDec.ChanDecode(self.link, i, buf)
}

var _ Marshaler = (*gobMarshaler)(nil)
