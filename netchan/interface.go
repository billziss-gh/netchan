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

// Package netchan enables Go channels to be used over the network.
// Messages sent over a channel on one machine will be received by a
// channel of the same type on a different machine. This includes messages
// that contain channels (i.e. it is possible to "marshal" channels using
// this package).
//
// There are two fundamental concepts in netchan: "publishing" and
// "connecting". A channel that is published, becomes associated with a
// public name ("ID") and available to receive messages. A channel on a
// different machine may then be connected to the published channel.
// Messages sent to the connected channel will be transported over a
// network transport and will become available to be received by the
// published channel. Effectively the two channels become the endpoints of
// a unidirectional network link.
//
// Publishing a channel
//
// In order to publish a channel under an ID the Publish() function must be
// used; there is also an Unpublish() function to unpublish a channel. If
// multiple channels are published under the same ID which channel(s)
// receive a message depends on the ID. ID's that start with a '+'
// character are considered "broadcast" ID's and messages sent to them are
// delivered to all channels published under that ID. All other ID's are
// considered "unicast" and messages sent to them are delivered to a single
// channel published under that ID (determined using a pseudo-random
// algorithm).
//
// To receive publish errors one can publish error channels (of type chan
// error) under the special broadcast ID "+err/". All such error channels
// will receive transport errors, etc. [The special broadcast ID "+err/" is
// local to the running process and cannot be remoted.]
//
// Connecting a channel
//
// In order to connect a channel to an "address" the Connect() function
// must be used; to disconnect the channel simply close the channel.
// Addresses in this package depend on the underlying transport and take
// the form of URI's. For the default TCP transport an address has the
// syntax: tcp://HOST[:PORT]/ID
//
// When using Connect() an error channel (of type chan error) may also be
// specified. This error channel will receive transport errors, etc.
// related to the connected channel.
//
// Marshaling
//
// This package uses the netgob format in order to transport messages. The
// netgob format is implemented by the netgob package
// (https://github.com/billziss-gh/netgob) and is an extension of the
// standard gob format that also allows for channels to be encoded/decoded.
// This package takes advantage of the netgob functionality and encodes
// channels into references that can later be decoded and reconstructed on
// a different machine.
//
// Channels that are marshaled in this way are also implicitly published
// and connected. When a message that is being sent contains a channel, a
// reference is computed for that channel and the channel is implicitly
// published under that reference. When the message arrives at the target
// machine the reference gets decoded and a new channel is constructed and
// implicitly connected back to the marshaled channel.
//
// It is now possible to use the implicitly connected channel to send
// messages back to the marshaled and implicitly published channel.
// Implicitly published channels that are no longer in use will be
// eventually garbage collected. Implicitly connected channels must be
// closed when they will no longer be used for communication.
//
// Transports
//
// This package comes with a number of default transports: tcp, tls. It is
// possible to add transports by implementing the Transport and Link
// interfaces.
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

type ChanEncoder interface {
	ChanEncode(link Link, ichan interface{}) ([]byte, error)
}

type ChanDecoder interface {
	ChanDecode(link Link, ichan interface{}, buf []byte) error
}

// Transport is used to provide network transport support to publishers and connectors.
type Transport interface {
	SetChanEncoder(chanEnc ChanEncoder)
	SetChanDecoder(chanDec ChanDecoder)
	SetRecver(func(link Link) error)
	SetSender(func(link Link) error)
	Listen() error
	Connect(uri *url.URL) (id string, link Link, err error)
	Close()
}

// Marshaler is used to provide message encoding/decoding support to publishers and connectors.
type Marshaler interface {
	RegisterType(val interface{})
	SetChanEncoder(chanEnc ChanEncoder)
	SetChanDecoder(chanDec ChanDecoder)
	Marshal(link Link, id string, vmsg reflect.Value) (buf []byte, err error)
	Unmarshal(link Link, buf []byte) (id string, vmsg reflect.Value, err error)
}
