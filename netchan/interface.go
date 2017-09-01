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

// Err is implemented by all errors reported by this library.
type Err interface {
	error

	// Nested returns the original error that is the cause of this error. May be nil.
	Nested() error

	// Chan returns a channel that is associated with this error. May be nil.
	Chan() interface{}
}

// Publisher is used to associate a channel with an id and make the channel publicly accessible.
type Publisher interface {
	// Publish publishes a channel under an id. Multiple channels may be published under the same
	// id. When a channel is published, it becomes publicly accessible and may receive messages
	// over a network.
	//
	// Messages that target a specific id may be unicast (delivered to a single associated
	// channel) or broadcast (delivered to all the associated channels). Id's that start with the
	// character '+' are broadcast id's, all other id's are unicast id's.
	//
	// The special broadcast id IdErr may be used to publish an error channel (type: chan error)
	// that will receive Publisher network errors.
	Publish(id string, ichan interface{}) error

	// Unpublish disassociates a channel from an id.
	Unpublish(id string, ichan interface{})
}

// Connector is used to connect a local channel to a remote channel.
type Connector interface {
	// Connect connects a local channel to a remote channel that is addressed by uri. The uri
	// depends on the underlying network transport and may contain addressing and id information.
	//
	// The uri may be of type string or *url.URL. An error channel (type: chan error) may be
	// supplied as well; it will receive Connector network errors.
	Connect(uri interface{}, ichan interface{}, echan chan error) error
}

// Link is used to provide network transport support to publishers and connectors.
// A link encapsulates a network transport connection between two hosts.
type Link interface {
	Open()
	Recv() (id string, vmsg reflect.Value, err error)
	Send(id string, vmsg reflect.Value) (err error)
}

// Transport is used to provide network transport support to publishers and connectors.
type Transport interface {
	SetRecver(func(link Link) error)
	SetSender(func(link Link) error)
	Listen() error
	Connect(uri *url.URL) (id string, link Link, err error)
	Close()
}

// Marshaler is used to provide message encoding/decoding support to publishers and connectors.
type Marshaler interface {
	Marshal(id string, vmsg reflect.Value) (buf []byte, err error)
	Unmarshal(buf []byte) (id string, vmsg reflect.Value, err error)
}
