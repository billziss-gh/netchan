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
	"time"
)

type pubinfo struct {
	vlist []reflect.Value
}

type publisher struct {
	transport Transport
	pubmux    sync.RWMutex
	pubmap    map[string]pubinfo
	wchanmap  *weakmap
}

func newPublisher(transport Transport) *publisher {
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

	if IdErr != id {
		err := self.transport.Listen()
		if nil != err {
			return err
		}
	} else {
		if errType != vchan.Type().Elem() {
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
		id, vmsg, err := link.Recv()
		if nil != err {
			id, vmsg = IdErr, reflect.ValueOf(err)
		}

		var vlist []reflect.Value
		if w, ok := RefDecode(id); ok {
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
				ok := func() (ok bool) {
					defer recover()
					vlist[(i+index)%len(vlist)].Send(vmsg)
					ok = true
					return
				}()

				if !broadcast && ok {
					break
				}
			}
		}

		if nil != err {
			if _, ok := err.(*ErrTransport); ok {
				return err
			}
		}
	}
}

func (self *publisher) ChanEncode(link Link, ichan interface{}) ([]byte, error) {
	w := self.wchanmap.weakref(ichan)
	if (weakref{}) == w {
		return nil, ErrMarshalerRefInvalid
	}

	return w[:], nil
}

var DefaultPublisher Publisher = newPublisher(DefaultTransport)
var IdErr = "+err/"
var strBroadcast = "+"
var errType = reflect.TypeOf((*error)(nil)).Elem()

// Publish uses the DefaultPublisher and
// publishes a channel under an id. Multiple channels may be published under the same
// id. When a channel is published, it becomes publicly accessible and may receive messages
// over a network.
//
// Messages that target a specific id may be unicast (delivered to a single associated
// channel) or broadcast (delivered to all the associated channels). Id's that start with the
// character '+' are broadcast id's, all other id's are unicast id's.
//
// The special broadcast id IdErr may be used to publish an error channel (type: chan error)
// that will receive Publisher network errors.
func Publish(id string, ichan interface{}) error {
	return DefaultPublisher.Publish(id, ichan)
}

// Unpublish uses the DefaultPublisher and
// disassociates a channel from an id using the DefaultPublisher.
func Unpublish(id string, ichan interface{}) {
	DefaultPublisher.Unpublish(id, ichan)
}
