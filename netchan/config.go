/*
 * config.go
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
	"time"
)

const (
	configMaxMsgSize = 16 * 1024 * 1024
	configMaxLinks   = 4
)

// Config contains configuration parameters for a Transport.
type Config struct {
	// MaxLinks contains the maximum number of links that may be opened
	// to a particular address/URI.
	MaxLinks int

	// RedialTimeout contains a timeout for "redial" attempts. If it is
	// non-zero a Transport will retry dialing if a dialing error occurs
	// for at least the duration specified in RedialTimeout. If this
	// field is zero no redial attempts will be made.
	RedialTimeout time.Duration

	// IdleTimeout will close connections that have been idle for the
	// specified duration. If this field is zero idle connections will
	// not be closed.
	IdleTimeout time.Duration
}

// Clone makes a shallow clone of the receiver Config.
func (self *Config) Clone() *Config {
	clone := *self
	return &clone
}
