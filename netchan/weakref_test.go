/*
 * weakref_test.go
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

func TestWeakref(t *testing.T) {
	ref := weakref(nil)
	if 0 != ref {
		t.Errorf("incorrect weakref: expect %v, got %v", 0, ref)
	}

	c0 := make(chan error)
	p0 := &c0

	r0 := weakref(p0)
	if 0 == r0 {
		t.Errorf("incorrect nil weakref")
	}

	ref = weakref(p0)
	if r0 != ref {
		t.Errorf("incorrect weakref: expect %v, got %v", r0, ref)
	}

	c1 := make(chan error)
	p1 := &c1

	r1 := weakref(p1)
	if 0 == r1 {
		t.Errorf("incorrect nil weakref")
	}

	ref = weakref(p1)
	if r1 != ref {
		t.Errorf("incorrect weakref: expect %v, got %v", r1, ref)
	}

	if r0 == r1 {
		t.Errorf("incorrect equal weakrefs")
	}

	q0 := strongref(r0)
	if p0 != q0 {
		t.Errorf("incorrect weakref: expect %v, got %v", p0, q0)
	}

	q1 := strongref(r1)
	if p1 != q1 {
		t.Errorf("incorrect weakref: expect %v, got %v", p0, q0)
	}

	p0 = nil
	q0 = nil
	runtime.GC()
	time.Sleep(300 * time.Millisecond)

	q0 = strongref(r0)
	if nil != q0 {
		t.Errorf("incorrect weakref: expect %v, got %v", nil, q0)
	}

	q1 = strongref(r1)
	if p1 != q1 {
		t.Errorf("incorrect weakref: expect %v, got %v", p0, q0)
	}
}
