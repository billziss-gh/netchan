/*
 * interface.go
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
	"net/url"
	"reflect"
)

type Err interface {
	error
	Nested() error
	Chan() interface{}
}

type Publisher interface {
	Publish(id string, ichan interface{}) error
	Unpublish(id string, ichan interface{})
}

type Connector interface {
	Connect(uri interface{}, ichan interface{}, echan chan error) error
}

type Link interface {
	Open()
	Recv() (id string, vmsg reflect.Value, err error)
	Send(id string, vmsg reflect.Value) (err error)
}

type Transport interface {
	SetRecver(func(link Link) error)
	SetSender(func(link Link) error)
	Listen() error
	Connect(uri *url.URL) (id string, link Link, err error)
	Close()
}

type Marshaler interface {
	Marshal(id string, vmsg reflect.Value) (buf []byte, err error)
	Unmarshal(buf []byte) (id string, vmsg reflect.Value, err error)
}
