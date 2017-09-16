/*
 * debug.go
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
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
)

func debugLogFunc(format string, v ...interface{}) {
	log.Output(2, fmt.Sprintf(format, v...))
}

func initDebugFlags() uint {
	godebug := os.Getenv("GODEBUG")

	for _, pair := range strings.Split(godebug, ",") {
		kv := strings.SplitN(pair, "=", 2)
		if 2 <= len(kv) && "netchan" == kv[0] {
			u64, _ := strconv.ParseUint(kv[1], 0, 32)
			return uint(u64)
		}
	}

	return 0
}

var debugFlags = initDebugFlags()
var msgDebugLog = []func(string, ...interface{}){nil, debugLogFunc}[debugFlags&1]
var gcDebugLog = []func(string, ...interface{}){nil, debugLogFunc}[(debugFlags&2)>>1]
