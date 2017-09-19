// -build websocket

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

func NewWsTransport(marshaler Marshaler, uri *url.URL, serveMux *http.ServeMux,
	cfg *Config) Transport {
	return NewWsTransportTLS(marshaler, uri, serveMux, cfg, nil)
}

func NewWsTransportTLS(marshaler Marshaler, uri *url.URL, serveMux *http.ServeMux,
	cfg *Config, tlscfg *tls.Config) Transport {
	return nil
}

func (self *wsTransport) Listen() error {
	return nil
}

func (self *wsTransport) Connect(uri *url.URL) (string, Link, error) {
	return "", nil, nil
}

var _ Transport = (*wsTransport)(nil)
