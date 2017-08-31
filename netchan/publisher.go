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
}

func newPublisher(transport Transport) *publisher {
	self := &publisher{
		transport: transport,
		pubmap:    make(map[string]pubinfo),
	}
	transport.SetRecver(self.recver)
	return self
}

func (self *publisher) Publish(id string, ichan interface{}) error {
	vchan := reflect.ValueOf(ichan)
	if reflect.Chan != vchan.Kind() || 0 == vchan.Type().ChanDir()&reflect.SendDir {
		panic(ErrArgumentInvalid)
	}

	err := self.transport.Listen()
	if nil != err {
		return err
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
			if _, ok := err.(*ErrTransport); ok {
				return err
			}
		} else {
			// make a copy so that we can safely use it outside the read lock
			self.pubmux.RLock()
			vlist := append([]reflect.Value(nil), self.pubmap[id].vlist...)
			self.pubmux.RUnlock()

			if 0 < len(vlist) {
				index := pubrnd.Intn(len(vlist))

				for i := range vlist {
					ok := func() (ok bool) {
						defer recover()
						vlist[i+index].Send(vmsg)
						ok = true
						return
					}()

					if ok {
						break
					}
				}
			}
		}
	}
}

var DefaultPublisher Publisher = newPublisher(DefaultTransport)

func Publish(id string, ichan interface{}) error {
	return DefaultPublisher.Publish(id, ichan)
}

func Unpublish(id string, ichan interface{}) {
	DefaultPublisher.Unpublish(id, ichan)
}
