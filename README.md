# netchan - Natural network channels for Go

[![Travis CI](https://img.shields.io/travis/billziss-gh/netchan.svg)](https://travis-ci.org/billziss-gh/netchan)
[![GoDoc](https://godoc.org/github.com/billziss-gh/netchan/netchan?status.svg)](https://godoc.org/github.com/billziss-gh/netchan/netchan)

Package netchan enables Go channels to be used over the network.
Messages sent over a channel on one machine will be received by a
channel of the same type on a different machine. This includes messages
that contain channels (i.e. it is possible to "marshal" channels using
this package).

There are two fundamental concepts in netchan: "publishing" and
"connecting". A channel that is published, becomes associated with a
public name ("ID") and available to receive messages. A channel on a
different machine may then be connected to the published channel.
Messages sent to the connected channel will be transported over a
network transport and will become available to be received by the
published channel. Effectively the two channels become the endpoints of
a unidirectional network link.

## Publishing a channel

In order to publish a channel under an ID the Publish() function must be
used; there is also an Unpublish() function to unpublish a channel. If
multiple channels are published under the same ID which channel(s)
receive a message depends on the ID. ID's that start with a '+'
character are considered "broadcast" ID's and messages sent to them are
delivered to all channels published under that ID. All other ID's are
considered "unicast" and messages sent to them are delivered to a single
channel published under that ID (determined using a pseudo-random
algorithm).

To receive publish errors one can publish error channels (of type chan
error) under the special broadcast ID "+err/". All such error channels
will receive transport errors, etc. [The special broadcast ID "+err/" is
local to the running process and cannot be remoted.]

## Connecting a channel

In order to connect a channel to an "address" the Connect() function
must be used; to disconnect the channel simply close the channel.
Addresses in this package depend on the underlying transport and take
the form of URI's. For the default TCP transport an address has the
syntax: tcp://HOST[:PORT]/ID

When using Connect() an error channel (of type chan error) may also be
specified. This error channel will receive transport errors, etc.
related to the connected channel.

## Marshaling

This package encodes/decodes messages using one of the following
builtin encoding formats. These builtin formats can also be used to
encode channels into references that can later be decoded and
reconstructed on a different machine.

- netgob: extension of the standard gob format that also allows for
channels to be encoded/decoded (https://github.com/billziss-gh/netgob)
- netjson: extension of the standard json format that also allows for
channels to be encoded/decoded (https://github.com/billziss-gh/netjson)

Channels that are marshaled in this way are also implicitly published
and connected. When a message that is being sent contains a channel, a
reference is computed for that channel and the channel is implicitly
published under that reference. When the message arrives at the target
machine the reference gets decoded and a new channel is constructed and
implicitly connected back to the marshaled channel.

It is now possible to use the implicitly connected channel to send
messages back to the marshaled and implicitly published channel.
Implicitly published channels that are no longer in use will be
eventually garbage collected. Implicitly connected channels must be
closed when they will no longer be used for communication.

## Transports

This package comes with a number of builtin transports:

- tcp: plain TCP transport
- tls: secure TLS (SSL) transport
- http: sockets over HTTP (similar to net/rpc protocol)
- https: sockets over HTTPS (similar to net/rpc protocol)
- ws: (optional) WebSocket transport
- wss: (optional) secure WebSocket transport

It is possible to add transports by implementing the Transport and
Link interfaces.

## Example

```go
package main

import (
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/billziss-gh/netchan/netchan"
)

func ping(wg *sync.WaitGroup, count int) {
	defer wg.Done()

	pingch := make(chan chan struct{})
	errch := make(chan error, 1)
	err := netchan.Connect("tcp://127.0.0.1/pingpong", pingch, errch)
	if nil != err {
		panic(err)
	}

	for i := 0; count > i; i++ {
		// send a new pong (response) channel
		pongch := make(chan struct{})
		pingch <- pongch

		fmt.Println("ping")

		// wait for pong response, error or timeout
		select {
		case <-pongch:
		case err = <-errch:
			panic(err)
		case <-time.After(10 * time.Second):
			err = errors.New("timeout")
			panic(err)
		}
	}

	pingch <- nil

	close(pingch)
}

func pong(wg *sync.WaitGroup, published chan struct{}) {
	defer wg.Done()

	pingch := make(chan chan struct{})
	err := netchan.Publish("pingpong", pingch)
	if nil != err {
		panic(err)
	}

	close(published)

	for {
		// receive the pong (response) channel
		pongch := <-pingch
		if nil == pongch {
			fmt.Println("END")
			break
		}

		fmt.Println("pong")

		// send the pong response
		pongch <- struct{}{}
	}

	netchan.Unpublish("pingpong", pingch)
}

func main() {
	wg := &sync.WaitGroup{}

	published := make(chan struct{})
	wg.Add(1)
	go pong(wg, published)
	<-published

	wg.Add(1)
	go ping(wg, 10)

	wg.Wait()

	// Output:
	// ping
	// pong
	// ping
	// pong
	// ping
	// pong
	// ping
	// pong
	// ping
	// pong
	// ping
	// pong
	// ping
	// pong
	// ping
	// pong
	// ping
	// pong
	// ping
	// pong
	// END
}
```
