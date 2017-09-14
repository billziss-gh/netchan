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

// ErrArgument encapsulates a function/method argument error.
// ErrArgument implements the Err interface.
type ErrArgument struct {
	errData
}

// NewErrArgument creates a new function/method argument error.
func NewErrArgument(args ...interface{}) *ErrArgument {
	err := &ErrArgument{}
	err._args(args)
	return err
}

// ErrTransport encapsulates a network transport error.
// ErrTransport implements the Err interface.
type ErrTransport struct {
	errData
}

// NewErrTransport creates a new network transport error.
func NewErrTransport(args ...interface{}) *ErrTransport {
	err := &ErrTransport{}
	err._args(args)
	return err
}

// ErrMarshaler encapsulates a message encoding/decoding error.
// ErrMarshaler implements the Err interface.
type ErrMarshaler struct {
	errData
}

// NewErrMarshaler creates a new message encoding/decoding error.
func NewErrMarshaler(args ...interface{}) *ErrMarshaler {
	err := &ErrMarshaler{}
	err._args(args)
	return err
}

// Errors reports by this package. Other errors are also possible.
// All errors reported implement the Err interface.
var (
	ErrArgumentInvalid         error = NewErrArgument("netchan: argument is invalid")
	ErrArgumentConnected       error = NewErrArgument("netchan: argument chan is connected")
	ErrArgumentNotConnected    error = NewErrArgument("netchan: argument chan is not connected")
	ErrTransportInvalid        error = NewErrTransport("netchan: transport is invalid")
	ErrTransportClosed         error = NewErrTransport("netchan: transport is closed")
	ErrTransportMessageCorrupt error = NewErrTransport("netchan: transport message is corrupt")
	ErrMarshalerNoChanEncoder  error = NewErrMarshaler("netchan: marshaler chan encoder not set")
	ErrMarshalerNoChanDecoder  error = NewErrMarshaler("netchan: marshaler chan decoder not set")
	ErrMarshalerPanic          error = NewErrMarshaler("netchan: marshaler panic")
)
