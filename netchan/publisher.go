/*
 * publisher.go
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
	"math/rand"
	"reflect"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

func vchansend(vchan reflect.Value, vmsg reflect.Value) (ok bool) {
	defer func() {
		recover()
	}()
	vchan.Send(vmsg)
	ok = true
	return
}

type pubinfo struct {
	vlist []reflect.Value
}

type publisher struct {
	transport Transport
	pubmux    sync.RWMutex
	pubmap    map[string]pubinfo
	wchanmap  *weakmap

	// monitored statistics
	statRecv, statRecvInv, statRecvErr uint32
}

// NewPublisher creates a new Publisher that can be used to publish
// channels. It is usually sufficient to use the DefaultPublisher instead.
func NewPublisher(transport Transport) Publisher {
	self := &publisher{
		transport: transport,
		pubmap:    make(map[string]pubinfo),
		wchanmap:  newWeakmap(),
	}
	transport.SetChanEncoder(self)
	transport.SetRecver(self.recver)
	return self
}

func (self *publisher) Publish(id string, ichan interface{}) error {
	vchan := reflect.ValueOf(ichan)
	if reflect.Chan != vchan.Kind() || 0 == vchan.Type().ChanDir()&reflect.SendDir {
		panic(ErrArgumentInvalid)
	}

	switch id {
	default:
		err := self.transport.Listen()
		if nil != err {
			return err
		}
	case IdErr:
		if errType != vchan.Type().Elem() {
			panic(ErrArgumentInvalid)
		}
	case IdInv:
		if msgType != vchan.Type().Elem() {
			panic(ErrArgumentInvalid)
		}
	}

	self.pubmux.Lock()
	defer self.pubmux.Unlock()

	info := self.pubmap[id]
	found := false
	for _, v := range info.vlist {
		if v == vchan {
			found = true
			break
		}
	}

	if !found {
		info.vlist = append(info.vlist, vchan)
		self.pubmap[id] = info
	}

	return nil
}

func (self *publisher) Unpublish(id string, ichan interface{}) {
	vchan := reflect.ValueOf(ichan)
	if reflect.Chan != vchan.Kind() || 0 == vchan.Type().ChanDir()&reflect.SendDir {
		panic(ErrArgumentInvalid)
	}

	self.pubmux.Lock()
	defer self.pubmux.Unlock()

	info := self.pubmap[id]
	for i, v := range info.vlist {
		if v == vchan {
			info.vlist = append(info.vlist[:i], info.vlist[i+1:]...)

			if 0 == len(info.vlist) {
				delete(self.pubmap, id)
			} else {
				self.pubmap[id] = info
			}

			return
		}
	}
}

func (self *publisher) recver(link Link) error {
	pubrnd := rand.New(rand.NewSource(time.Now().UnixNano()))

	for {
		if nil != debugLog {
			debugLog("%v Recv()", link)
		}
		id, vmsg, err := link.Recv()
		if nil != debugLog {
			debugLog("%v Recv = (id=%#v, vmsg=%#v, err=%#v)", link, id, vmsg, err)
		}

		if nil == err {
			atomic.AddUint32(&self.statRecv, 1)

			ok := self.deliver(id, vmsg, pubrnd)
			if !ok {
				atomic.AddUint32(&self.statRecvInv, 1)

				self.deliver(IdInv, reflect.ValueOf(Message{id, vmsg}), pubrnd)
			}
		} else {
			atomic.AddUint32(&self.statRecvErr, 1)

			self.deliver(IdErr, reflect.ValueOf(err), pubrnd)
			if _, ok := err.(*ErrTransport); ok {
				return err
			}
		}
	}
}

func (self *publisher) deliver(id string, vmsg reflect.Value, pubrnd *rand.Rand) (success bool) {
	var vlist []reflect.Value
	if w, ok := refDecode(id); ok {
		ichan := self.wchanmap.strongref(w, nil)
		if nil != ichan {
			vlist = append(vlist, reflect.ValueOf(ichan))
		}
	} else {
		// make a copy so that we can safely use it outside the read lock
		self.pubmux.RLock()
		vlist = append(vlist, self.pubmap[id].vlist...)
		self.pubmux.RUnlock()
	}

	if 0 < len(vlist) {
		index := pubrnd.Intn(len(vlist))
		broadcast := strings.HasPrefix(id, strBroadcast)

		for i := range vlist {
			if vchansend(vlist[(i+index)%len(vlist)], vmsg) {
				success = true
				if !broadcast {
					break
				}
			}
		}
	}

	return
}

func (self *publisher) ChanEncode(link Link, ichan interface{}) ([]byte, error) {
	w := self.wchanmap.weakref(ichan)
	return w[:], nil
}

func (self *publisher) StatNames() []string {
	return []string{"Recv", "RecvInv", "RecvErr"}
}

func (self *publisher) Stat(name string) float64 {
	switch name {
	case "Recv":
		return float64(atomic.LoadUint32(&self.statRecv))
	case "RecvInv":
		return float64(atomic.LoadUint32(&self.statRecvInv))
	case "RecvErr":
		return float64(atomic.LoadUint32(&self.statRecvErr))
	default:
		return 0
	}
}

// DefaultPublisher is the default Publisher of the running process.
// Instead of DefaultPublisher you can use the Publish and Unpublish
// functions.
var DefaultPublisher Publisher = NewPublisher(DefaultTransport)

// IdErr contains the special error broadcast ID. A channel (of type
// chan error) published under this ID will receive publisher errors.
// This special broadcast ID is local to the running process and
// cannot be accessed remotely.
var IdErr = "+err/"

// IdInv contains the special invalid message broadcast ID. A channel
// (of type chan Message) published under this ID will receive invalid
// messages. Invalid messages are messages that cannot be delivered to
// a published channel for any of a number of reasons: because they
// contain the wrong message ID, because their payload is the wrong type,
// because the destination channels have been closed, etc. This special
// broadcast ID is local to the running process and cannot be accessed
// remotely.
var IdInv = "+inv/"

var strBroadcast = "+"
var errType = reflect.TypeOf((*error)(nil)).Elem()
var msgType = reflect.TypeOf(Message{})

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
// To receive publisher errors one can publish error channels (of type
// chan error) under the special broadcast ID "+err/". All such error
// channels will receive transport errors, etc. This special broadcast
// ID is local to the running process and cannot be accessed remotely.
//
// It is also possible to receive "invalid" messages on channels (of
// type chan Message) published under the special broadcast ID
// "+inv/". Invalid messages are messages that cannot be delivered
// for any of a number of reasons: because they contain the wrong
// message ID, because their payload is the wrong type, because the
// destination channels have been closed, etc. As with "+err/" this
// special broadcast ID is local to the running process and cannot
// be accessed remotely.
//
// Publish publishes a channel with the DefaultPublisher.
func Publish(id string, ichan interface{}) error {
	return DefaultPublisher.Publish(id, ichan)
}

// Unpublish unpublishes a channel. It disassociates it from the ID
// and makes it unavailable to receive messages under that ID.
//
// Unpublish unpublishes a channel from the DefaultPublisher.
func Unpublish(id string, ichan interface{}) {
	DefaultPublisher.Unpublish(id, ichan)
}
