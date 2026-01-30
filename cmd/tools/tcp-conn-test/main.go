// Command tcp-conn-test provides an interactive CLI terminal for testing TCP connections.
package main

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"log"
	"net/textproto"
	"os"
	"os/signal"

	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"

	"github.com/smart-core-os/sc-bos/pkg/util/transport"
)

var (
	conf transport.TcpConfig
)

func init() {
	flag.StringVar(&conf.Ip, "ip", "localhost", "ip address")
	flag.IntVar(&conf.Port, "port", 9000, "port")
}

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	tcp := transport.NewTcp(conf, zap.L())

	reader := bufio.NewReader(os.Stdin)
	grp, ctx := errgroup.WithContext(ctx)
	grp.Go(func() error {
		return tcp.Connect(ctx)
	})
	grp.Go(func() error {
		reader := bufio.NewReader(tcp)
		tp := textproto.NewReader(reader)
		for {
			log.Printf("read line")
			line, err := tp.ReadLine()
			if err != nil {
				log.Printf("got error: %s", err)
			} else {
				log.Printf("got line: %s", line)
			}
		}
	})
	grp.Go(func() error {
		for {
			text, _ := reader.ReadString('\n')
			log.Printf("write: %s", text)
			n, err := tcp.Write([]byte(text))
			if err != nil {
				log.Printf("write error: %s", err)
			} else {
				log.Printf("written %d", n)
			}
		}
	})
	grp.Go(func() error {
		prev := transport.Idle
		for {
			state, change := tcp.WaitForStateChange(ctx, prev)
			if !change {
				return fmt.Errorf("no change")
			}
			prev = state
			if state == transport.Connected {
				log.Printf("connected")
			} else if state == transport.Disconnected {
				log.Printf("disconnected")
			}
		}
	})
	err := grp.Wait()
	if err != nil {
		log.Printf("%s", err)
	}
}
