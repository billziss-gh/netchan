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

type loginMsg struct {
	Name, Pass string
	User       chan chatMsg
	Resp       chan chan chatMsg
}

type chatMsg struct {
	From, To, Text string
}

type session struct {
	name       string
	user, serv chan chatMsg
}

var (
	sessionMux sync.RWMutex
	sessionMap = make(map[string]*session)
)

func chat(src *session) {
	errchan := make(chan error, 1)
	netchan.Connect(nil, src.user, errchan)

loop:
	for {
		var msg chatMsg
		select {
		case msg = <-src.serv:
			if "" == msg.To {
				break loop
			}
		case err := <-errchan:
			log.Printf("error: %s\n", err)
			break
		}

		sessionMux.RLock()
		dst, ok := sessionMap[msg.To]
		sessionMux.RUnlock()
		if ok {
			msg.From = src.name
			dst.user <- msg
		}
	}

	sessionMux.Lock()
	delete(sessionMap, src.name)
	close(src.user)
	sessionMux.Unlock()
}

func run(login chan loginMsg) {
	for {
		msg := <-login
		if "" == msg.Name || nil == msg.User || nil == msg.Resp {
			if nil != msg.User {
				close(msg.User)
			}
			if nil != msg.Resp {
				close(msg.Resp)
			}
			continue
		}
		if msg.Name != msg.Pass { // "security" check
			msg.Resp <- nil
			close(msg.User)
			close(msg.Resp)
			continue
		}

		sessionMux.Lock()
		src, ok := sessionMap[msg.Name]
		if ok {
			close(src.user)
			src.user = msg.User
		} else {
			src = &session{msg.Name, msg.User, make(chan chatMsg, 1)}
			sessionMap[msg.Name] = src
		}
		sessionMux.Unlock()
		go chat(src)
		msg.Resp <- src.serv
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
	marshaler.RegisterType(chatMsg{})
	netchan.RegisterTransport("ws",
		netchan.NewWsTransport(marshaler, uri, http.DefaultServeMux, nil))

	login := make(chan loginMsg, 64)
	err := netchan.Publish("login", login)
	if nil == err {
		go run(login)
		err = http.ListenAndServe(":"+uri.Port(), nil)
	}
	if nil != err && http.ErrServerClosed != err {
		log.Fatalf("error: %v\n", err)
	}
}
