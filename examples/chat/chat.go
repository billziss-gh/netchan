// +build websocket

/*
 * chat.go
 *
 * Copyright 2017 Bill Zissimopoulos
 */
/*
 * This file is part of netchan.
 *
 * It is licensed under the MIT license. The full license text can be found
 * in the License.txt file at the root of this project.
 */

package main

import (
	"log"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"sync"

	"github.com/billziss-gh/netchan/netchan"
)

type session struct {
	Name       string
	User, Serv chan sessionMsg
}

type loginMsg struct {
	Name, Pass string
	User       chan sessionMsg
	Resp       chan chan sessionMsg
}

type sessionMsg struct {
	From, To, Text string
}

var (
	sessionMux sync.RWMutex
	sessionMap = make(map[string]*session)
	login      = make(chan loginMsg, 64)
)

func chat(src *session) {
	for {
		msg := <-src.Serv
		if "" == msg.To {
			sessionMux.Lock()
			delete(sessionMap, src.Name)
			sessionMux.Unlock()
			break
		}

		sessionMux.RLock()
		dst, ok := sessionMap[msg.To]
		sessionMux.RUnlock()
		if ok {
			msg.From = src.Name
			dst.User <- msg
		}
	}
}

func run() {
	for {
		msg := <-login
		if "" == msg.Name || nil == msg.User || nil == msg.Resp {
			continue
		}
		if msg.Name != msg.Pass { // "security" check
			msg.Resp <- nil
			close(msg.User)
			close(msg.Resp)
			continue
		}

		var src *session
		sessionMux.Lock()
		src, ok := sessionMap[msg.Name]
		if !ok {
			src = &session{
				msg.Name,
				msg.User,
				make(chan sessionMsg, 1),
			}
			sessionMap[msg.Name] = src
			go chat(src)
		}
		sessionMux.Unlock()
		msg.Resp <- src.Serv

		close(msg.Resp)
	}
}

func main() {
	var uri *url.URL
	if 2 == len(os.Args) {
		uri, _ = url.Parse(os.Args[1])
	}
	if nil == uri || "ws" != uri.Scheme || "" == uri.Port() {
		log.Fatalf("usage: %s ws://:PORT/PATH\n", filepath.Base(os.Args[0]))
	}

	marshaler := netchan.NewJsonMarshaler()
	marshaler.RegisterType(loginMsg{})
	netchan.RegisterTransport("ws",
		netchan.NewWsTransport(marshaler, uri, http.DefaultServeMux, nil))

	err := netchan.Publish("login", login)
	if nil == err {
		err = http.ListenAndServe(":"+uri.Port(), nil)
	}
	if nil != err && http.ErrServerClosed != err {
		log.Fatalf("error: %v\n", err)
	}
}
