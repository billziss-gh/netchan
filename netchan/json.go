/*
 * json.go
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
	"encoding/json"
	"reflect"
)

type jsonMessage struct {
	Id  string      `json:"i"`
	Msg interface{} `json:"m"`
}

type jsonMarshaler struct {
	chanEnc ChanEncoder
	chanDec ChanDecoder
}

// NewJsonMarshaler creates a new Marshaler that uses the json format
// for encoding/decoding.
func NewJsonMarshaler() Marshaler {
	return &jsonMarshaler{}
}

func (self *jsonMarshaler) RegisterType(val interface{}) {
}

func (self *jsonMarshaler) SetChanEncoder(chanEnc ChanEncoder) {
	self.chanEnc = chanEnc
}

func (self *jsonMarshaler) SetChanDecoder(chanDec ChanDecoder) {
	self.chanDec = chanDec
}

func (self *jsonMarshaler) Marshal(
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

	jmsg := jsonMessage{id, vmsg.Interface()}
	buf, err = json.Marshal(jmsg)
	if nil != err {
		buf = nil
		err = MakeErrMarshaler(err)
		return
	}

	return
}

func (self *jsonMarshaler) Unmarshal(
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

	jmsg := jsonMessage{}
	err = json.Unmarshal(buf, &jmsg)
	if nil != err {
		err = MakeErrMarshaler(err)
		return
	}

	id = jmsg.Id
	vmsg = reflect.ValueOf(jmsg.Msg)

	return
}

var _ Marshaler = (*jsonMarshaler)(nil)
