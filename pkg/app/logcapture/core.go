// Package logcapture provides a zapcore.Core that allows dynamically
// registering and deregistering additional observer cores at runtime.
// All log entries forwarded by the base core are also sent to any
// registered extras, making it possible to add a capturing core long
// after the root logger has been created.
package logcapture

import (
	"sync"

	"go.uber.org/zap/zapcore"
)

// Core wraps a base zapcore.Core and forwards every log entry to zero or
// more dynamically-registered extra cores.
type Core struct {
	base zapcore.Core

	mu    sync.RWMutex
	extra []zapcore.Core
}

// Wrap creates a new Core that wraps base.
func Wrap(base zapcore.Core) *Core {
	return &Core{base: base}
}

// WrapCoreOption returns a zap.Option that installs this Core as a wrapper
// around the logger's existing core.  The existing core becomes the base.
// Use as: zap.New(core, capture.WrapCoreOption()) or
// logger.WithOptions(capture.WrapCoreOption()).
// After Build() the Core's base field will be set.
func (c *Core) WrapCoreFunc() func(zapcore.Core) zapcore.Core {
	return func(base zapcore.Core) zapcore.Core {
		c.base = base
		return c
	}
}

// Add registers an extra core to receive log entries.
// Returns a cancel function that deregisters it.
func (c *Core) Add(extra zapcore.Core) func() {
	c.mu.Lock()
	c.extra = append(c.extra, extra)
	c.mu.Unlock()
	return func() {
		c.mu.Lock()
		defer c.mu.Unlock()
		for i, e := range c.extra {
			if e == extra {
				c.extra = append(c.extra[:i], c.extra[i+1:]...)
				return
			}
		}
	}
}

func (c *Core) Enabled(level zapcore.Level) bool {
	if c.base.Enabled(level) {
		return true
	}
	c.mu.RLock()
	defer c.mu.RUnlock()
	for _, e := range c.extra {
		if e.Enabled(level) {
			return true
		}
	}
	return false
}

// With returns a child core with the fields applied to the base.
// Extra cores are NOT given the With fields; they receive only per-Write fields.
// This is a deliberate trade-off: static context fields (e.g. service.id) are
// omitted from captured messages in favour of a simpler implementation.
func (c *Core) With(fields []zapcore.Field) zapcore.Core {
	return &childCore{
		base:   c.base.With(fields),
		parent: c,
	}
}

func (c *Core) Check(entry zapcore.Entry, ce *zapcore.CheckedEntry) *zapcore.CheckedEntry {
	ce = c.base.Check(entry, ce)
	c.mu.RLock()
	for _, e := range c.extra {
		ce = e.Check(entry, ce)
	}
	c.mu.RUnlock()
	return ce
}

func (c *Core) Write(entry zapcore.Entry, fields []zapcore.Field) error {
	return c.base.Write(entry, fields)
}

func (c *Core) Sync() error {
	err := c.base.Sync()
	c.mu.RLock()
	defer c.mu.RUnlock()
	for _, e := range c.extra {
		if syncErr := e.Sync(); syncErr != nil && err == nil {
			err = syncErr
		}
	}
	return err
}

// childCore is a Core with permanent With-fields applied to the base.
// Extra cores from the parent are resolved dynamically at Check time so that
// cores added after this child was created still receive entries.
type childCore struct {
	base   zapcore.Core
	parent *Core
}

func (c *childCore) Enabled(level zapcore.Level) bool {
	if c.base.Enabled(level) {
		return true
	}
	c.parent.mu.RLock()
	defer c.parent.mu.RUnlock()
	for _, e := range c.parent.extra {
		if e.Enabled(level) {
			return true
		}
	}
	return false
}

func (c *childCore) With(fields []zapcore.Field) zapcore.Core {
	return &childCore{
		base:   c.base.With(fields),
		parent: c.parent,
	}
}

func (c *childCore) Check(entry zapcore.Entry, ce *zapcore.CheckedEntry) *zapcore.CheckedEntry {
	ce = c.base.Check(entry, ce)
	c.parent.mu.RLock()
	for _, e := range c.parent.extra {
		ce = e.Check(entry, ce)
	}
	c.parent.mu.RUnlock()
	return ce
}

func (c *childCore) Write(entry zapcore.Entry, fields []zapcore.Field) error {
	return c.base.Write(entry, fields)
}

func (c *childCore) Sync() error {
	return c.base.Sync()
}
