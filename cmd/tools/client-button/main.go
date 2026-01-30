// Command client-button provides a CLI tool for interacting with a [buttonpb.ButtonApiClient].
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/chzyer/readline"
	"github.com/pborman/uuid"
	"golang.org/x/sync/errgroup"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/known/fieldmaskpb"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/smart-core-os/sc-bos/pkg/proto/buttonpb"
	"github.com/smart-core-os/sc-bos/pkg/util/client"
)

var (
	clientConfig    client.Config
	flagInteractive bool
)

func init() {
	flag.StringVar(&clientConfig.Endpoint, "endpoint", "localhost:23557", "smart core endpoint")
	flag.BoolVar(&clientConfig.Get, "get", true, "perform a get request")
	flag.BoolVar(&clientConfig.Pull, "pull", false, "pull changes")
	flag.StringVar(&clientConfig.Name, "name", "", "smart core name for requests")
	flag.BoolVar(&clientConfig.TLS.InsecureNoClientCert, "insecure-no-client-cert", false, "")
	flag.BoolVar(&clientConfig.TLS.InsecureSkipVerify, "insecure-skip-verify", false, "")
	flag.BoolVar(&flagInteractive, "interactive", false, "start an interactive repl for sending updates")
}

func main() {
	flag.Parse()

	err := run(context.Background())
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "error: %s\n", err.Error())
		os.Exit(1)
	}
}

func run(ctx context.Context) (err error) {
	var (
		rl  *readline.Instance
		out io.Writer
	)
	if flagInteractive {
		rl, err = readline.NewEx(&readline.Config{
			Prompt: "> ",
		})
		if err != nil {
			return err
		}
		out = rl.Stdout()
		defer func() {
			_ = rl.Close()
		}()
	} else {
		out = os.Stdout
	}

	conn, err := client.NewConnection(clientConfig)
	if err != nil {
		return fmt.Errorf("dial %s: %w", clientConfig.Endpoint, err)
	}

	buttonClient := buttonpb.NewButtonApiClient(conn)

	if clientConfig.Get {
		err = poll(ctx, buttonClient, out)
	}

	group, ctx := errgroup.WithContext(ctx)
	if clientConfig.Pull {
		group.Go(func() error {
		outer:
			for ctx.Err() == nil {
				stream, err := buttonClient.PullButtonState(ctx, &buttonpb.PullButtonStateRequest{Name: clientConfig.Name})
				if err != nil {
					_, _ = fmt.Fprintf(rl.Stderr(), "PullButtonState failed: %s\n", err.Error())
					time.Sleep(5 * time.Second)
					continue
				}

				for {
					res, err := stream.Recv()
					if err != nil {
						_, _ = fmt.Fprintf(rl.Stderr(), "PullButtonState failed: %s\n", err.Error())
						continue outer
					}

					for _, change := range res.Changes {
						_, _ = fmt.Fprintf(out, "PullButtonState change to %q at %v:\n%s\n",
							change.Name, change.ChangeTime.AsTime().String(), protojson.Format(change.ButtonState))
					}
				}
			}
			return ctx.Err()
		})
	}

	if flagInteractive {
		group.Go(func() error {
			for {
				line, err := rl.Readline()
				if err != nil {
					return err
				}

				parts := splitRegexp.Split(line, -1)
				if len(parts) == 0 {
					continue
				}
				cmd := parts[0]
				args := parts[1:]

				switch strings.ToLower(cmd) {
				case "get":
					err = poll(ctx, buttonClient, out)
				case "update":
					err = handleUpdate(ctx, args, buttonClient, out)
				default:
					err = fmt.Errorf("unknown command %q", cmd)
				}

				if err != nil {
					_, _ = fmt.Fprintln(rl.Stderr(), err.Error())
				}

			}
		})
	}

	return group.Wait()
}

func poll(ctx context.Context, client buttonpb.ButtonApiClient, out io.Writer) error {
	buttonState, err := client.GetButtonState(ctx, &buttonpb.GetButtonStateRequest{
		Name: clientConfig.Name,
	})
	if err != nil {
		return err
	}
	_, err = fmt.Fprintf(out, "GetButtonState({%q}):\n%s\n", clientConfig.Name, protojson.Format(buttonState))
	return err
}

var splitRegexp = regexp.MustCompile(`\s+`)

func handleUpdate(ctx context.Context, args []string, client buttonpb.ButtonApiClient, stdout io.Writer) error {
	if len(args) == 0 {
		return errors.New("must supply subcommand name")
	}

	subcmd := args[0]
	args = args[1:]

	switch strings.ToLower(subcmd) {
	case "click":
		return handleClick(ctx, args, client, stdout)
	default:
		return fmt.Errorf("unknown update subcommand %q", subcmd)
	}
}

func handleClick(ctx context.Context, args []string, client buttonpb.ButtonApiClient, stdout io.Writer) error {
	mask, err := fieldmaskpb.New(&buttonpb.ButtonState{},
		"most_recent_gesture",
		"most_recent_gesture.id",
		"most_recent_gesture.kind",
		"most_recent_gesture.count",
		"most_recent_gesture.start_time",
		"most_recent_gesture.end_time",
	)
	if err != nil {
		return err
	}

	count := 1
	if len(args) >= 1 {
		count, err = strconv.Atoi(args[0])
		if err != nil {
			return err
		}
	}

	t := timestamppb.Now()
	updated, err := client.UpdateButtonState(ctx, &buttonpb.UpdateButtonStateRequest{
		Name:       clientConfig.Name,
		UpdateMask: mask,
		ButtonState: &buttonpb.ButtonState{
			MostRecentGesture: &buttonpb.ButtonState_Gesture{
				Id:        uuid.New(),
				Kind:      buttonpb.ButtonState_Gesture_CLICK,
				Count:     int32(count),
				StartTime: t,
				EndTime:   t,
			},
		},
	})

	_, _ = fmt.Fprintf(stdout, "UpdateButtonState returned: error=%v, output:\n%s\n", err, protojson.Format(updated))
	return nil
}
