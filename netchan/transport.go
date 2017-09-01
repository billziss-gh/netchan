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
	"io"
	"net/url"
	"sync"
)

func readMsg(r io.Reader) ([]byte, error) {
	buf := [4]byte{}
	_, err := io.ReadFull(r, buf[:])
	if nil != err {
		return nil, newErrTransport(err)
	}

	n := int(buf[0]) | (int(buf[1]) << 8) | (int(buf[2]) << 16) | (int(buf[3]) << 24)
	if 4 > n || configMaxMsgSize < n {
		return nil, ErrTransportMessageCorrupt
	}

	msg := make([]byte, n)
	_, err = io.ReadFull(r, msg[4:])
	if nil != err {
		return nil, newErrTransport(err)
	}

	msg[0] = buf[0]
	msg[1] = buf[1]
	msg[2] = buf[2]
	msg[3] = buf[3]

	return msg, nil
}

func writeMsg(w io.Writer, msg []byte) error {
	n := len(msg)
	if 4 > n || configMaxMsgSize < n {
		return ErrTransportMessageCorrupt
	}

	msg[0] = byte(n & 0xff)
	msg[1] = byte((n >> 8) & 0xff)
	msg[2] = byte((n >> 16) & 0xff)
	msg[3] = byte((n >> 24) & 0xff)

	_, err := w.Write(msg)
	if nil != err {
		return newErrTransport(err)
	}

	return nil
}

type defaultTransport struct {
	recver    func(link Link) error
	sender    func(link Link) error
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

func (self *defaultTransport) SetRecver(recver func(link Link) error) {
	self.recver = recver
}

func (self *defaultTransport) SetSender(sender func(link Link) error) {
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

		self.transportSetRecverSender()
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

	self.transportSetRecverSender()
	return transport.Connect(uri)
}

func (self *defaultTransport) Close() {
	/*
	 * Close is ignored on the defaultTransport.
	 */
}

func (self *defaultTransport) transportSetRecverSender() {
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

var DefaultTransport Transport = newDefaultTransport()

func RegisterTransport(scheme string, transport Transport) Transport {
	return DefaultTransport.(*defaultTransport).registerTransport(scheme, transport)
}

func UnregisterTransport(scheme string) Transport {
	return DefaultTransport.(*defaultTransport).unregisterTransport(scheme)
}
