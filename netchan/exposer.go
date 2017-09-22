/*
 * exposer.go
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

type expinfo struct {
	vlist []reflect.Value
}

type exposer struct {
	transport Transport
	expmux    sync.RWMutex
	expmap    map[string]expinfo
	wchanmap  *weakmap

	// monitored statistics
	statRecv, statRecvInv, statRecvErr uint32
}

type exposerImpl exposer

// NewExposer creates a new Exposer that can be used to expose channels.
// It is usually sufficient to use the DefaultExposer instead.
func NewExposer(transport Transport) Exposer {
	self := &exposer{
		transport: transport,
		expmap:    make(map[string]expinfo),
		wchanmap:  newWeakmap(),
	}
	transport.SetRecver((*exposerImpl)(self))
	return self
}

func (self *exposer) Expose(id string, ichan interface{}) error {
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

	self.expmux.Lock()
	defer self.expmux.Unlock()

	info := self.expmap[id]
	found := false
	for _, v := range info.vlist {
		if v == vchan {
			found = true
			break
		}
	}

	if !found {
		info.vlist = append(info.vlist, vchan)
		self.expmap[id] = info
	}

	return nil
}

func (self *exposer) Unexpose(id string, ichan interface{}) {
	vchan := reflect.ValueOf(ichan)
	if reflect.Chan != vchan.Kind() || 0 == vchan.Type().ChanDir()&reflect.SendDir {
		panic(ErrArgumentInvalid)
	}

	self.expmux.Lock()
	defer self.expmux.Unlock()

	info := self.expmap[id]
	for i, v := range info.vlist {
		if v == vchan {
			info.vlist = append(info.vlist[:i], info.vlist[i+1:]...)

			if 0 == len(info.vlist) {
				delete(self.expmap, id)
			} else {
				self.expmap[id] = info
			}

			return
		}
	}
}

func (impl *exposerImpl) Recver(link Link) error {
	self := (*exposer)(impl)
	pubrnd := rand.New(rand.NewSource(time.Now().UnixNano()))

	for {
		if nil != msgDebugLog {
			msgDebugLog("%v Recv()", link)
		}
		id, vmsg, err := link.Recv()
		if nil != msgDebugLog {
			msgDebugLog("%v Recv = (id=%#v, vmsg=%#v, err=%#v)", link, id, vmsg, err)
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

			self.deliver(IdErr, reflect.ValueOf(MakeErrExposer(err)), pubrnd)
			if _, ok := err.(*ErrTransport); ok {
				return err
			}
		}
	}
}

func (self *exposer) deliver(id string, vmsg reflect.Value, pubrnd *rand.Rand) (success bool) {
	var vlist []reflect.Value
	if w, ok := refDecode(id); ok {
		ichan := self.wchanmap.strongref(w, nil)
		if nil != ichan {
			vlist = append(vlist, reflect.ValueOf(ichan))
		}
	} else {
		// make a copy so that we can safely use it outside the read lock
		self.expmux.RLock()
		vlist = append(vlist, self.expmap[id].vlist...)
		self.expmux.RUnlock()
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

func (impl *exposerImpl) ChanEncode(link Link,
	vchan reflect.Value, accum map[string]reflect.Value) ([]byte, error) {
	self := (*exposer)(impl)
	w := self.wchanmap.weakref(vchan.Interface())
	return w[:], nil
}

func (impl *exposerImpl) ChanEncodeAccum(link Link,
	accum map[string]reflect.Value) error {
	return nil
}

func (self *exposer) StatNames() []string {
	return []string{"Recv", "RecvInv", "RecvErr"}
}

func (self *exposer) Stat(name string) float64 {
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

// DefaultExposer is the default exposer of the running process.
// Instead of DefaultExposer you can use the Expose and Unexpose
// functions.
var DefaultExposer Exposer = NewExposer(DefaultTransport)

// IdErr contains the special error broadcast ID. A channel (of type
// chan error) exposed under this ID will receive exposer errors.
// This special broadcast ID is local to the running process and
// cannot be accessed remotely.
var IdErr = "+err/"

// IdInv contains the special invalid message broadcast ID. A channel
// (of type chan Message) exposed under this ID will receive invalid
// messages. Invalid messages are messages that cannot be delivered to
// an exposed channel for any of a number of reasons: because they
// contain the wrong message ID, because their payload is the wrong type,
// because the destination channels have been closed, etc. This special
// broadcast ID is local to the running process and cannot be accessed
// remotely.
var IdInv = "+inv/"

var strBroadcast = "+"
var errType = reflect.TypeOf((*error)(nil)).Elem()
var msgType = reflect.TypeOf(Message{})

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
//
// Expose exposes a channel using the DefaultExposer.
func Expose(id string, ichan interface{}) error {
	return DefaultExposer.Expose(id, ichan)
}

// Unexpose unexposes a channel. It disassociates it from the ID
// and makes it unavailable to receive messages under that ID.
//
// Unexpose unexposes a channel using the DefaultExposer.
func Unexpose(id string, ichan interface{}) {
	DefaultExposer.Unexpose(id, ichan)
}
