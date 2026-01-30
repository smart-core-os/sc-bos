package transport

import (
	"time"

	"github.com/smart-core-os/sc-bos/pkg/util/jsontypes"
)

type ConnectionConfig struct {
	// Timeout is the dial timeout when attempting connection, defaults to 15s
	Timeout jsontypes.Duration
	// ReadTimeout is the timeout used when attempting to read, defaults to 0
	ReadTimeout jsontypes.Duration
	// WriteTimeout is the timeout used when attempting to write, defaults to 15s
	WriteTimeout jsontypes.Duration
}

func (c *ConnectionConfig) defaults() {
	if c.Timeout.Duration == 0 {
		c.Timeout.Duration = time.Second * 15
	}
	if c.WriteTimeout.Duration == 0 {
		c.WriteTimeout.Duration = time.Second * 15
	}
}
