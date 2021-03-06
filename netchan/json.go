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
	"bytes"
	"reflect"
	"sync"

	"github.com/billziss-gh/netjson/json"
)

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
	jsonRegisterType(val)
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

	wrt := &bytes.Buffer{}
	wrt.Write(make([]byte, hdrlen))
	enc := json.NewEncoder(wrt)
	accum := make(map[string]reflect.Value)
	enc.SetNetjsonEncoder(&jsonMarshalerNetjsonEncoder{self.chanEnc, link, accum})

	err = enc.Encode(id)
	if nil != err {
		err = MakeErrMarshaler(err)
		return
	}

	jsonTypeMux.RLock()
	nam := jsonTypeToNameMap[vmsg.Type()]
	jsonTypeMux.RUnlock()

	err = enc.Encode(nam)
	if nil != err {
		err = MakeErrMarshaler(err)
		return
	}

	msg := vmsg.Interface()
	err = enc.Encode(msg)
	if nil != err {
		err = MakeErrMarshaler(err)
		return
	}

	if 0 < len(accum) {
		err = self.chanEnc.ChanEncodeAccum(link, accum)
		if nil != err {
			err = MakeErrMarshaler(err)
			return
		}
	}

	buf = wrt.Bytes()
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

	rdr := bytes.NewBuffer(buf[hdrlen:])
	dec := json.NewDecoder(rdr)
	accum := make(map[string]reflect.Value)
	dec.SetNetjsonDecoder(&jsonMarshalerNetjsonDecoder{self.chanDec, link, accum})

	err = dec.Decode(&id)
	if nil != err {
		err = MakeErrMarshaler(err)
		return
	}

	nam := ""
	err = dec.Decode(&nam)
	if nil != err {
		err = MakeErrMarshaler(err)
		return
	}

	jsonTypeMux.RLock()
	typ, ok := jsonNameToTypeMap[nam]
	jsonTypeMux.RUnlock()

	var msg, val interface{}
	if ok {
		msg = reflect.New(typ).Interface()
	} else {
		msg = &val
	}
	err = dec.Decode(msg)
	if nil != err {
		id = ""
		vmsg = reflect.Value{}
		err = MakeErrMarshaler(err)
		return
	}

	if 0 < len(accum) {
		err = self.chanDec.ChanDecodeAccum(link, accum)
		if nil != err {
			id = ""
			vmsg = reflect.Value{}
			err = MakeErrMarshaler(err)
			return
		}
	}

	vmsg = reflect.ValueOf(msg).Elem()
	return
}

func jsonRegisterType(val interface{}) {
	typ := reflect.TypeOf(val)
	nam := typ.String()

	jsonTypeMux.Lock()
	defer jsonTypeMux.Unlock()

	jsonNameToTypeMap[nam] = typ
	jsonTypeToNameMap[typ] = nam
}

func jsonRegisterBasicTypes() int {
	jsonRegisterType((chan byte)(nil))
	jsonRegisterType((chan int)(nil))
	jsonRegisterType((chan int8)(nil))
	jsonRegisterType((chan int16)(nil))
	jsonRegisterType((chan int32)(nil))
	jsonRegisterType((chan int64)(nil))
	jsonRegisterType((chan uint)(nil))
	jsonRegisterType((chan uint8)(nil))
	jsonRegisterType((chan uint16)(nil))
	jsonRegisterType((chan uint32)(nil))
	jsonRegisterType((chan uint64)(nil))
	jsonRegisterType((chan float32)(nil))
	jsonRegisterType((chan float64)(nil))
	jsonRegisterType((chan complex64)(nil))
	jsonRegisterType((chan complex128)(nil))
	jsonRegisterType((chan uintptr)(nil))
	jsonRegisterType((chan bool)(nil))
	jsonRegisterType((chan string)(nil))
	jsonRegisterType(struct{}{})
	jsonRegisterType((chan struct{})(nil))
	jsonRegisterType((chan interface{})(nil))
	jsonRegisterType((chan error)(nil))

	return 0
}

var (
	jsonTypeMux       sync.RWMutex
	jsonNameToTypeMap = make(map[string]reflect.Type)
	jsonTypeToNameMap = make(map[reflect.Type]string)
	_                 = jsonRegisterBasicTypes()
)

type jsonMarshalerNetjsonEncoder struct {
	chanEnc ChanEncoder
	link    Link
	accum   map[string]reflect.Value
}

func (self *jsonMarshalerNetjsonEncoder) NetjsonEncode(v reflect.Value) ([]byte, error) {
	if nil == self.chanEnc {
		return nil, ErrMarshalerNoChanEncoder
	}
	return self.chanEnc.ChanEncode(self.link, v, self.accum)
}

type jsonMarshalerNetjsonDecoder struct {
	chanDec ChanDecoder
	link    Link
	accum   map[string]reflect.Value
}

func (self *jsonMarshalerNetjsonDecoder) NetjsonDecode(v reflect.Value, buf []byte) error {
	if nil == self.chanDec {
		return ErrMarshalerNoChanDecoder
	}
	return self.chanDec.ChanDecode(self.link, v, buf, self.accum)
}

var _ Marshaler = (*jsonMarshaler)(nil)
