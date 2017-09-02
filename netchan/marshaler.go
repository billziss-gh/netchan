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
	"crypto/aes"
	"crypto/cipher"
	crand "crypto/rand"
	"unsafe"
)

const sizeofUintptr = unsafe.Sizeof(uintptr(0))

var refCipher = newRefCipher()

func newRefCipher() cipher.Block {
	var key [16]byte

	_, err := crand.Read(key[:])
	if nil != err {
		panic(err)
	}

	c, err := aes.NewCipher(key[:])
	if nil != err {
		panic(err)
	}

	return c
}

func RefMarshal(val interface{}) ([]byte, error) {
	ref := weakref(val)
	if 0 == ref {
		return nil, ErrMarshalerRef
	}

	buf := make([]byte, 16)
	copy(buf, (*(*[sizeofUintptr]byte)(unsafe.Pointer(&ref)))[:])
	refCipher.Encrypt(buf, buf)

	return buf, nil
}

func RefUnmarshal(buf []byte) (interface{}, error) {
	var refbuf [16]byte
	refCipher.Decrypt(refbuf[:], buf)

	ref := *(*uintptr)(unsafe.Pointer(&refbuf))

	val := strongref(ref)
	if nil == val {
		return nil, ErrMarshalerRef
	}

	return val, nil
}

var DefaultMarshaler Marshaler = newGobMarshaler()
