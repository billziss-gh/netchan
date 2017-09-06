/*
 * errors.go
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

type errArgs interface {
	args(args ...interface{})
}

type errData struct {
	message string
	nested  error
	ichan   interface{}
}

func (err *errData) _args(args []interface{}) {
	for _, arg := range args {
		switch a := arg.(type) {
		case string:
			err.message = a
		case error:
			err.nested = a
		default:
			err.ichan = a
		}
	}
}

func (err *errData) args(args ...interface{}) {
	err._args(args)
}

func (err *errData) Error() string {
	if "" == err.message && nil != err.nested {
		return err.nested.Error()
	}
	return err.message
}

func (err *errData) Nested() error {
	return err.nested
}

func (err *errData) Chan() interface{} {
	return err.ichan
}

// Function/method argument errors.
type ErrArgument struct {
	errData
}

func newErrArgument(args ...interface{}) error {
	err := &ErrArgument{}
	err._args(args)
	return err
}

// Network transport errors.
type ErrTransport struct {
	errData
}

func newErrTransport(args ...interface{}) error {
	err := &ErrTransport{}
	err._args(args)
	return err
}

// Message encoding/decoding errors.
type ErrMarshaler struct {
	errData
}

func newErrMarshaler(args ...interface{}) error {
	err := &ErrMarshaler{}
	err._args(args)
	return err
}

var (
	ErrArgumentInvalid         error = newErrArgument("netchan: argument is invalid")
	ErrTransportInvalid        error = newErrTransport("netchan: transport is invalid")
	ErrTransportClosed         error = newErrTransport("netchan: transport is closed")
	ErrTransportMessageCorrupt error = newErrTransport("netchan: transport message is corrupt")
	ErrMarshalerNoChanEncoder  error = newErrMarshaler("netchan: marshaler chan encoder not set")
	ErrMarshalerNoChanDecoder  error = newErrMarshaler("netchan: marshaler chan decoder not set")
	ErrMarshalerRefInvalid     error = newErrMarshaler("netchan: marshaler ref is invalid")
	ErrMarshalerPanic          error = newErrMarshaler("netchan: marshaler panic")
)
