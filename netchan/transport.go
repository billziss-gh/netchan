/*
 * transport.go
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
	"sync"
)

type defaultTransport struct {
	recver    Recver
	sender    Sender
	lmux      sync.Mutex
	listening bool
	tonce     sync.Once
	tmux      sync.RWMutex
	transport map[string]Transport
}

func newDefaultTransport() *defaultTransport {
	return &defaultTransport{
		transport: make(map[string]Transport),
	}
}

func (self *defaultTransport) SetRecver(recver Recver) {
	self.recver = recver
}

func (self *defaultTransport) SetSender(sender Sender) {
	self.sender = sender
}

func (self *defaultTransport) Listen() (err error) {
	/*
	 * Listen succeeds if any underlying transport is able to listen.
	 * If all transports fail to listen, Listen returns the error
	 * from the "tcp" transport.
	 */

	self.lmux.Lock()
	defer self.lmux.Unlock()

	if !self.listening {
		self.tmux.RLock()
		defer self.tmux.RUnlock()

		self.initTransports()
		for scheme, transport := range self.transport {
			err0 := transport.Listen()
			if nil == err0 {
				self.listening = true
			} else {
				if "tcp" == scheme {
					err = err0
				}
			}
		}

		if self.listening {
			err = nil
		}
	}

	return
}

func (self *defaultTransport) Connect(uri *url.URL) (string, Link, error) {
	self.tmux.RLock()
	defer self.tmux.RUnlock()

	transport, ok := self.transport[uri.Scheme]
	if !ok {
		return "", nil, ErrTransportInvalid
	}

	self.initTransports()
	return transport.Connect(uri)
}

func (self *defaultTransport) Close() {
	/*
	 * Close is ignored on the defaultTransport.
	 */
}

func (self *defaultTransport) initTransports() {
	self.tonce.Do(func() {
		for _, transport := range self.transport {
			transport.SetRecver(self.recver)
			transport.SetSender(self.sender)
		}
	})
}

func (self *defaultTransport) registerTransport(scheme string, transport Transport) Transport {
	self.tmux.Lock()
	defer self.tmux.Unlock()

	self.transport[scheme] = transport

	return transport
}

func (self *defaultTransport) unregisterTransport(scheme string) Transport {
	self.tmux.Lock()
	defer self.tmux.Unlock()

	transport, ok := self.transport[scheme]
	if ok {
		delete(self.transport, scheme)
	}

	return transport
}

// DefaultTransport is the default Transport of the running process.
var DefaultTransport Transport = newDefaultTransport()

// RegisterTransport associates a URI scheme with a network transport.
func RegisterTransport(scheme string, transport Transport) Transport {
	return DefaultTransport.(*defaultTransport).registerTransport(scheme, transport)
}

// UnregisterTransport disassociates a URI scheme from a network transport.
func UnregisterTransport(scheme string) Transport {
	return DefaultTransport.(*defaultTransport).unregisterTransport(scheme)
}
