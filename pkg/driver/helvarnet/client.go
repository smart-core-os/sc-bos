package helvarnet

import (
	"context"
	"fmt"
	"net"
	"strings"
	"sync"
	"time"

	"go.uber.org/zap"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/smart-core-os/sc-bos/pkg/driver/helvarnet/config"
	"github.com/smart-core-os/sc-bos/pkg/gen"
	"github.com/smart-core-os/sc-golang/pkg/resource"
)

type tcpClient struct {
	addr   *net.TCPAddr
	cfg    *config.Root
	conn   net.Conn
	logger *zap.Logger
	mu     sync.Mutex
	status *resource.Value // *gen.StatusLog
}

func newTcpClient(addr *net.TCPAddr, l *zap.Logger, cfg *config.Root) *tcpClient {
	return &tcpClient{
		addr:   addr,
		cfg:    cfg,
		conn:   nil,
		logger: l,
		mu:     sync.Mutex{},
		status: resource.NewValue(resource.WithInitialValue(&gen.StatusLog{}), resource.WithNoDuplicates()),
	}
}

func (c *tcpClient) connect() error {
	conn, err := net.DialTimeout("tcp", c.addr.String(), c.cfg.ConnectTimeout.Duration)

	if err != nil {
		c.conn = nil
		return err
	}
	c.conn = conn
	return nil
}

func (c *tcpClient) close() {
	if c.conn != nil {
		err := c.conn.Close()
		if err != nil {
			c.logger.Warn("failed to close connection", zap.Error(err))
		}
		c.conn = nil
	}
}

// make 3 attempts to try to send pkt and receive response. want is the prefix of the response we are expecting.
// if no response is expected, want should be an empty string.
// if the response is not what we expect, we return an error
func (c *tcpClient) sendAndReceive(ctx context.Context, pkt string, want string) (string, error) {

	c.mu.Lock()
	defer c.mu.Unlock()
	for tries := 3; tries > 0; tries-- {
		select {
		case <-ctx.Done():
			return "", ctx.Err()
		default:
		}
		var err error
		err = c.sendPacket(pkt)
		if err != nil {
			c.logger.Error("failed to send packet", zap.Error(err))
			time.Sleep(c.cfg.RetrySleepDuration.Duration)
			c.close()
			continue
		}

		if want == "" {
			// If no response is expected, just return an empty string
			return "", nil
		}

		response, err := c.receivePacket()
		if err != nil {
			c.logger.Error("failed to receive packet", zap.Error(err))
			c.close()
		} else {
			if !(strings.HasPrefix(response, want) &&
				strings.HasSuffix(response, "#")) {
				c.logger.Error("unexpected response", zap.String("response", response),
					zap.String("want", want))
				time.Sleep(c.cfg.RetrySleepDuration.Duration)
				c.close()
				continue
			}
			_, _ = c.status.Set(&gen.StatusLog{
				Level:       gen.StatusLog_NOMINAL,
				RecordTime:  timestamppb.New(time.Now()),
				Description: "Communication with lighting server successful",
			})
			return response, nil
		}
		time.Sleep(c.cfg.RetrySleepDuration.Duration)
	}
	_, _ = c.status.Set(&gen.StatusLog{
		Level:       gen.StatusLog_NON_FUNCTIONAL,
		RecordTime:  timestamppb.New(time.Now()),
		Description: "Can't connect to the lighting server",
	})
	return "", fmt.Errorf("failed to send and receive packet")
}

// do not use this function directly it is not thread safe
// do all send and receive operations through sendAndReceive
func (c *tcpClient) sendPacketWithTimeout(packet string, duration time.Duration) error {

	if c.conn == nil {
		err := c.connect()
		if err != nil {
			return err
		}
	}

	err := c.conn.SetWriteDeadline(time.Now().Add(duration))
	if err != nil {
		c.logger.Warn("failed to set deadline", zap.Error(err))
		return err
	}
	data := []byte(packet)
	toWrite := len(data)
	written := 0
	for toWrite > 0 {
		n, err := c.conn.Write(data[written:])
		if err != nil {
			c.close()
			return err
		}
		written += n
		toWrite -= n
	}
	return nil
}

// do not use this function directly it is not thread safe
// do all send and receive operations through sendAndReceive
func (c *tcpClient) sendPacket(packet string) error {
	return c.sendPacketWithTimeout(packet, c.cfg.SendPacketTimeout.Duration)
}

// do not use this function directly it is not thread safe
// do all send and receive operations through sendAndReceive
func (c *tcpClient) receivePacket() (string, error) {
	return c.receivePacketWithTimeout(c.cfg.RxTimeout.Duration)
}

// do not use this function directly it is not thread safe
// do all send and receive operations through sendAndReceive
func (c *tcpClient) receivePacketWithTimeout(timeout time.Duration) (string, error) {

	if c.conn == nil {
		return "", fmt.Errorf("connection is nil when attempting to receive")
	}

	err := c.conn.SetReadDeadline(time.Now().Add(timeout))
	if err != nil {
		c.logger.Warn("failed to set deadline", zap.Error(err))
		return "", err
	}
	buf := make([]byte, *c.cfg.RxBufferSize)
	n, err := c.conn.Read(buf)
	if err != nil {
		c.close()
		return "", err
	}
	return string(buf[:n]), nil
}
