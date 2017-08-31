/*
 * connector.go
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
	"sync"
)

type coninfo struct {
	slist []reflect.SelectCase
	ilist []string
	elist []chan error
}

type connector struct {
	transport Transport
	conmux    sync.RWMutex
	conmap    map[Link]coninfo
}

func newConnector(transport Transport) *connector {
	self := &connector{
		transport: transport,
		conmap:    make(map[Link]coninfo),
	}
	transport.SetSender(self.sender)
	return self
}

func (self *connector) Connect(uri *url.URL, ichan interface{}, echan chan error) error {
	vchan := reflect.ValueOf(ichan)
	if reflect.Chan != vchan.Kind() || 0 == vchan.Type().ChanDir()&reflect.SendDir {
		panic(ErrArgumentInvalid)
	}

	id, link, err := self.transport.Connect(uri)
	if nil != err {
		return err
	}

	// It is a programmatic error to Connect the same channel multiple times.

	self.conmux.Lock()
	defer self.conmux.Unlock()

	info := self.conmap[link]
	found := false
	for _, s := range info.slist {
		if s.Chan == vchan {
			found = true
			break
		}
	}

	if !found {
		if nil == info.slist {
			info.slist = append(info.slist,
				reflect.SelectCase{
					Dir:  reflect.SelectRecv,
					Chan: reflect.ValueOf(make(chan struct{}, 0x7fffffff)),
				})
			info.ilist = append(info.ilist, "")
			info.elist = append(info.elist, nil)
		}

		info.slist = append(info.slist,
			reflect.SelectCase{Dir: reflect.SelectRecv, Chan: vchan})
		info.ilist = append(info.ilist, id)
		info.elist = append(info.elist, echan)

		self.conmap[link] = info
		info.slist[0].Chan.Send(reflect.ValueOf(struct{}{}))

		link.Open()
	}

	return nil
}

func (self *connector) disconnect(link Link, vchan reflect.Value) {
	self.conmux.Lock()
	defer self.conmux.Unlock()

	info := self.conmap[link]
	for i, s := range info.slist {
		if s.Chan == vchan {
			info.slist = append(info.slist[:i], info.slist[i+1:]...)
			info.ilist = append(info.ilist[:i], info.ilist[i+1:]...)
			info.elist = append(info.elist[:i], info.elist[i+1:]...)

			self.conmap[link] = info

			return
		}
	}
}

func (self *connector) sender(link Link) error {
outer:
	for {
		// make a copy so that we can safely use it outside the read lock
		self.conmux.RLock()
		info := self.conmap[link]
		slist := append([]reflect.SelectCase(nil), info.slist...)
		ilist := append([]string(nil), info.ilist...)
		elist := append([]chan error(nil), info.elist...)
		self.conmux.RUnlock()

		for {
			i, vmsg, ok := reflect.Select(slist)
			if 0 == i {
				continue outer
			}
			if !ok {
				self.disconnect(link, slist[i].Chan)
				continue outer
			}

			err := link.Send(ilist[i], vmsg)
			if nil != err {
				errmap := make(map[chan error]int)
				for _, echan := range elist {
					if nil != echan {
						errmap[echan]++
					}
				}

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
			}
		}
	}
}

var DefaultConnector Connector = newConnector(DefaultTransport)

func Connect(uri *url.URL, ichan interface{}, echan chan error) error {
	return DefaultConnector.Connect(uri, ichan, echan)
}
