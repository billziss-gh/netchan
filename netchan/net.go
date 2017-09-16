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
	"fmt"
	"io"
	"math/rand"
	"net"
	"net/url"
	"reflect"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

const (
	netRedialDelayMin = 3 * time.Second
	netRedialDelayMax = 60 * time.Second
)

func netDial(uri *url.URL, redialTimeout time.Duration,
	tlscfg *tls.Config) (conn net.Conn, err error) {
	var deadline time.Time
	var rnd *rand.Rand

	if 0 != redialTimeout {
		deadline = time.Now().Add(redialTimeout)
	}

	for delay := netRedialDelayMin; ; delay *= 2 {
		if nil == tlscfg {
			conn, err = net.Dial("tcp", uri.Host)
		} else {
			conn, err = tls.Dial("tcp", uri.Host, tlscfg)
		}

		if nil == err || 0 == redialTimeout {
			break
		}

		now := time.Now()
		remain := deadline.Sub(now)
		if 0 >= remain {
			break
		}

		if nil == rnd {
			rnd = rand.New(rand.NewSource(now.UnixNano()))
		}
		if netRedialDelayMax < delay {
			delay = netRedialDelayMax
		}

		delay += time.Duration(rnd.Int63n(int64(delay)))

		if remain < delay {
			time.Sleep(remain)
		} else {
			time.Sleep(delay)
		}

	}

	return
}

func netListen(address string, tlscfg *tls.Config) (net.Listener, error) {
	if nil == tlscfg {
		return net.Listen("tcp", address)
	} else {
		return tls.Listen("tcp", address, tlscfg)
	}
}

func netReadMsg(conn net.Conn, idleTimeout time.Duration) ([]byte, error) {
	if 0 != idleTimeout {
		// net.Conn does not have "idle" deadline, so emulate with read deadline on message length
		conn.SetReadDeadline(time.Now().Add(idleTimeout))
	}

	buf := [4]byte{}
	_, err := io.ReadFull(conn, buf[:])
	if nil != err {
		return nil, MakeErrTransport(err)
	}

	n := int(buf[0]) | (int(buf[1]) << 8) | (int(buf[2]) << 16) | (int(buf[3]) << 24)
	if 4 > n || configMaxMsgSize < n {
		return nil, ErrTransportMessageCorrupt
	}

	if 0 != idleTimeout {
		// extend read deadline to allow enough time for message to be read
		conn.SetReadDeadline(time.Now().Add(idleTimeout))
	}

	msg := make([]byte, n)
	_, err = io.ReadFull(conn, msg[4:])
	if nil != err {
		return nil, MakeErrTransport(err)
	}

	msg[0] = buf[0]
	msg[1] = buf[1]
	msg[2] = buf[2]
	msg[3] = buf[3]

	return msg, nil
}

func netWriteMsg(conn net.Conn, idleTimeout time.Duration, msg []byte) error {
	n := len(msg)
	if 4 > n || configMaxMsgSize < n {
		return ErrTransportMessageCorrupt
	}

	msg[0] = byte(n & 0xff)
	msg[1] = byte((n >> 8) & 0xff)
	msg[2] = byte((n >> 16) & 0xff)
	msg[3] = byte((n >> 24) & 0xff)

	if 0 != idleTimeout {
		// extend read deadline to allow enough time for message to be written
		conn.SetReadDeadline(time.Now().Add(idleTimeout))
	}

	_, err := conn.Write(msg)
	if nil != err {
		return MakeErrTransport(err)
	}

	if 0 != idleTimeout {
		// extend the "idle" deadline as we just got a message
		conn.SetReadDeadline(time.Now().Add(idleTimeout))
	}

	return nil
}

func sigchanclose(sigchan chan struct{}) (ok bool) {
	defer func() {
		recover()
	}()
	close(sigchan)
	ok = true
	return
}

type netLink struct {
	owner       *netMultiLink // access to link uri and transport
	sigchan     chan struct{} // signal channel; closed when link is closed
	mux         sync.Mutex    // guards following fields
	cond        sync.Cond     // guards condition (self.done || nil != self.conn); uses mux
	conn        net.Conn      // network connection; nil when not connected
	idleTimeout time.Duration // idle timeout for dialed (not accepted) connections
	init        bool          // true: recver/sender goroutines active
	done        bool          // true: link has been closed
}

