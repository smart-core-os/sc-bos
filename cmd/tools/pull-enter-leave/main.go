// Command pull-enter-leave provides a CLI tool that pulls from a [traits.EnterLeaveSensorApiClient].
package main

import (
	"context"
	"crypto/tls"
	"log"
	"time"

	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"

	"github.com/smart-core-os/sc-api/go/traits"
	"github.com/smart-core-os/sc-bos/pkg/task"
)

func main() {
	logger, err := zap.NewDevelopment()
	if err != nil {
		panic(err)
	}
	task.Run(context.Background(), func(ctx context.Context) (task.Next, error) {
		tlsConfig := &tls.Config{
			InsecureSkipVerify: true,
		}
		conn, err := grpc.NewClient("localhost:23557", grpc.WithTransportCredentials(credentials.NewTLS(tlsConfig)))
		if err != nil {
			return 0, err
		}
		client := traits.NewEnterLeaveSensorApiClient(conn)
		stream, err := client.PullEnterLeaveEvents(ctx, &traits.PullEnterLeaveEventsRequest{Name: "enter-leave"})
		if err != nil {
			return 0, err
		}
		for {
			event, err := stream.Recv()
			if err != nil {
				return task.ResetBackoff, err
			}
			log.Printf("Got event: %+v", event)
		}
	}, task.WithRetry(task.RetryUnlimited), task.WithBackoff(time.Second, 30*time.Second), task.WithErrorLogger(logger))
}
