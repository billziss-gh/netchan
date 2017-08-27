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

type ErrArgument struct {
	Err
}

type ErrTransport struct {
	Err
}

type ErrMarshaler struct {
	Err
}

var (
	ErrArgumentInvalid         error = new(ErrArgument).message("argument is invalid")
	ErrTransportInvalid        error = new(ErrTransport).message("transport is invalid")
	ErrTransportClosed         error = new(ErrTransport).message("transport is closed")
	ErrTransportMessageCorrupt error = new(ErrTransport).message("transport message is corrupt")
	ErrMarshalerPanic          error = new(ErrMarshaler).message("marshaler panic")
)

func (err *Err) message(message string) *Err {
	err.Message = message
	return err
}

func (err *Err) nested(nested error) *Err {
	err.Nested = nested
	return err
}

func (err *Err) Error() string {
	if "" == err.Message && nil != err.Nested {
		return err.Nested.Error()
	}
	return err.Message
}
