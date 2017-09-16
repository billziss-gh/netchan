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
			return MakeErrArgument(err)
		}
	case *url.URL:
		uri = u
	case nil:
		uri = nil
	default:
		panic(ErrArgumentInvalid)
	}

	if nil == uri {
		return self.connect("", nil, vchan, echan)
	}

	err = self.transport.Listen()
	if nil != err {
		return err
	}

	id, link, err := self.transport.Connect(uri)
	if nil != err {
		return err
	}

	err = self.connect(id, link, vchan, echan)
	link.Dereference()
	return err
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
		info.slist = append(info.slist,
			reflect.SelectCase{Dir: reflect.SelectRecv, Chan: vchan})
		info.ilist = append(info.ilist, id)
		info.elist = append(info.elist, echan)

		self.lnkmap[ichan] = link
		self.conmap[link] = info
		link.Sigchan() <- struct{}{}

		link.Reference()
		link.Activate()
	} else {
		self.conmap[link] = info
		link.Sigchan() <- struct{}{}
	}

	return nil
}

func (self *connector) disconnect(link Link, vchan reflect.Value) {
	self.conmux.Lock()
	defer self.conmux.Unlock()

	info := self.conmap[link]
	for i, s := range info.slist {
		if s.Chan == vchan {
			link.Dereference()

			info.slist = append(info.slist[:i], info.slist[i+1:]...)
			info.ilist = append(info.ilist[:i], info.ilist[i+1:]...)
			info.elist = append(info.elist[:i], info.elist[i+1:]...)

			if 0 == len(info.slist) {
				delete(self.conmap, link)
			} else {
				self.conmap[link] = info
			}
			delete(self.lnkmap, vchan.Interface())

			return
		}
	}
}

func (self *connector) sender(link Link) error {
	sigchan := link.Sigchan()
	vsigsel := reflect.SelectCase{
		Dir:  reflect.SelectRecv,
		Chan: reflect.ValueOf(link.Sigchan()),
	}

outer:
	for {
		// make a copy so that we can safely use it outside the read lock
		self.conmux.RLock()
		info := self.conmap[link]
		slist := append([]reflect.SelectCase{vsigsel}, info.slist...)
		ilist := append([]string{""}, info.ilist...)
		elist := append([]chan error{nil}, info.elist...)
		self.conmux.RUnlock()

		for {
			i, vmsg, ok := reflect.Select(slist)
			if 0 == i {
			drain:
				for {
					select {
					case _, ok = <-sigchan:
						if !ok {
							return ErrTransportClosed
						}
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

			if nil != msgDebugLog {
				msgDebugLog("%v Send(id=%#v, vmsg=%#v)", link, ilist[i], vmsg)
			}
			err := link.Send(ilist[i], vmsg)
			if nil != msgDebugLog {
				msgDebugLog("%v Send = (err=%#v)", link, err)
			}
			if nil == err {
				atomic.AddUint32(&self.statSend, 1)
			} else {
				atomic.AddUint32(&self.statSendErr, 1)

				if nil != elist[i] {
					echansend(elist[i], MakeErrConnector(err, slist[i].Chan.Interface()))
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
