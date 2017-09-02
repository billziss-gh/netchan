/*
 * weakref.go
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
	"runtime"
	"sync"
	"unsafe"
)

type weakdat struct {
	ref uintptr
	ptr [2]uintptr
}

var (
	weakmux sync.Mutex
	weakmap = [2]map[uintptr]*weakdat{
		make(map[uintptr]*weakdat),
		make(map[uintptr]*weakdat),
	}
	weakcnt uintptr
)

func weakref(val interface{}) uintptr {
	if nil == val {
		return 0
	}

	ptr := *(*[2]uintptr)(unsafe.Pointer(&val))

	weakmux.Lock()
	defer weakmux.Unlock()

	dat := weakmap[1][ptr[1]]
	if nil == dat {
		for {
			weakcnt++
			if 0 != weakcnt {
				break
			}
		}
		dat = &weakdat{weakcnt, ptr}

		fin := func(interface{}) {
			weakmux.Lock()
			defer weakmux.Unlock()
			delete(weakmap[0], dat.ref)
			delete(weakmap[1], dat.ptr[1])
		}
		runtime.SetFinalizer((*struct{})(unsafe.Pointer(ptr[1])), fin)

		weakmap[0][dat.ref] = dat
		weakmap[1][dat.ptr[1]] = dat
	}

	return dat.ref
}

func strongref(ref uintptr) interface{} {
	weakmux.Lock()
	defer weakmux.Unlock()

	dat := weakmap[0][ref]
	if nil == dat {
		return nil
	}

	/*
	 * Converting a uintptr to an unsafe.Pointer is not guaranteed to work in general.
	 * However it does work in the current go runtimes. See https://goo.gl/zqfXU7
	 */

	return *(*interface{})(unsafe.Pointer(&dat.ptr))
}
