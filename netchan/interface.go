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
// There are two fundamental concepts in netchan: "exposing" and
// "binding". A channel that is exposed, becomes associated with a
// public name ("ID") and available to receive messages. A channel on a
// different machine may then be bound to the exposed channel.
// Messages sent to the bound channel will be transported over a
// network transport and will become available to be received by the
// exposed channel. Effectively the two channels become the endpoints of
// a unidirectional network link.
//
// Exposing a channel
//
// In order to expose a channel under an ID the Expose() function must be
// used; there is also an Unexpose() function to unexpose a channel. If
// multiple channels are exposed under the same ID which channel(s)
// receive a message depends on the ID. ID's that start with a '+'
// character are considered "broadcast" ID's and messages sent to them are
// delivered to all channels exposed under that ID. All other ID's are
// considered "unicast" and messages sent to them are delivered to a single
// channel exposed under that ID (determined using a pseudo-random
// algorithm).
//
// To receive exposer errors one can expose error channels (of type chan
// error) under the special broadcast ID "+err/". All such error channels
// will receive transport errors, etc.
//
// Binding a channel
//
// In order to bind a channel to an "address" the Bind() function must be
// used; to unbind the channel simply close it.
// Addresses in this package depend on the underlying transport and take
// the form of URI's. For the default TCP transport an address has the
// syntax: tcp://HOST[:PORT]/ID
//
// When using Bind() an error channel (of type chan error) may also be
// specified. This error channel will receive transport errors, etc.
// related to the bound channel.
//
// Marshaling
//
// This package encodes/decodes messages using one of the following
// builtin encoding formats. These builtin formats can also be used to
// encode channels into references that can later be decoded and
// reconstructed on a different machine.
//
//     netgob   extension of the standard gob format that also allows for
//              channels to be encoded/decoded
//              (https://github.com/billziss-gh/netgob)
//     netjson  extension of the standard json format that also allows for
//              channels to be encoded/decoded
//              (https://github.com/billziss-gh/netjson)
//
// Channels that are marshaled in this way are also implicitly exposed
// and bound. When a message that is being sent contains a channel, a
// reference is computed for that channel and the channel is implicitly
// exposed under that reference. When the message arrives at the target
// machine the reference gets decoded and a new channel is constructed and
// implicitly bound back to the marshaled channel.
//
// It is now possible to use the implicitly bound channel to send
// messages back to the marshaled and implicitly exposed channel.
// Implicitly exposed channels that are no longer in use will be
// eventually garbage collected. Implicitly bound channels must be
// closed when they will no longer be used for communication.
//
// Transports
//
// This package comes with a number of builtin transports:
//
//     tcp      plain TCP transport
//     tls      secure TLS (SSL) transport
//     http     sockets over HTTP (similar to net/rpc protocol)
//     https    sockets over HTTPS (similar to net/rpc protocol)
//     ws       (optional) WebSocket transport
//     wss      (optional) secure WebSocket transport
//
// It is possible to add transports by implementing the Transport and
// Link interfaces.
package netchan

import (
	"net/url"
	"reflect"
)

// Exposer is used to expose and unexpose channels.
type Exposer interface {
	// Expose exposes a channel under an ID. Exposing a channel
	// associates it with the ID and makes it available to receive
	// messages.
	//
	// If multiple channels are exposed under the same ID which
	// channel(s) receive a message depends on the ID. ID's that start
	// with a '+' character are considered "broadcast" ID's and messages
	// sent to them are delivered to all channels exposed under that
	// ID. All other ID's are considered "unicast" and messages sent to
	// them are delivered to a single channel exposed under that ID
	// (determined using a pseudo-random algorithm).
	//
	// To receive exposer errors one can expose error channels (of type
	// chan error) under the special broadcast ID "+err/". All such error
	// channels will receive transport errors, etc. This special broadcast
	// ID is local to the running process and cannot be accessed remotely.
	//
	// It is also possible to receive "invalid" messages on channels (of
	// type chan Message) exposed under the special broadcast ID
	// "+inv/". Invalid messages are messages that cannot be delivered
	// for any of a number of reasons: because they contain the wrong
	// message ID, because their payload is the wrong type, because the
	// destination channels have been closed, etc. As with "+err/" this
	// special broadcast ID is local to the running process and cannot
	// be accessed remotely.
	Expose(id string, ichan interface{}) error

	// Unexpose unexposes a channel. It disassociates it from the ID
	// and makes it unavailable to receive messages under that ID.
	Unexpose(id string, ichan interface{})
}

