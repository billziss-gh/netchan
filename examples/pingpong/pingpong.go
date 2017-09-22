/*
 * pingpong.go
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
	"fmt"
	"net/url"
	"os"
	"strconv"
	"sync"
	"time"

	"github.com/billziss-gh/netchan/netchan"
)

var mainWG = &sync.WaitGroup{}

func ping(uri string, count int) {
	chch := make(chan chan struct{})

	err := netchan.Bind(uri, chch, nil)
	if nil != err {
		panic(err)
	}

	for i := 0; count > i; i++ {
		ch := make(chan struct{})
		chch <- ch
		fmt.Println("ping")
		_ = <-ch
	}

	close(chch)

	mainWG.Done()
}

func pong(id string, count int) {
	chch := make(chan chan struct{})

	err := netchan.Expose(id, chch)
	if nil != err {
		panic(err)
	}

	for i := 0; count > i; i++ {
		ch := <-chch
		fmt.Println("pong")
		ch <- struct{}{}
		close(ch)
	}

	netchan.Unexpose(id, chch)

	time.Sleep(100 * time.Millisecond)

	mainWG.Done()
}

func iping(uri string, count int) {
	chch := make(chan interface{})

	err := netchan.Bind(uri, chch, nil)
	if nil != err {
		panic(err)
	}

	for i := 0; count > i; i++ {
		ch := make(chan interface{})
		chch <- ch
		fmt.Println("iping")
		close(chch)
		chch = (<-ch).(chan interface{})
	}

	close(chch)

	mainWG.Done()
}

func ipong(id string, count int) {
	chch := make(chan interface{})

	err := netchan.Expose(id, chch)
	if nil != err {
		panic(err)
	}

	for i := 0; count > i; i++ {
		ch := (<-chch).(chan interface{})
		fmt.Println("ipong")
		if 0 == i {
			netchan.Unexpose(id, chch)
		}
		chch = make(chan interface{})
		ch <- chch
		close(ch)
	}

	time.Sleep(100 * time.Millisecond)

	mainWG.Done()
}

func main() {
	args := os.Args

	cmd := args[1]
	lnu, _ := url.Parse(args[2])
	uri := args[3]
	cnt, _ := strconv.Atoi(args[4])

	netchan.RegisterTransport("tcp",
		netchan.NewNetTransport(netchan.DefaultMarshaler, lnu, nil))

	if "ping" == cmd {
		mainWG.Add(1)
		go ping(uri, cnt)
	} else if "pong" == cmd {
		mainWG.Add(1)
		go pong(uri, cnt)
	} else if "iping" == cmd {
		mainWG.Add(1)
		go iping(uri, cnt)
	} else if "ipong" == cmd {
		mainWG.Add(1)
		go ipong(uri, cnt)
	}

	mainWG.Wait()
}
