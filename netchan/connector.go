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
	conset    map[interface{}]struct{}
}

func newConnector(transport Transport) *connector {
	self := &connector{
		transport: transport,
		conmap:    make(map[Link]coninfo),
		conset:    make(map[interface{}]struct{}),
	}
	transport.SetChanDecoder(self)
	transport.SetSender(self.sender)
	return self
}

func (self *connector) Connect(iuri interface{}, ichan interface{}, echan chan error) error {
	vchan := reflect.ValueOf(ichan)
	if reflect.Chan != vchan.Kind() || 0 == vchan.Type().ChanDir()&reflect.SendDir {
		panic(ErrArgumentInvalid)
	}

	var uri *url.URL
	var err error
	switch u := iuri.(type) {
	case string:
		uri, err = url.Parse(u)
		if nil != err {
			return newErrTransport(err)
		}
	case *url.URL:
		uri = u
	default:
		panic(ErrArgumentInvalid)
	}

	id, link, err := self.transport.Connect(uri)
	if nil != err {
		return err
	}

	/*
	 * At this point Transport.Connect() has allocated a link and may have allocated
	 * additional resources. Ideally we would not want the connect() operation below
	 * to fail. Unfortunately there is still the possibility of getting an already
	 * connected channel, which we must treat as an error.
	 *
	 * Although we could check for that possibility before the Transport.Connect()
	 * call, to do this without race conditions we would have to hold the conmux lock
	 * over Transport.Connect() which is a potentially slow call. So we choose the
	 * least worse evil, which is to sometimes fail the connect() operation after a
	 * successful Transport.Connect().
	 */

	return self.connect(id, link, vchan, echan)
}

func (self *connector) connect(id string, link Link, vchan reflect.Value, echan chan error) error {
	self.conmux.Lock()
	defer self.conmux.Unlock()

	ichan := vchan.Interface()
	_, ok := self.conset[ichan]
	if ok {
		return ErrArgumentChanConnected
	}

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

	self.conset[ichan] = struct{}{}

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
				if nil != elist[i] {
					if e, ok := err.(errArgs); ok {
						e.args(slist[i].Chan.Interface())
					}

					func() {
						defer recover()
						elist[i] <- err
					}()
				}

				if _, ok := err.(*ErrTransport); ok {
					return err
				}
			}
		}
	}
}

func (self *connector) ChanDecode(link Link, ichan interface{}, buf []byte) error {
	var w weakref
	copy(w[:], buf)
	id := RefEncode(w)

	v := reflect.ValueOf(ichan).Elem()
	u := reflect.MakeChan(v.Type(), 1)
	v.Set(u)

	/*
	 * connect() may only return "chan is already connected" error,
	 * which cannot happen in this scenario (because we always create
	 * new channels with reflect.MakeChan()).
	 */

	self.connect(id, link, u, nil)

	return nil
}

var DefaultConnector Connector = newConnector(DefaultTransport)

// Connect uses the DefaultConnector and
// connects a local channel to a remote channel that is addressed by uri. The uri
// depends on the underlying network transport and may contain addressing and id information.
//
// The uri may be of type string or *url.URL. An error channel (type: chan error) may be
// supplied as well; it will receive Connector network errors.
func Connect(uri interface{}, ichan interface{}, echan chan error) error {
	return DefaultConnector.Connect(uri, ichan, echan)
}
