/*
 * marshaler.go
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
	"encoding/base64"
)

var DefaultMarshaler Marshaler = NewGobMarshaler()

func RefEncode(w weakref) string {
	return "(" + base64.RawURLEncoding.EncodeToString(w[:]) + ")"
}

func RefDecode(s string) (weakref, bool) {
	if 2 < len(s) && '(' == s[0] && ')' == s[len(s)-1] {
		var w weakref
		_, err := base64.RawURLEncoding.Decode(w[:], []byte(s[1:len(s)-1]))
		if nil == err {
			return w, true
		}
	}
	return weakref{}, false
}
