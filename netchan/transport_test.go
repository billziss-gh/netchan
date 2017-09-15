/*
 * transport_test.go
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
	"crypto/tls"
	"net/url"
	"reflect"
	"sync/atomic"
	"testing"
	"time"
)

type testTransportRecverSender struct {
	id       string
	msg      string
	t        *testing.T
	rcnt     uint32
	done     chan struct{}
	doneflag bool
}

func newTestTransportRecverSender(id string, msg string, t *testing.T) *testTransportRecverSender {
	return &testTransportRecverSender{
		id,
		msg,
		t,
		0,
		make(chan struct{}),
		false,
	}
}

func (self *testTransportRecverSender) transportRecver(link Link) error {
	id, vmsg, err := link.Recv()
	if nil != err {
		if self.doneflag {
			return ErrTransportClosed
		}
		self.t.Errorf("Recver: error %v", err)
		return err
	}
	if id != self.id {
		self.t.Errorf("Recver: id mismatch")
	}
	msg := vmsg.Interface()
	if !reflect.DeepEqual(msg, self.msg) {
		self.t.Errorf("Recver: msg mismatch")
	}

	if 2 == atomic.AddUint32(&self.rcnt, 1) {
		self.doneflag = true
		close(self.done)
	}

	return nil
}

func (self *testTransportRecverSender) transportSender(link Link) error {
	err := link.Send(self.id, reflect.ValueOf(self.msg))
	if nil != err {
		self.t.Errorf("Sender: error %v", err)
		return err
	}

	<-self.done

	return ErrTransportClosed
}

func testTransport(t *testing.T, transport Transport, scheme string) {
	id0 := "NAME"
	msg := "42,43,44,45,46"
	trs := newTestTransportRecverSender(id0, msg, t)
	transport.SetRecver(trs.transportRecver)
	transport.SetSender(trs.transportSender)

	err := transport.Listen()
	if nil != err {
		panic(err)
	}
	err = transport.Listen()
	if nil != err {
		panic(err)
	}

	id, link, err := transport.Connect(&url.URL{
		Scheme: scheme,
		Host:   "127.0.0.1",
		Path:   "/" + id0,
	})
	if nil != err {
		panic(err)
	}

	if id0 != id {
		t.Errorf("Connect: incorrect id: %v", id)
	}

	link.Activate()
	link.Activate()

	<-trs.done
}

func TestTcpTransport(t *testing.T) {
	marshaler := NewGobMarshaler()
	transport := NewNetTransport(
		marshaler,
		&url.URL{
			Scheme: "tcp",
			Host:   ":25000",
		},
		nil)
	defer func() {
		transport.Close()
		time.Sleep(100 * time.Millisecond)
	}()

	testTransport(t, transport, "tcp")
}

func TestTlsTransport(t *testing.T) {
	cert, err := tls.X509KeyPair([]byte(tlscert), []byte(tlskey))
	if nil != err {
		panic(err)
	}

	marshaler := NewGobMarshaler()
	transport := NewNetTransportTLS(
		marshaler,
		&url.URL{
			Scheme: "tls",
			Host:   ":25000",
		},
		nil,
		&tls.Config{
			Certificates:       []tls.Certificate{cert},
			InsecureSkipVerify: true,
		})
	defer func() {
		transport.Close()
		time.Sleep(100 * time.Millisecond)
	}()

	testTransport(t, transport, "tls")
}

type testTransportIdleRecverSender struct {
	id   string
	msg  string
	t    *testing.T
	rcnt uint32
	done chan struct{}
}

func newTestTransportIdleRecverSender(
	id string, msg string, t *testing.T) *testTransportIdleRecverSender {
	return &testTransportIdleRecverSender{
		id,
		msg,
		t,
		0,
		make(chan struct{}),
	}
}

func (self *testTransportIdleRecverSender) transportRecver(link Link) error {
	id, vmsg, err := link.Recv()
	if nil != err {
		if 2 == atomic.AddUint32(&self.rcnt, 1) {
			close(self.done)
		}

		return ErrTransportClosed
	}
	if id != self.id {
		self.t.Errorf("Recver: id mismatch")
	}
	msg := vmsg.Interface()
	if !reflect.DeepEqual(msg, self.msg) {
		self.t.Errorf("Recver: msg mismatch")
	}

	return nil
}

func (self *testTransportIdleRecverSender) transportSender(link Link) error {
	err := link.Send(self.id, reflect.ValueOf(self.msg))
	if nil != err {
		self.t.Errorf("Sender: error %v", err)
		return err
	}

	<-self.done

	return ErrTransportClosed
}

func testTransportIdle(t *testing.T, transport Transport, scheme string) {
	id0 := "NAME"
	msg := "42,43,44,45,46"
	trs := newTestTransportIdleRecverSender(id0, msg, t)
	transport.SetRecver(trs.transportRecver)
	transport.SetSender(trs.transportSender)

	err := transport.Listen()
	if nil != err {
		panic(err)
	}

	_, link, err := transport.Connect(&url.URL{
		Scheme: scheme,
		Host:   "127.0.0.1",
		Path:   "/" + id0,
	})
	if nil != err {
		panic(err)
	}

	link.Activate()

	<-trs.done

	link.Dereference()
}

func TestTcpTransportIdle(t *testing.T) {
	marshaler := NewGobMarshaler()
	transport := NewNetTransport(
		marshaler,
		&url.URL{
			Scheme: "tcp",
			Host:   ":25000",
		},
		&Config{IdleTimeout: time.Second})
	defer func() {
		transport.Close()
		time.Sleep(100 * time.Millisecond)
		// transport must have garbage collected its links now!
		if 0 != len(transport.(*netTransport).mlink) {
			t.Errorf("netTransport GC fail: len(mlink): expect %v, got %v",
				0, len(transport.(*netTransport).mlink))
		}
	}()

	testTransportIdle(t, transport, "tcp")
}

type testTransportRedialRecverSender struct {
	t    *testing.T
	done chan struct{}
}

func newtestTransportRedialRecverSender(t *testing.T) *testTransportRedialRecverSender {
	return &testTransportRedialRecverSender{
		t,
		make(chan struct{}),
	}
}

func (self *testTransportRedialRecverSender) transportRecver(link Link) error {
	link.Recv()
	return ErrTransportClosed
}

func (self *testTransportRedialRecverSender) transportSender(link Link) error {
	err := link.Send("id", reflect.ValueOf("msg"))
	if nil == err {
		self.t.Error("Sender: expect error, got nil")
	}

	close(self.done)
	return ErrTransportClosed
}

func testTransportRedial(t *testing.T, transport Transport, scheme string) {
	now := time.Now()
	trs := newtestTransportRedialRecverSender(t)
	transport.SetRecver(trs.transportRecver)
	transport.SetSender(trs.transportSender)

	err := transport.Listen()
	if nil != err {
		panic(err)
	}

	_, link, err := transport.Connect(&url.URL{
		Scheme: scheme,
		Host:   "127.0.0.1:25001",
		Path:   "/id",
	})
	if nil != err {
		panic(err)
	}

	link.Activate()

	<-trs.done

	if time.Second > time.Now().Sub(now) {
		t.Error("expect 1 second of redial attempts; got less")
	}
}

func TestTcpTransportRedial(t *testing.T) {
	marshaler := NewGobMarshaler()
	transport := NewNetTransport(
		marshaler,
		&url.URL{
			Scheme: "tcp",
			Host:   ":25000",
		},
		&Config{RedialTimeout: time.Second})
	defer func() {
		transport.Close()
		time.Sleep(100 * time.Millisecond)
	}()

	testTransportRedial(t, transport, "tcp")
}
