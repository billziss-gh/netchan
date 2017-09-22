/*
 * binder.go
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

type bndinfo struct {
	slist []reflect.SelectCase
	ilist []string
	elist []chan error
}

type binder struct {
	transport Transport
	bndmux    sync.RWMutex
	bndmap    map[Link]bndinfo
	lnkmap    map[interface{}]Link

	// monitored statistics
	statSend, statSendErr uint32
}

type binderImpl binder

// NewBinder creates a new binder that can be used to bind channels.
// It is usually sufficient to use the DefaultBinder instead.
func NewBinder(transport Transport) Binder {
	self := &binder{
		transport: transport,
		bndmap:    make(map[Link]bndinfo),
		lnkmap:    make(map[interface{}]Link),
	}
	transport.SetSender((*binderImpl)(self))
	return self
}

func (self *binder) Bind(iuri interface{}, ichan interface{}, echan chan error) error {
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
		return self.bind("", nil, vchan, echan)
	}

	id, link, err := self.transport.Dial(uri)
	if nil != err {
		return err
	}

	err = self.bind(id, link, vchan, echan)
	link.Dereference()
	return err
}

func (self *binder) bind(id string, link Link, vchan reflect.Value, echan chan error) error {
	self.bndmux.Lock()
	defer self.bndmux.Unlock()

	ichan := vchan.Interface()
	oldlink, ok := self.lnkmap[ichan]
	if !ok {
		if nil == link {
			return ErrBinderChanNotBound
		}
	} else {
		if nil != link {
			return ErrBinderChanBound
		}

		link = oldlink
	}

	info := self.bndmap[link]
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
		self.bndmap[link] = info
		link.Sigchan() <- struct{}{}

		link.Reference()
		link.Activate()
	} else {
		self.bndmap[link] = info
		link.Sigchan() <- struct{}{}
	}

	return nil
}

func (self *binder) unbind(link Link, vchan reflect.Value) {
	self.bndmux.Lock()
	defer self.bndmux.Unlock()

	info := self.bndmap[link]
	for i, s := range info.slist {
		if s.Chan == vchan {
			link.Dereference()

			info.slist = append(info.slist[:i], info.slist[i+1:]...)
			info.ilist = append(info.ilist[:i], info.ilist[i+1:]...)
			info.elist = append(info.elist[:i], info.elist[i+1:]...)

			if 0 == len(info.slist) {
				delete(self.bndmap, link)
			} else {
				self.bndmap[link] = info
			}
			delete(self.lnkmap, vchan.Interface())

			return
		}
	}
}

func (impl *binderImpl) Sender(link Link) error {
	self := (*binder)(impl)
	sigchan := link.Sigchan()
	vsigsel := reflect.SelectCase{
		Dir:  reflect.SelectRecv,
		Chan: reflect.ValueOf(link.Sigchan()),
	}

outer:
	for {
		// make a copy so that we can safely use it outside the read lock
		self.bndmux.RLock()
		info := self.bndmap[link]
		slist := append([]reflect.SelectCase{vsigsel}, info.slist...)
		ilist := append([]string{""}, info.ilist...)
		elist := append([]chan error{nil}, info.elist...)
		self.bndmux.RUnlock()

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
				self.unbind(link, slist[i].Chan)
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
					echansend(elist[i], MakeErrBinder(err, slist[i].Chan.Interface()))
				}

				if _, ok := err.(*ErrTransport); ok {
					return err
				}
			}
		}
	}
}

func (impl *binderImpl) ChanDecode(link Link,
	vchan reflect.Value, buf []byte, accum map[string]reflect.Value) error {
	vchan = vchan.Elem()

	var w weakref
	copy(w[:], buf)

	var c reflect.Value
	if (weakref{}) != w {
		var ok bool
		id := refEncode(w)
		c, ok = accum[id]
		if !ok {
			c = reflect.MakeChan(vchan.Type(), 1)
			accum[id] = c
		}
	} else {
		c = reflect.Zero(vchan.Type())
	}
	vchan.Set(c)

	return nil
}

func (impl *binderImpl) ChanDecodeAccum(link Link,
	accum map[string]reflect.Value) error {
	self := (*binder)(impl)
	for id, c := range accum {
		self.bind(id, link, c, nil)
	}

	return nil
}

func (self *binder) StatNames() []string {
	return []string{"Send", "SendErr"}
}

func (self *binder) Stat(name string) float64 {
	switch name {
	case "Send":
		return float64(atomic.LoadUint32(&self.statSend))
	case "SendErr":
		return float64(atomic.LoadUint32(&self.statSendErr))
	default:
		return 0
	}
}

// DefaultBinder is the default binder of the running process.
// Instead of DefaultBinder you can use the Bind function.
var DefaultBinder Binder = NewBinder(DefaultTransport)

// Bind binds a local channel to a URI that is addressing a remote
// channel. After the binding is established, the bound channel may
// be used to send messages to the remote channel.
//
// Remotely published channels are addressed by URI's. The URI
// syntax depends on the underlying transport. For the default TCP
// transport an address has the syntax: tcp://HOST[:PORT]/ID
//
// The uri parameter contains the URI and can be of type string or
// *url.URL. An error channel (of type chan error) may also be
// specified. This error channel will receive transport errors, etc.
// related to the bound channel.
//
// It is also possible to associate a new error channel with an
// already bound channel. For this purpose use a nil uri and
// the new error channel to associate with the bound channel.
//
// To unbind a bound channel simply close it.
//
// Bind binds a channel using the DefaultBinder.
func Bind(uri interface{}, ichan interface{}, echan chan error) error {
	return DefaultBinder.Bind(uri, ichan, echan)
}
