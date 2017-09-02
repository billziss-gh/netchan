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
	r0 := weakref(c0)
	if 0 == r0 {
		t.Errorf("incorrect non-nil weakref")
	}

	ref = weakref(c0)
	if r0 != ref {
		t.Errorf("incorrect weakref: expect %v, got %v", r0, ref)
	}

	c1 := make(chan error)
	r1 := weakref(c1)
	if 0 == r1 {
		t.Errorf("incorrect non-nil weakref")
	}

	ref = weakref(c1)
	if r1 != ref {
		t.Errorf("incorrect weakref: expect %v, got %v", r1, ref)
	}

	if r0 == r1 {
		t.Errorf("incorrect equal weakrefs")
	}

	d0 := strongref(r0)
	if c0 != d0 {
		t.Errorf("incorrect weakref: expect %v, got %v", c0, d0)
	}

	d1 := strongref(r1)
	if c1 != d1 {
		t.Errorf("incorrect weakref: expect %v, got %v", c1, d1)
	}

	c0 = nil
	d0 = nil
	runtime.GC()
	time.Sleep(300 * time.Millisecond)

	d0 = strongref(r0)
	if nil != d0 {
		t.Errorf("incorrect weakref: expect %v, got %v", nil, d0)
	}

	d1 = strongref(r1)
	if c1 != d1 {
		t.Errorf("incorrect weakref: expect %v, got %v", c1, d1)
	}
}
