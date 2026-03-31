package transport

import (
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/cenkalti/backoff/v4"
	"go.uber.org/zap"
)

// Tcp implements Transport, back by a tcp connection. Reconnecting is handled automatically, with an exponential
// backoff for retries.
type Tcp struct {
	*Connection
	conf   TcpConfig
	logger *zap.Logger

	// control access to conn
	lock sync.RWMutex
	conn *net.Conn
}

// NewTcp creates a new Tcp transport with the given config
func NewTcp(conf TcpConfig, logger *zap.Logger) *Tcp {
	conf.defaults()
	bo := backoff.NewExponentialBackOff()
	bo.MaxElapsedTime = 0 // never give up
	tcp := &Tcp{
		conf:   conf,
		logger: logger,
	}
	conn := NewConnection(conf.ConnectionConfig, bo, tcp.dial, readWriteCloserFuncs(tcp.read, tcp.write, tcp.close))
	tcp.Connection = conn
	return tcp
}

func (t *Tcp) read(p []byte) (n int, err error) {
	t.lock.RLock()
	c := *t.conn
	t.lock.RUnlock()

	if t.conf.ReadTimeout.Duration != 0 {
		err = c.SetReadDeadline(time.Now().Add(t.conf.ReadTimeout.Duration))
		if err != nil {
			t.state.update(Disconnected)
			return 0, err
		}
	}
	return c.Read(p)
}

func (t *Tcp) write(p []byte) (n int, err error) {
	t.lock.RLock()
	c := *t.conn
	t.lock.RUnlock()

	if t.conf.WriteTimeout.Duration != 0 {
		err = c.SetWriteDeadline(time.Now().Add(t.conf.WriteTimeout.Duration))
		if err != nil {
			return 0, err
		}
	}
	return c.Write(p)
}

func (t *Tcp) close() error {
	t.lock.Lock()
	defer t.lock.Unlock()
	if t.conn != nil {
		err := (*t.conn).Close()
		t.conn = nil
		return err
	}
	return nil
}

func (t *Tcp) dial() error {
	address := net.JoinHostPort(t.conf.Ip, fmt.Sprintf("%d", t.conf.Port))
	t.logger.Debug("dialling", zap.String("address", address))
	t.lock.Lock()
	defer t.lock.Unlock()
	if t.conn != nil {
		_ = (*t.conn).Close()
		t.conn = nil
	}
	conn, err := net.DialTimeout("tcp", address, t.conf.Timeout.Duration)
	if err != nil {
		return err
	}
	t.logger.Debug("connected")
	t.conn = &conn

	return nil
}
