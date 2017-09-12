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
	"sync/atomic"
)

func echansend(echan chan error, err error) (ok bool) {
	defer func() {
		recover()
	}()
	echan <- err
	ok = true
	return
}

type coninfo struct {
	slist []reflect.SelectCase
	ilist []string
	elist []chan error
}

type connector struct {
	transport Transport
	conmux    sync.RWMutex
	conmap    map[Link]coninfo
	lnkmap    map[interface{}]Link

	// monitored statistics
	statSend, statSendErr uint32
}

// NewConnector creates a new Connector that can be used to connect
// channels. It is usually sufficient to use the DefaultConnector instead.
func NewConnector(transport Transport) Connector {
	self := &connector{
		transport: transport,
		conmap:    make(map[Link]coninfo),
		lnkmap:    make(map[interface{}]Link),
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
			return NewErrArgument(err)
		}
	case *url.URL:
		uri = u
	case nil:
		uri = nil
	default:
		panic(ErrArgumentInvalid)
	}

	var id string
	var link Link
	if nil != uri {
		var err error
		err = self.transport.Listen()
		if nil != err {
			return err
		}

		id, link, err = self.transport.Connect(uri)
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
	}

	return self.connect(id, link, vchan, echan)
}

func (self *connector) connect(id string, link Link, vchan reflect.Value, echan chan error) error {
	self.conmux.Lock()
	defer self.conmux.Unlock()

	ichan := vchan.Interface()
	oldlink, ok := self.lnkmap[ichan]
	if !ok {
		if nil == link {
			return ErrArgumentNotConnected
		}
	} else {
		if nil != link {
			return ErrArgumentConnected
		}

		link = oldlink
	}

	info := self.conmap[link]
	found := false
	for i, s := range info.slist {
		if s.Chan == vchan {
			found = true
			info.elist[i] = echan
			break
		}
	}

	if !found {
		self.addsigchan(&info)

		info.slist = append(info.slist,
			reflect.SelectCase{Dir: reflect.SelectRecv, Chan: vchan})
		info.ilist = append(info.ilist, id)
		info.elist = append(info.elist, echan)

		self.conmap[link] = info
		info.slist[0].Chan.Send(reflect.ValueOf(struct{}{}))

		link.Open()
	} else {
		self.conmap[link] = info
		info.slist[0].Chan.Send(reflect.ValueOf(struct{}{}))
	}

	self.lnkmap[ichan] = link

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

func (self *connector) addsigchan(info *coninfo) {
	if nil == info.slist {
		info.slist = append(info.slist,
			reflect.SelectCase{
				Dir:  reflect.SelectRecv,
				Chan: reflect.ValueOf(make(chan struct{}, 0x7fffffff)),
			})
		info.ilist = append(info.ilist, "")
		info.elist = append(info.elist, nil)
	}
}

func (self *connector) sender(link Link) error {
	self.conmux.Lock()
	info := self.conmap[link]
	self.addsigchan(&info)
	self.conmap[link] = info
	self.conmux.Unlock()

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
				sigchan := slist[0].Chan.Interface().(chan struct{})
			drain:
				for {
					select {
					case <-sigchan:
					default:
						break drain
					}
				}
				continue outer
			}
			if !ok {
				self.disconnect(link, slist[i].Chan)
				continue outer
			}

			if nil != debugLog {
				debugLog("%v Send(id=%#v, vmsg=%#v)", link, ilist[i], vmsg)
			}
			err := link.Send(ilist[i], vmsg)
			if nil != debugLog {
				debugLog("%v Send = (err=%#v)", link, err)
			}
			if nil == err {
				atomic.AddUint32(&self.statSend, 1)
			} else {
				atomic.AddUint32(&self.statSendErr, 1)

				if nil != elist[i] {
					if e, ok := err.(errArgs); ok {
						e.args(slist[i].Chan.Interface())
					}

					echansend(elist[i], err)
				}

				if _, ok := err.(*ErrTransport); ok {
					return err
				}
			}
		}
	}
}

func (self *connector) ChanDecode(link Link, ichan interface{}, buf []byte) error {
	v := reflect.ValueOf(ichan).Elem()

	var w weakref
	copy(w[:], buf)

	if (weakref{}) != w {
		u := reflect.MakeChan(v.Type(), 1)
		v.Set(u)

		id := refEncode(w)
		self.connect(id, link, u, nil)
	} else {
		u := reflect.Zero(v.Type())
		v.Set(u)
	}

	return nil
}

func (self *connector) StatNames() []string {
	return []string{"Send", "SendErr"}
}

func (self *connector) Stat(name string) float64 {
	switch name {
	case "Send":
		return float64(atomic.LoadUint32(&self.statSend))
	case "SendErr":
		return float64(atomic.LoadUint32(&self.statSendErr))
	default:
		return 0
	}
}

// DefaultConnector is the default Connector of the running process.
// Instead of DefaultConnector you can use the Connect function.
var DefaultConnector Connector = NewConnector(DefaultTransport)

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
//
// Connect connects a channel using the DefaultConnector.
func Connect(uri interface{}, ichan interface{}, echan chan error) error {
	return DefaultConnector.Connect(uri, ichan, echan)
}
