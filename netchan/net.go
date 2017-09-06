/*
 * net.go
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
	"crypto/tls"
	"net"
	"net/url"
	"reflect"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
)

type netLink struct {
	owner *netMultiLink // access to link uri and transport
	mux   sync.Mutex    // guards following fields
	cond  sync.Cond     // guards condition (self.done || nil != self.conn); uses mux
	conn  net.Conn      // network connection; nil when not connected
	init  bool          // true: recver/sender goroutines active
	done  bool          // true: link has been closed
}

func newNetLink(owner *netMultiLink) *netLink {
	self := &netLink{owner: owner}
	self.cond.L = &self.mux
	return self
}

func (self *netLink) Open() {
	self.mux.Lock()
	defer self.mux.Unlock()

	if !self.done && !self.init {
		self.init = true
		go self.recver()
		go self.sender()
	}
}

func (self *netLink) close() {
	self.reset(true)
}

func (self *netLink) reset(done bool) {
	self.mux.Lock()
	defer self.mux.Unlock()

	if done {
		self.done = true
	}

	if nil != self.conn {
		self.conn.Close()
		self.conn = nil
	}

	if done {
		self.cond.Signal()
	}
}

func (self *netLink) accept(conn net.Conn) *netLink {
	self.mux.Lock()
	defer self.mux.Unlock()

	if self.done || nil != self.conn || nil == conn {
		return nil
	}

	self.conn = conn
	self.cond.Signal()

	return self
}

func (self *netLink) connect() (net.Conn, error) {
	self.mux.Lock()
	defer self.mux.Unlock()

	if self.done {
		return nil, ErrTransportClosed
	}

	if nil == self.conn {
		var conn net.Conn
		var err error
		if nil == self.owner.transport.tlscfg {
			conn, err = net.Dial("tcp", self.owner.uri.Host)
		} else {
			conn, err = tls.Dial("tcp", self.owner.uri.Host, self.owner.transport.tlscfg)
		}
		if nil != err {
			return nil, newErrTransport(err)
		}

		self.conn = conn
		self.cond.Signal()
	}

	return self.conn, nil
}

func (self *netLink) waitconn() (net.Conn, error) {
	self.mux.Lock()
	defer self.mux.Unlock()

	for {
		if self.done {
			return nil, ErrTransportClosed
		}

		if nil != self.conn {
			return self.conn, nil
		}

		self.cond.Wait()
	}
}

func (self *netLink) Recv() (id string, vmsg reflect.Value, err error) {
	conn, err := self.waitconn()
	if nil != err {
		return
	}

	buf, err := readMsg(conn)
	if nil != err {
		self.reset(false)
		return
	}

	id, vmsg, err = self.owner.transport.marshaler.Unmarshal(self, buf)
	if nil != err {
		self.reset(false)
		return
	}

	return
}

func (self *netLink) Send(id string, vmsg reflect.Value) (err error) {
	conn, err := self.connect()
	if nil != err {
		return
	}

	buf, err := self.owner.transport.marshaler.Marshal(self, id, vmsg)
	if nil != err {
		// do not reset the link
		return
	}

	err = writeMsg(conn, buf)
	if nil != err {
		self.reset(false)
		return
	}

	return
}

func (self *netLink) recver() {
	for {
		err := self.owner.transport.recver(self)
		if ErrTransportClosed == err || self.done {
			break
		}
	}
	self.reset(true)
}

func (self *netLink) sender() {
	for {
		err := self.owner.transport.sender(self)
		if ErrTransportClosed == err || self.done {
			break
		}
	}
	self.reset(true)
}

var _ Link = (*netLink)(nil)

type netMultiLink struct {
	transport *netTransport
	uri       *url.URL
	index     uint32
	link      []*netLink
}

func newNetMultiLink(transport *netTransport, uri *url.URL) *netMultiLink {
	self := &netMultiLink{
		transport: transport,
		uri:       uri,
		index:     ^uint32(0), // start at -1
		link:      make([]*netLink, configMaxConn),
	}
	for i := range self.link {
		self.link[i] = newNetLink(self)
	}
	return self
}

func (self *netMultiLink) close() {
	for _, link := range self.link {
		link.close()
	}
}

func (self *netMultiLink) accept(conn net.Conn) *netLink {
	index := int(atomic.AddUint32(&self.index, +1))
	for i := range self.link {
		link := self.link[(i+index)%len(self.link)].accept(conn)
		if nil != link {
			return link
		}
	}
	return nil
}

func (self *netMultiLink) choose() *netLink {
	index := int(atomic.AddUint32(&self.index, +1))
	return self.link[index%len(self.link)]
}

type netTransport struct {
	marshaler Marshaler
	uri       *url.URL
	tlscfg    *tls.Config
	recver    func(link Link) error
	sender    func(link Link) error
	mux       sync.Mutex
	done      bool
	listen    net.Listener
	mlink     map[string]*netMultiLink
}

func newNetTransport(marshaler Marshaler, uri *url.URL) *netTransport {
	return newNetTransportTLS(marshaler, uri, nil)
}

func newNetTransportTLS(marshaler Marshaler, uri *url.URL, tlscfg *tls.Config) *netTransport {
	if nil != tlscfg {
		tlscfg = tlscfg.Clone()
	}

	return &netTransport{
		marshaler: marshaler,
		uri:       uri,
		tlscfg:    tlscfg,
		mlink:     make(map[string]*netMultiLink),
	}
}

func (self *netTransport) SetChanEncoder(chanEnc ChanEncoder) {
	self.marshaler.SetChanEncoder(chanEnc)
}

func (self *netTransport) SetChanDecoder(chanDec ChanDecoder) {
	self.marshaler.SetChanDecoder(chanDec)
}

func (self *netTransport) SetRecver(recver func(link Link) error) {
	self.recver = recver
}

func (self *netTransport) SetSender(sender func(link Link) error) {
	self.sender = sender
}

func (self *netTransport) Listen() error {
	if (nil == self.tlscfg && "tcp" != self.uri.Scheme) ||
		(nil != self.tlscfg && "tls" != self.uri.Scheme) ||
		"" == self.uri.Port() {
		return ErrTransportInvalid
	}

	self.mux.Lock()
	defer self.mux.Unlock()
	if self.done {
		return ErrTransportClosed
	}

	if nil == self.listen {
		var listen net.Listener
		var err error
		if nil == self.tlscfg {
			listen, err = net.Listen("tcp", self.uri.Host)
		} else {
			listen, err = tls.Listen("tcp", self.uri.Host, self.tlscfg)
		}
		if nil != err {
			return newErrTransport(err)
		}

		self.listen = listen
		go self.accepter()
	}

	return nil
}

func (self *netTransport) Connect(uri *url.URL) (string, Link, error) {
	if (nil == self.tlscfg && "tcp" != self.uri.Scheme) ||
		(nil != self.tlscfg && "tls" != self.uri.Scheme) {
		return "", nil, ErrTransportInvalid
	}

	id := strings.TrimPrefix(uri.Path, "/")
	if "" == id || strings.ContainsAny(id, "/") {
		return "", nil, ErrArgumentInvalid
	}

	mlink, err := self.connect(uri)
	if nil != err {
		return "", nil, err
	}

	return id, mlink.choose(), err
}

func (self *netTransport) Close() {
	self.mux.Lock()
	defer self.mux.Unlock()
	self.done = true
	if nil != self.listen {
		self.listen.Close()
	}
	for _, mlink := range self.mlink {
		mlink.close()
	}
}

func (self *netTransport) accepter() {
	for {
		conn, err := self.listen.Accept()
		if nil != err {
			if self.done {
				break
			}
			continue
		}

		host, _, err := net.SplitHostPort(conn.RemoteAddr().String())
		if nil != err {
			conn.Close()
			continue
		}

		/* Localhost loop fix
		 *
		 * The localhost loop happens when the channel receiver and sender
		 * are both on the localhost. It is possible for the transport to
		 * pick the same link (and the same socket/net.Conn) for both the
		 * receiver and the sender side. This of course does not work as
		 * one cannot use the same socket to communicate with itself.
		 *
		 * Here is the hackfix: upon seeing that the remote is 127.0.0.1
		 * (localhost), we force the address to 127.0.0.127 (another address
		 * for localhost). Since links are keyed by uri/address we ensure
		 * that a different link (and hence different socket) will be picked
		 * up, thus using a pair of sockets for communication.
		 */
		if "127.0.0.1" == host {
			host = "127.0.0.127"
		}

		mlink, err := self.connect(&url.URL{
			Scheme: conn.RemoteAddr().Network(),
			Host:   host,
		})
		if nil != err {
			conn.Close()
			continue
		}

		link := mlink.accept(conn)
		if nil == link {
			conn.Close()
			continue
		}

		link.Open()
	}
}

func (self *netTransport) connect(uri *url.URL) (*netMultiLink, error) {
	if self.done {
		return nil, ErrTransportClosed
	}

	port := uri.Port()
	if "" != port {
		portnum, err := net.LookupPort("tcp", uri.Port())
		if nil != err {
			return nil, newErrTransport(err)
		}
		port = strconv.Itoa(portnum)
	} else {
		port = self.uri.Port()
		if "" == port {
			return nil, ErrTransportInvalid
		}
	}

	hosts, err := net.LookupHost(uri.Hostname())
	if nil != err {
		return nil, newErrTransport(err)
	}

	uri = &url.URL{
		Scheme: uri.Scheme,
		Host:   net.JoinHostPort(hosts[0], port),
	}
	uristr := uri.String()

	self.mux.Lock()
	defer self.mux.Unlock()
	if self.done {
		return nil, ErrTransportClosed
	}

	mlink, ok := self.mlink[uristr]
	if !ok {
		mlink = newNetMultiLink(self, uri)
		self.mlink[uristr] = mlink
	}

	return mlink, nil
}

var _ Transport = RegisterTransport("tcp", newNetTransport(
	DefaultMarshaler,
	&url.URL{
		Scheme: "tcp",
		Host:   ":25454",
	}))
