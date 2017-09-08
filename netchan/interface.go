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
// will receive transport errors, etc.
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

// Err is implemented by errors reported by this package.
type Err interface {
	error

	// Nested returns the original error that is the cause of this error.
	// May be nil.
	Nested() error

	// Chan returns a channel that is associated with this error.
	// May be nil.
	Chan() interface{}
}

// Publisher is used to publish and unpublish channels.
type Publisher interface {
	// Publish publishes a channel under an ID. Publishing a channel
	// associates it with the ID and makes it available to receive
	// messages.
	//
	// If multiple channels are published under the same ID which
	// channel(s) receive a message depends on the ID. ID's that start
	// with a '+' character are considered "broadcast" ID's and messages
	// sent to them are delivered to all channels published under that
	// ID. All other ID's are considered "unicast" and messages sent to
	// them are delivered to a single channel published under that ID
	// (determined using a pseudo-random algorithm).
	//
	// To receive publish errors one can publish error channels (of type
	// chan error) under the special broadcast ID "+err/". All such error
	// channels will receive transport errors, etc. This special broadcast
	// ID is local to the running process and cannot be accessed remotely.
	Publish(id string, ichan interface{}) error

	// Unpublish unpublishes a channel. It disassociates it from the ID
	// and makes it unavailable to receive messages under that ID.
	Unpublish(id string, ichan interface{})
}

// Connector is used to connect a local channel to a remotely published
// channel.
type Connector interface {
	// Connect connects a local channel to a remotely published channel.
	// After the connection is established, the connected channel may be
	// used to send messages to the remote channel.
	//
	// Remotely published channels may be addressed by URI's. The URI
	// syntax depends on the underlying transport. For the default TCP
	// transport an address has the syntax: tcp://HOST[:PORT]/ID
	//
	// The uri parameter contains the URI and can be of type string or
	// *url.URL. An error channel (of type chan error) may also be
	// specified. This error channel will receive transport errors, etc.
	// related to the connected channel.
	//
	// It is also possible to associate a new error channel with an
	// already connected channel. For this purpose use a nil uri and
	// the new error channel to associate with the connected channel.
	//
	// To disconnect a connected channel simply close it.
	Connect(uri interface{}, ichan interface{}, echan chan error) error
}

// Link encapsulates a network transport link between two machines.
// It is used to send and receive messages between machines.
//
// Link is useful to users implementing a new Transport.
type Link interface {
	Open()
	Recv() (id string, vmsg reflect.Value, err error)
	Send(id string, vmsg reflect.Value) (err error)
}

// ChanEncoder is used to encode a channel as a marshaling reference.
//
// ChanEncoder is useful to users implementing a new Marshaler.
type ChanEncoder interface {
	ChanEncode(link Link, ichan interface{}) ([]byte, error)
}

// ChanDecoder is used to decode a marshaling reference into a channel.
//
// ChanDecoder is useful to users implementing a new Marshaler.
type ChanDecoder interface {
	ChanDecode(link Link, ichan interface{}, buf []byte) error
}

// Transport is used to transport messages over a network.
//
// Transport is useful to users implementing a new network transport.
type Transport interface {
	SetChanEncoder(chanEnc ChanEncoder)
	SetChanDecoder(chanDec ChanDecoder)
	SetRecver(func(link Link) error)
	SetSender(func(link Link) error)
	Listen() error
	Connect(uri *url.URL) (id string, link Link, err error)
	Close()
}

// Marshaler is used to encode/decode messages for transporting over a
// network.
//
// Marshaler is useful to users implementing a new marshaling layer.
type Marshaler interface {
	RegisterType(val interface{})
	SetChanEncoder(chanEnc ChanEncoder)
	SetChanDecoder(chanDec ChanDecoder)
	Marshal(link Link, id string, vmsg reflect.Value) (buf []byte, err error)
	Unmarshal(link Link, buf []byte) (id string, vmsg reflect.Value, err error)
}
