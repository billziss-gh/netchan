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
	"fmt"
	"net/http"
	"net/url"
	"reflect"
	//"github.com/gorilla/websocket"
)

type wsLink struct {
}

func (self *wsLink) Sigchan() chan struct{} {
	return nil
}

func (self *wsLink) Reference() {
}

func (self *wsLink) Dereference() {
}

func (self *wsLink) Activate() {
}

func (self *wsLink) Recv() (id string, vmsg reflect.Value, err error) {
	return
}

func (self *wsLink) Send(id string, vmsg reflect.Value) (err error) {
	return
}

func (self *wsLink) String() string {
	return ""
}

var _ Link = (*wsLink)(nil)
var _ fmt.Stringer = (*wsLink)(nil)

type wsTransport struct {
}

func NewWsTransport(marshaler Marshaler, uri *url.URL, serveMux *http.ServeMux,
	cfg *Config) Transport {
	return NewWsTransportTLS(marshaler, uri, serveMux, cfg, nil)
}

func NewWsTransportTLS(marshaler Marshaler, uri *url.URL, serveMux *http.ServeMux,
	cfg *Config, tlscfg *tls.Config) Transport {
	return nil
}

func (self *wsTransport) SetChanEncoder(chanEnc ChanEncoder) {
}

func (self *wsTransport) SetChanDecoder(chanDec ChanDecoder) {
}

func (self *wsTransport) SetRecver(recver func(link Link) error) {
}

func (self *wsTransport) SetSender(sender func(link Link) error) {
}

func (self *wsTransport) Listen() error {
	return nil
}

func (self *wsTransport) Connect(uri *url.URL) (string, Link, error) {
	return "", nil, nil
}

func (self *wsTransport) Close() {
}

var _ Transport = (*wsTransport)(nil)
