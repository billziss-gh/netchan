/*
 * http.go
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
	"bytes"
	"crypto/tls"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"
)

func httpDial(uri *url.URL, redialTimeout time.Duration, tlscfg *tls.Config) (net.Conn, error) {
	conn, err := netDial(uri, redialTimeout, tlscfg)
	if nil != err {
		return nil, err
	}

	_, err = conn.Write([]byte("CONNECT " + uri.Path + " HTTP/1.0\r\n\r\n"))
	if nil != err {
		conn.Close()
		return nil, err
	}

	buf := make([]byte, len(httpStatus200))
	_, err = io.ReadFull(conn, buf)
	if nil != err {
		conn.Close()
		return nil, err
	}

	if !bytes.Equal(buf, httpStatus200) {
		conn.Close()
		return nil, ErrTransportUnexpectedResponse
	}

	return conn, nil
}

type httpTransport struct {
	netTransport
	serveMux *http.ServeMux
	server   *http.Server
	listen   bool
}

func NewHttpTransport(marshaler Marshaler, uri *url.URL, serveMux *http.ServeMux,
	cfg *Config) Transport {
	return NewHttpTransportTLS(marshaler, uri, nil, cfg, nil)
}

func NewHttpTransportTLS(marshaler Marshaler, uri *url.URL, serveMux *http.ServeMux,
	cfg *Config, tlscfg *tls.Config) Transport {
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

	port := uri.Port()
	if "" == port {
		if "http" == uri.Scheme {
			port = "80"
		} else {
			port = "443"
		}
	}

	uri = &url.URL{
		Scheme: uri.Scheme,
		Host:   net.JoinHostPort(uri.Hostname(), port),
		Path:   uri.Path,
	}

	return &httpTransport{
		netTransport: netTransport{
			marshaler: marshaler,
			uri:       uri,
			cfg:       cfg,
			tlscfg:    tlscfg,
			dial:      httpDial,
			mlink:     make(map[string]*netMultiLink),
		},
		serveMux: serveMux,
	}
}

func (self *httpTransport) Listen() error {
	if nil != self.uri {
		if (nil == self.tlscfg && "http" != self.uri.Scheme) ||
			(nil != self.tlscfg && "http" != self.uri.Scheme) {
			return ErrTransportInvalid
		}
	}

	self.mux.Lock()
	defer self.mux.Unlock()
	if self.done {
		return ErrTransportClosed
	}

	if nil == self.uri {
		return nil
	}

	if !self.listen {
		path := self.uri.Path
		if "" == path {
			path = "/"
		} else if !strings.HasSuffix(path, "/") {
			path += "/"
		}

		serveMux := self.serveMux
		if nil == serveMux {
			serveMux = http.NewServeMux()
			server := &http.Server{
				Addr:      self.uri.Host,
				TLSConfig: self.tlscfg,
				Handler:   serveMux,
			}

			var err error
			if "http" == self.uri.Scheme {
				err = server.ListenAndServe()
			} else {
				err = server.ListenAndServeTLS("", "")
			}
			if nil != err {
				return err
			}

			self.server = server
			self.serveMux = serveMux
		}

		serveMux.HandleFunc(path, self.serverRecv)

		self.listen = true
	}

	return nil
}

func (self *httpTransport) Connect(uri *url.URL) (string, Link, error) {
	if (nil == self.tlscfg && "http" != uri.Scheme) ||
		(nil != self.tlscfg && "https" != uri.Scheme) {
		return "", nil, ErrTransportInvalid
	}

	var path, id string
	index := strings.LastIndex(uri.Path, "/")
	if 0 < index {
		path = uri.Path[:index+1]
		id = uri.Path[index+1:]
	}
	if "" == id {
		return "", nil, ErrArgumentInvalid
	}

	mlink, err := self.connect(&url.URL{
		Scheme: uri.Scheme,
		Host:   uri.Host,
		Path:   path,
	})
	if nil != err {
		return "", nil, err
	}

	return id, mlink.choose(), err
}

func (self *httpTransport) Close() {
	self.mux.Lock()
	defer self.mux.Unlock()
	self.done = true
	if nil != self.server {
		self.server.Close()
		self.server = nil
		self.listen = false
	}
	for _, mlink := range self.mlink {
		mlink.close()
	}
}

func (self *httpTransport) serverRecv(w http.ResponseWriter, r *http.Request) {
	if "CONNECT" != r.Method {
		http.Error(w, "netchan: only CONNECT is allowed", http.StatusMethodNotAllowed)
		return
	}

	hj, ok := w.(http.Hijacker)
	if !ok {
		return
	}

	conn, _, err := hj.Hijack()
	if nil != err {
		return
	}

	_, err = conn.Write(httpStatus200)
	if nil != err {
		conn.Close()
		return
	}

	err = self.accept(conn)
	if nil != err {
		conn.Close()
	}
}

var httpStatus200 = []byte("HTTP/1.0 200 netchan: connected\r\n\r\n")

var _ Transport = (*httpTransport)(nil)
