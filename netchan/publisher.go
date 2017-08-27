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
	elist []chan error
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

func (self *publisher) Publish(id string, ichan interface{}, echan chan error) error {
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
		info.elist = append(info.elist, echan)
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
			info.elist = append(info.elist[:i], info.elist[i+1:]...)
			break
		}
	}

	if 0 == len(info.vlist) {
		delete(self.pubmap, id)
	} else {
		self.pubmap[id] = info
	}
}

func (self *publisher) recver(link Link) error {
	pubrnd := rand.New(rand.NewSource(time.Now().UnixNano()))

	for {
		id, vmsg, err := link.Recv()
		if nil != err {
			// make a copy so that we can safely use it outside the read lock
			self.pubmux.RLock()
			errmap := make(map[chan error]int)
			for _, info := range self.pubmap {
				for _, echan := range info.elist {
					if nil != echan {
						errmap[echan]++
					}
				}
			}
			self.pubmux.RUnlock()

			for echan := range errmap {
				func() {
					defer recover()
					echan <- err
				}()
			}

			if _, ok := err.(*ErrTransport); !ok {
				continue
			}

			return err
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
