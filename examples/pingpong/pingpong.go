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

	err := netchan.Connect(uri, chch, nil)
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

	err := netchan.Publish(id, chch)
	if nil != err {
		panic(err)
	}

	for i := 0; count > i; i++ {
		ch := <-chch
		fmt.Println("pong")
		ch <- struct{}{}
		close(ch)
	}

	netchan.Unpublish(id, chch)

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
		netchan.NewNetTransport(netchan.DefaultMarshaler, lnu))

	if "ping" == cmd {
		mainWG.Add(1)
		go ping(uri, cnt)
	} else if "pong" == cmd {
		mainWG.Add(1)
		go pong(uri, cnt)
	}

	mainWG.Wait()
}
