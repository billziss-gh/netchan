/*
 * weakmap_test.go
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
	"testing"
	"time"
)

func TestWeakmap(t *testing.T) {
	wmap := newWeakmap()

	ref := wmap.weakref(nil)
	if (weakref{}) != ref {
		t.Errorf("incorrect weakref: expect %v, got %v", weakref{}, ref)
	}

	c0 := make(chan error)
	r0 := wmap.weakref(c0)
	if (weakref{}) == r0 {
		t.Errorf("incorrect zero weakref")
	}

	ref = wmap.weakref(c0)
	if r0 != ref {
		t.Errorf("incorrect weakref: expect %v, got %v", r0, ref)
	}

	c1 := make(chan error)
	r1 := wmap.weakref(c1)
	if (weakref{}) == r1 {
		t.Errorf("incorrect zero weakref")
	}

	ref = wmap.weakref(c1)
	if r1 != ref {
		t.Errorf("incorrect weakref: expect %v, got %v", r1, ref)
	}

	if r0 == r1 {
		t.Errorf("incorrect equal weakrefs")
	}

	d0 := wmap.strongref(weakref{}, nil)
	if nil != d0 {
		t.Errorf("incorrect strongref: expect %v, got %v", nil, d0)
	}

	d0 = wmap.strongref(r0, nil)
	if c0 != d0 {
		t.Errorf("incorrect strongref: expect %v, got %v", c0, d0)
	}

	d1 := wmap.strongref(r1, nil)
	if c1 != d1 {
		t.Errorf("incorrect strongref: expect %v, got %v", c1, d1)
	}

	c0 = nil
	d0 = nil
	runtime.GC()
	time.Sleep(300 * time.Millisecond)

	d0 = wmap.strongref(r0, nil)
	if nil != d0 {
		t.Errorf("incorrect strongref: expect %v, got %v", nil, d0)
	}

	d1 = wmap.strongref(r1, nil)
	if c1 != d1 {
		t.Errorf("incorrect strongref: expect %v, got %v", c1, d1)
	}

	d0 = wmap.strongref(weakref{42}, func() interface{} {
		return make(chan string)
	})
	if nil == d0 {
		t.Errorf("incorrect nil strongref")
	}

	_ = d0.(chan string)

	if d0 == d1 {
		t.Errorf("incorrect equal strongrefs")
	}

	d1 = wmap.strongref(weakref{42}, nil)
	if d0 != d1 {
		t.Errorf("incorrect strongref: expect %v, got %v", d0, d1)
	}

	ref = wmap.weakref(d0)
	if (weakref{42}) != ref {
		t.Errorf("incorrect weakref: expect %v, got %v", weakref{42}, ref)
	}
}