func newNetLink(owner *netMultiLink) *netLink {
	self := &netLink{owner: owner, sigchan: make(chan struct{}, 0x7fffffff)}
	self.cond.L = &self.mux
	return self
}

func (self *netLink) Sigchan() chan struct{} {
	return self.sigchan
}

func (self *netLink) Reference() {
	self.owner.gcref()
}

func (self *netLink) Dereference() {
	self.owner.gcderef()
}

func (self *netLink) Activate() {
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
		self.idleTimeout = 0
		self.owner.gcderef()
	}

	if done {
		self.cond.Signal()
		sigchanclose(self.sigchan)
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
	self.owner.gcref()

	return self
}

func (self *netLink) connect() (net.Conn, time.Duration, error) {
	self.mux.Lock()
	defer self.mux.Unlock()

	if self.done {
		return nil, 0, ErrTransportClosed
	}

	if nil == self.conn {
		self.mux.Unlock()
		conn, err := self.owner.transport.dial(
			self.owner.uri,
			self.owner.transport.cfg.RedialTimeout,
			self.owner.transport.tlscfg)
		self.mux.Lock()
		if nil != err {
			return nil, 0, MakeErrTransport(err)
		}

		if nil == self.conn {
			self.conn = conn
			self.idleTimeout = self.owner.transport.cfg.IdleTimeout
			self.cond.Signal()
			self.owner.gcref()
		} else {
			conn.Close()
		}
	}

	return self.conn, self.idleTimeout, nil
}

func (self *netLink) waitconn() (net.Conn, time.Duration, error) {
	self.mux.Lock()
	defer self.mux.Unlock()

	for {
		if self.done {
			return nil, 0, ErrTransportClosed
		}

		if nil != self.conn {
			return self.conn, self.idleTimeout, nil
		}

		self.cond.Wait()
	}
}

func (self *netLink) Recv() (id string, vmsg reflect.Value, err error) {
	conn, idleTimeout, err := self.waitconn()
	if nil != err {
		return
	}

	buf, err := netReadMsg(conn, idleTimeout)
	if nil != err {
		self.reset(false)
		return
	}

	id, vmsg, err = self.owner.transport.marshaler.Unmarshal(self, buf)
	if nil != err {
		// do not reset the link
		return
	}

	return
}