// Binder is used to bind a local channel to a remote channel.
type Binder interface {
	// Bind binds a local channel to a URI that is addressing a remote
	// channel. After the binding is established, the bound channel may
	// be used to send messages to the remote channel.
	//
	// Remotely exposed channels are addressed by URI's. The URI
	// syntax depends on the underlying transport. For the default TCP
	// transport an address has the syntax: tcp://HOST[:PORT]/ID
	//
	// The uri parameter contains the URI and can be of type string or
	// *url.URL. An error channel (of type chan error) may also be
	// specified. This error channel will receive transport errors, etc.
	// related to the bound channel.
	//
	// It is also possible to associate a new error channel with an
	// already bound channel. For this purpose use a nil uri and
	// the new error channel to associate with the bound channel.
	//
	// To unbind a bound channel simply close it.
	Bind(uri interface{}, ichan interface{}, echan chan error) error
}

// Stats is used to monitor the activity of a netchan component.
// Stats is a collection of values accessible by name; these values
// typically represent a count or ratio.
//
// Exposers and binders implement Stats to provide insights into
// their internal workings.
type Stats interface {
	// StatNames returns a list of value names.
	StatNames() []string

	// Stat returns the value associated with a particular name.
	Stat(name string) float64
}

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

// Message is a struct that contains a message ID and value.
type Message struct {
	// Id contains the ID of the intended recipient.
	Id string

	// Value contains the message payload (as decoded by the marshaling
	// layer).
	Value reflect.Value
}

// Link encapsulates a network transport link between two machines.
// It is used to send and receive messages between machines.
//
// Link is useful to users implementing a new Transport.
type Link interface {
	Sigchan() chan struct{}
	Reference()
	Dereference()
	Activate()
	Recv() (id string, vmsg reflect.Value, err error)
	Send(id string, vmsg reflect.Value) (err error)
}

// Recver is used to receive messages over a network transport.
//
// Recver is useful to users implementing a new Transport.
type Recver interface {
	Recver(link Link) error
}

// Sender is used to send messages over a network transport.
//
// Sender is useful to users implementing a new Transport.
type Sender interface {
	Sender(link Link) error
}

// Transport is used to transport messages over a network.
//
// Transport is useful to users implementing a new network transport.
type Transport interface {
	SetRecver(recver Recver)
	SetSender(sender Sender)
	Listen() error
	Dial(uri *url.URL) (id string, link Link, err error)
	Close()
}

// ChanEncoder is used to encode a channel as a marshaling reference.
//
// ChanEncoder is useful to users implementing a new Marshaler.
type ChanEncoder interface {
	ChanEncode(link Link, vchan reflect.Value, accum map[string]reflect.Value) ([]byte, error)
	ChanEncodeAccum(link Link, accum map[string]reflect.Value) error
}

// ChanDecoder is used to decode a marshaling reference into a channel.
//
// ChanDecoder is useful to users implementing a new Marshaler.
type ChanDecoder interface {
	ChanDecode(link Link, vchan reflect.Value, buf []byte, accum map[string]reflect.Value) error
	ChanDecodeAccum(link Link, accum map[string]reflect.Value) error
}

// Marshaler is used to encode/decode messages for transporting over a
// network.
//
// Marshaler is useful to users implementing a new marshaling layer.
type Marshaler interface {
	RegisterType(val interface{})
	SetChanEncoder(chanEnc ChanEncoder)
	SetChanDecoder(chanDec ChanDecoder)
	Marshal(link Link, id string, vmsg reflect.Value, hdrlen int) (buf []byte, err error)
	Unmarshal(link Link, buf []byte, hdrlen int) (id string, vmsg reflect.Value, err error)
}
