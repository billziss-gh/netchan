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

type Config struct {
	MaxLinks      int
	RedialTimeout time.Duration
	IdleTimeout   time.Duration
}

func (self *Config) Clone() *Config {
	clone := *self
	return &clone
}