func (self *netLink) Send(id string, vmsg reflect.Value) (err error) {
	conn, idleTimeout, err := self.connect()
	if nil != err {
		return
	}

	buf, err := self.owner.transport.marshaler.Marshal(self, id, vmsg)
	if nil != err {
		// do not reset the link
		return
	}

	err = netWriteMsg(conn, idleTimeout, buf)
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

func (self *netLink) String() string {
	return self.owner.linkString(self)
}

var _ Link = (*netLink)(nil)
var _ fmt.Stringer = (*netLink)(nil)

type netMultiLink struct {
	transport *netTransport
	uri       *url.URL
	index     uint32
	link      []*netLink
	refcnt    int32
}

func newNetMultiLink(transport *netTransport, uri *url.URL) *netMultiLink {
	self := &netMultiLink{
		transport: transport,
		uri:       uri,
		index:     ^uint32(0), // start at -1
		link:      make([]*netLink, transport.cfg.MaxLinks),
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

func (self *netMultiLink) gcref() {
	atomic.AddInt32(&self.refcnt, +1)
}

func (self *netMultiLink) gcderef() {
	refcnt := atomic.AddInt32(&self.refcnt, -1)
	if 0 == refcnt {
		go self.transport.gc(self)
	}
}

func (self *netMultiLink) gcisref() bool {
	return 0 != atomic.LoadInt32(&self.refcnt)
}

func (self *netMultiLink) linkString(link *netLink) string {
	for i, l := range self.link {
		if l == link {
			return fmt.Sprintf("%v[%v]", self.uri, i)
		}
	}

	return fmt.Sprintf("%v[?]", self.uri)
}

type netTransport struct {
	marshaler Marshaler
	uri       *url.URL
	cfg       *Config
	tlscfg    *tls.Config
	recver    func(link Link) error
	sender    func(link Link) error
	dial      func(uri *url.URL, redialTimeout time.Duration, tlscfg *tls.Config) (net.Conn, error)
	mux       sync.Mutex
	done      bool
	listen    net.Listener
	mlink     map[string]*netMultiLink
}

// NewNetTransport creates a new TCP Transport. The URI to listen to
// should have the syntax tcp://[HOST]:PORT.
func NewNetTransport(marshaler Marshaler, uri *url.URL, cfg *Config) Transport {
	return NewNetTransportTLS(marshaler, uri, cfg, nil)
}

// NewNetTransportTLS creates a new TLS Transport. The URI to listen to
// should have the syntax tls://[HOST]:PORT.
func NewNetTransportTLS(marshaler Marshaler, uri *url.URL, cfg *Config,
	tlscfg *tls.Config) Transport {
	if nil != cfg {
		cfg = cfg.Clone()
	} else {
		cfg = &Config{}
	}
	if 0 == cfg.MaxLinks {
		cfg.MaxLinks = configMaxLinks
	}

	if nil != tlscfg {
		tlscfg = tlscfg.Clone()
	}

	uri = &url.URL{
		Scheme: uri.Scheme,
		Host:   uri.Host,
	}

	return &netTransport{
		marshaler: marshaler,
		uri:       uri,
		cfg:       cfg,
		tlscfg:    tlscfg,
		dial:      netDial,
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
		listen, err := netListen(self.uri.Host, self.tlscfg)
		if nil != err {
			return MakeErrTransport(err)
		}

		self.listen = listen
		go self.accepter()
	}

	return nil
}

func (self *netTransport) Connect(uri *url.URL) (string, Link, error) {
	if (nil == self.tlscfg && "tcp" != uri.Scheme) ||
		(nil != self.tlscfg && "tls" != uri.Scheme) {
		return "", nil, ErrTransportInvalid
	}

	id := strings.TrimPrefix(uri.Path, "/")
	if "" == id || strings.ContainsAny(id, "/") {
		return "", nil, ErrArgumentInvalid
	}

	mlink, err := self.connect(&url.URL{
		Scheme: uri.Scheme,
		Host:   uri.Host,
	})
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

		err = self.accept(conn)
		if nil != err {
			conn.Close()
		}
	}
}

func (self *netTransport) accept(conn net.Conn) error {
	host, _, err := net.SplitHostPort(conn.RemoteAddr().String())
	if nil != err {
		return err
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
		Scheme: self.uri.Scheme,
		Host:   host,
		Path:   self.uri.Path,
	})
	if nil != err {
		return err
	}

	link := mlink.accept(conn)
	mlink.gcderef()
	if nil == link {
		return err
	}

	link.Activate()

	return nil
}

func (self *netTransport) connect(uri *url.URL) (*netMultiLink, error) {
	if self.done {
		return nil, ErrTransportClosed
	}

	port := uri.Port()
	if "" != port {
		portnum, err := net.LookupPort("tcp", uri.Port())
		if nil != err {
			return nil, MakeErrTransport(err)
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
		return nil, MakeErrTransport(err)
	}

	uri = &url.URL{
		Scheme: uri.Scheme,
		Host:   net.JoinHostPort(hosts[0], port),
		Path:   uri.Path,
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

	mlink.gcref()

	return mlink, nil
}

func (self *netTransport) gc(mlink *netMultiLink) {
	self.mux.Lock()
	defer self.mux.Unlock()

	/*
	 * We take care to gcref() this link in connect() while holding the mux.
	 * This ensures that gcisref() will check for a reference in a safe manner.
	 */

	if !mlink.gcisref() {
		delete(self.mlink, mlink.uri.String())
		mlink.close()
		if nil != gcDebugLog {
			gcDebugLog("GC: %v", mlink.uri)
		}
	}
}

var _ Transport = RegisterTransport("tcp", NewNetTransport(
	DefaultMarshaler,
	&url.URL{
		Scheme: "tcp",
		Host:   ":25454",
	},
	nil))
