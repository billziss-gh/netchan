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

type errData struct {
	message string
	nested  error
	ichan   interface{}
}

func (err *errData) _args(args []interface{}) Err {
	var nested Err
	for _, arg := range args {
		switch a := arg.(type) {
		case string:
			err.message = a
		case Err:
			nested = a
			err.nested = a
		case error:
			err.nested = a
		default:
			err.ichan = a
		}
	}
	if 1 == len(args) {
		return nested
	}
	return nil
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

// MakeErrArgument makes a function/method argument error.
func MakeErrArgument(args ...interface{}) *ErrArgument {
	err := &ErrArgument{}
	n := err._args(args)
	if e, ok := n.(*ErrArgument); ok {
		return e
	}
	return err
}

// ErrPublisher encapsulates a publisher error.
// ErrPublisher implements the Err interface.
type ErrPublisher struct {
	errData
}

// MakeErrPublisher makes a publisher error.
func MakeErrPublisher(args ...interface{}) *ErrPublisher {
	err := &ErrPublisher{}
	n := err._args(args)
	if e, ok := n.(*ErrPublisher); ok {
		return e
	}
	return err
}

// ErrConnector encapsulates a connector error.
// ErrConnector implements the Err interface.
type ErrConnector struct {
	errData
}

// MakeErrConnector makes a connector error.
func MakeErrConnector(args ...interface{}) *ErrConnector {
	err := &ErrConnector{}
	n := err._args(args)
	if e, ok := n.(*ErrConnector); ok {
		return e
	}
	return err
}

// ErrTransport encapsulates a network transport error.
// ErrTransport implements the Err interface.
type ErrTransport struct {
	errData
}

// MakeErrTransport makes a network transport error.
func MakeErrTransport(args ...interface{}) *ErrTransport {
	err := &ErrTransport{}
	n := err._args(args)
	if e, ok := n.(*ErrTransport); ok {
		return e
	}
	return err
}

// ErrMarshaler encapsulates a message encoding/decoding error.
// ErrMarshaler implements the Err interface.
type ErrMarshaler struct {
	errData
}

// MakeErrMarshaler makes a message encoding/decoding error.
func MakeErrMarshaler(args ...interface{}) *ErrMarshaler {
	err := &ErrMarshaler{}
	n := err._args(args)
	if e, ok := n.(*ErrMarshaler); ok {
		return e
	}
	return err
}

// Errors reports by this package. Other errors are also possible.
// All errors reported implement the Err interface.
var (
	ErrArgumentInvalid             error = MakeErrArgument("netchan: argument: invalid")
	ErrConnectorChanConnected      error = MakeErrConnector("netchan: connector: chan is connected")
	ErrConnectorChanNotConnected   error = MakeErrConnector("netchan: connector: chan is not connected")
	ErrTransportInvalid            error = MakeErrTransport("netchan: transport: invalid")
	ErrTransportClosed             error = MakeErrTransport("netchan: transport: closed")
	ErrTransportUnexpectedResponse error = MakeErrTransport("netchan: transport: unexpected response")
	ErrTransportMessageCorrupt     error = MakeErrTransport("netchan: transport: message is corrupt")
	ErrMarshalerNoChanEncoder      error = MakeErrMarshaler("netchan: marshaler: chan encoder not set")
	ErrMarshalerNoChanDecoder      error = MakeErrMarshaler("netchan: marshaler: chan decoder not set")
	ErrMarshalerPanic              error = MakeErrMarshaler("netchan: marshaler: panic")
)
