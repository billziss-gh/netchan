/*
 * transport.go
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
	"io"
	"net/url"
)

func readMsg(r io.Reader) ([]byte, error) {
	buf := [4]byte{}
	_, err := io.ReadFull(r, buf[:])
	if nil != err {
		return nil, new(ErrTransport).nested(err)
	}

	n := int(buf[0]) | (int(buf[1]) << 8) | (int(buf[2]) << 16) | (int(buf[3]) << 24)
	if 4 > n || configMaxMsgSize < n {
		return nil, ErrTransportMessageCorrupt
	}

	msg := make([]byte, n)
	_, err = io.ReadFull(r, msg[4:])
	if nil != err {
		return nil, new(ErrTransport).nested(err)
	}
	msg[0] = buf[0]
	msg[1] = buf[1]
	msg[2] = buf[2]
	msg[3] = buf[3]

	return msg, nil
}

func writeMsg(w io.Writer, msg []byte) error {
	n := len(msg)
	if 4 > n || configMaxMsgSize < n {
		return ErrTransportMessageCorrupt
	}

	msg[0] = byte(n & 0xff)
	msg[1] = byte((n >> 8) & 0xff)
	msg[2] = byte((n >> 16) & 0xff)
	msg[3] = byte((n >> 24) & 0xff)

	_, err := w.Write(msg)
	if nil != err {
		return new(ErrTransport).nested(err)
	}

	return nil
}

var DefaultTransport Transport = newNetTransport(
	DefaultMarshaler,
	&url.URL{
		Scheme: "tcp",
		Host:   ":25454",
	})
