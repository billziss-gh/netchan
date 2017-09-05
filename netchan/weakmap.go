/*
 * weakmap.go
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
	"runtime"
	"sync"
	"unsafe"
)

type weakref [16]byte

type weakmap struct {
	mux    sync.Mutex
	refmap map[weakref]*_weaktup
	ptrmap map[uintptr]*_weaktup
	cnt    weakref
}

type _weaktup struct {
	ref weakref
	ptr [2]uintptr
}

func newWeakmap() *weakmap {
	return &weakmap{
		refmap: make(map[weakref]*_weaktup),
		ptrmap: make(map[uintptr]*_weaktup),
	}
}

func (self *weakmap) weakref(val interface{}) weakref {
	if nil == val {
		return weakref{}
	}

	self.mux.Lock()
	defer self.mux.Unlock()

	ptr := *(*[2]uintptr)(unsafe.Pointer(&val))
	tup := self.ptrmap[ptr[1]]
	if nil == tup {
		tup = &_weaktup{ptr: ptr}
		_weakmapCipher.Encrypt(tup.ref[:], self.cnt[:])
		self._addtup(tup)

		*(*uint64)(unsafe.Pointer(&self.cnt)) = *(*uint64)(unsafe.Pointer(&self.cnt)) + 1
	}

	return tup.ref
}

func (self *weakmap) strongref(ref weakref, newfn func() interface{}) interface{} {
	if (weakref{}) == ref {
		return nil
	}

	self.mux.Lock()
	defer self.mux.Unlock()

	tup := self.refmap[ref]
	if nil == tup {
		if nil == newfn {
			return nil
		}

		val := newfn()
		ptr := *(*[2]uintptr)(unsafe.Pointer(&val))

		tup = &_weaktup{ref: ref, ptr: ptr}
		self._addtup(tup)
	}

	/*
	 * Converting a uintptr to an unsafe.Pointer is not guaranteed to work in general.
	 * However it does work in the current go runtimes. See https://goo.gl/zqfXU7
	 */

	return *(*interface{})(unsafe.Pointer(&tup.ptr))
}

func (self *weakmap) _addtup(tup *_weaktup) {
	fin := func(interface{}) {
		self.mux.Lock()
		defer self.mux.Unlock()
		delete(self.refmap, tup.ref)
		delete(self.ptrmap, tup.ptr[1])
	}
	runtime.SetFinalizer((*struct{})(unsafe.Pointer(tup.ptr[1])), fin)

	self.refmap[tup.ref] = tup
	self.ptrmap[tup.ptr[1]] = tup
}

var _weakmapCipher = _newWeakmapCipher()

func _newWeakmapCipher() cipher.Block {
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
