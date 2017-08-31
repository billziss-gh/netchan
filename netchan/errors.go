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

type Err struct {
	Message string
	Nested  error
}

func (err *Err) args(args []interface{}) {
	for _, arg := range args {
		switch a := arg.(type) {
		case string:
			err.Message = a
		case error:
			err.Nested = a
		}
	}
}

func (err *Err) Error() string {
	if "" == err.Message && nil != err.Nested {
		return err.Nested.Error()
	}
	return err.Message
}

type ErrArgument struct {
	Err
}

func newErrArgument(args ...interface{}) error {
	err := &ErrArgument{}
	err.args(args)
	return err
}

type ErrTransport struct {
	Err
}

func newErrTransport(args ...interface{}) error {
	err := &ErrTransport{}
	err.args(args)
	return err
}

type ErrMarshaler struct {
	Err
}

func newErrMarshaler(args ...interface{}) error {
	err := &ErrMarshaler{}
	err.args(args)
	return err
}

var (
	ErrArgumentInvalid         error = newErrArgument("argument is invalid")
	ErrTransportInvalid        error = newErrTransport("transport is invalid")
	ErrTransportClosed         error = newErrTransport("transport is closed")
	ErrTransportMessageCorrupt error = newErrTransport("transport message is corrupt")
	ErrMarshalerPanic          error = newErrMarshaler("marshaler panic")
)
