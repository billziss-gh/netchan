// +build websocket

/*
 * ws.go
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
	"net/http"
	"net/url"
	"time"

	"github.com/gorilla/websocket"
)

var wsTransportOptab = netOptab{
	wsDial,
	wsRemoteAddr,
	wsReadMsg,
	wsWriteMsg,
	wsClose,
}

func wsDial(transport *netTransport, uri *url.URL) (conn interface{}, err error) {
	dialer := websocket.Dialer{
		NetDial: func(network, addr string) (net.Conn, error) {
			conn, err := netDial(transport, &url.URL{
				Scheme: network,
				Host:   addr,
			})
			if nil != err {
				return nil, err
			}
			return conn.(net.Conn), nil
		},
	}

	conn, _, err = dialer.Dial(uri.String(), nil)
	return
}

func wsRemoteAddr(conn0 interface{}) string {
	conn := conn0.(*websocket.Conn)

	return conn.RemoteAddr().String()
}

func wsReadMsg(conn0 interface{}, idleTimeout time.Duration) ([]byte, error) {
	conn := conn0.(*websocket.Conn)

	if 0 != idleTimeout {
		// net.Conn does not have "idle" deadline, so emulate with read deadline on message length
		conn.SetReadDeadline(time.Now().Add(idleTimeout))
	}

	_, msg, err := conn.ReadMessage()
	if nil != err {
		return nil, MakeErrTransport(err)
	}

	return msg, nil
}

func wsWriteMsg(conn0 interface{}, idleTimeout time.Duration, msg []byte) error {
	conn := conn0.(*websocket.Conn)

	n := len(msg)
	if configMaxMsgSize < n {
		return ErrTransportMessageCorrupt
	}

	if 0 != idleTimeout {
		// extend read deadline to allow enough time for message to be written
		conn.SetReadDeadline(time.Now().Add(idleTimeout))
	}

	err := conn.WriteMessage(websocket.BinaryMessage, msg)
	if nil != err {
		return MakeErrTransport(err)
	}

	if 0 != idleTimeout {
		// extend the "idle" deadline as we just got a message
		conn.SetReadDeadline(time.Now().Add(idleTimeout))
	}

	return nil
}

func wsClose(conn0 interface{}) error {
	conn := conn0.(*websocket.Conn)

	return conn.Close()
}

type wsTransport struct {
	httpTransport
}

// NewWsTransport creates a new websocket Transport. The URI to listen to
// should have the syntax ws://[HOST]:PORT/PATH. If a ServeMux is
// provided, it will be used instead of creating a new HTTP server.
func NewWsTransport(marshaler Marshaler, uri *url.URL, serveMux *http.ServeMux,
	cfg *Config) Transport {

	self := &wsTransport{}
	self.init(marshaler, uri, serveMux, cfg, nil)
	return self
}

// NewWsTransportTLS creates a new secure websocket Transport. The URI
// to listen to should have the syntax wss://[HOST]:PORT/PATH. If a
// ServeMux is provided, it will be used instead of creating a new HTTPS
// server.
func NewWsTransportTLS(marshaler Marshaler, uri *url.URL, serveMux *http.ServeMux,
	cfg *Config, tlscfg *tls.Config) Transport {

	self := &wsTransport{}
	self.init(marshaler, uri, serveMux, cfg, tlscfg)
	return self
}

func (self *wsTransport) init(marshaler Marshaler, uri *url.URL, serveMux *http.ServeMux,
	cfg *Config, tlscfg *tls.Config) {

	(&self.httpTransport).init(marshaler, uri, serveMux, cfg, tlscfg)

	if nil != uri {
		port := uri.Port()
		if "" == port {
			if "ws" == uri.Scheme {
				port = "80"
			} else if "wss" == uri.Scheme {
				port = "443"
			}
		}

		uri = &url.URL{
			Scheme: uri.Scheme,
			Host:   net.JoinHostPort(uri.Hostname(), port),
			Path:   uri.Path,
		}
	}

	self.handler = self.wsHandler
	self.uri = uri
}

func (self *wsTransport) wsHandler(w http.ResponseWriter, r *http.Request) {
	upgrader := websocket.Upgrader{}
	conn, err := upgrader.Upgrade(w, r, nil)
	if nil != err {
		return
	}

	err = self.accept(conn)
	if nil != err {
		conn.Close()
	}
}

var _ Transport = (*wsTransport)(nil)
