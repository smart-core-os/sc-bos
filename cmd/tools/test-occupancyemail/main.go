// Command test-occupancyemail tests the [occupancyemail] package, sending to a real email address.
package main

import (
	"bufio"
	"context"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"time"

	"go.uber.org/zap"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/smart-core-os/sc-api/go/traits"
	"github.com/smart-core-os/sc-bos/pkg/auto"
	"github.com/smart-core-os/sc-bos/pkg/auto/occupancyemail"
	"github.com/smart-core-os/sc-bos/pkg/gentrait/historypb"
	"github.com/smart-core-os/sc-bos/pkg/history/memstore"
	"github.com/smart-core-os/sc-bos/pkg/node"
	"github.com/smart-core-os/sc-bos/pkg/proto/occupancysensorpb"
	"github.com/smart-core-os/sc-golang/pkg/trait"
)

func main() {
	logger, err := zap.NewDevelopment()
	if err != nil {
		panic(err)
	}
	root := node.New("test")

	now, _ := time.Parse(time.DateTime, "2023-11-15 11:36:00")
	now = now.Round(time.Second) // get rid of millis, etc

	oc := func(age time.Duration, pc int) *traits.PullOccupancyResponse_Change {
		return &traits.PullOccupancyResponse_Change{
			ChangeTime: timestamppb.New(now.Add(-age)),
			Occupancy:  &traits.Occupancy{PeopleCount: int32(pc)},
		}
	}
	testData := []*traits.PullOccupancyResponse_Change{
		// note: these _must_ be in chronological order
		oc(7*24*time.Hour+time.Second, 20), // before the 7-day window
		oc(7*24*time.Hour-2*time.Second, 6),
		oc(7*24*time.Hour-2*time.Hour, 0),
		oc(7*24*time.Hour-3*time.Hour, 7),
		oc(3*24*time.Hour, 4),
		oc(-time.Second, 22), // in the future, just in case
	}

	// use sample prod data instead
	now = sampleNow
	testData = parseSampleData()

	store := memstore.New()
	for _, td := range testData {
		td := td
		memstore.SetNow(store, td.ChangeTime.AsTime)
		payload, _ := proto.Marshal(td.Occupancy)
		_, err := store.Append(nil, payload)
		if err != nil {
			panic(err)
		}
	}
	device := historypb.NewOccupancySensorServer(store)
	client := occupancysensorpb.WrapHistory(device)
	root.Announce("test", node.HasTrait(trait.OccupancySensor, node.WithClients(client)))

	serv := auto.Services{
		Logger: logger,
		Node:   root,
		Now: func() time.Time {
			return now.Add(-2 * time.Second)
		},
	}
	lifecycle := occupancyemail.Factory.New(serv)
	defer lifecycle.Stop()
	cfg := `{
  "name": "emails", "type": "occupancyemail",
  "source": {
    "name": "test",
    "title": "Enterprise Wharf"
  },
  "destination": {
    "host": "smtp.gmail.com",
    "from": "Enterprise Wharf <no-reply@enterprisewharf.co.uk>",
    "to": ["Matt Nathan <matt.nathan@vanti.co.uk>"],
    "passwordFile": ".secrets/ew-email-pass",
    "sendTime": "0 0 * * MON-FRI"
  }
}`
	_, err = lifecycle.Configure([]byte(cfg))
	if err != nil {
		panic(err)
	}
	_, err = lifecycle.Start()
	if err != nil {
		panic(err)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()
	<-ctx.Done()
}

var sampleData = `
2023-11-16T15:13:39Z,1
2023-11-21T11:38:22Z,0
2023-11-21T11:53:49Z,0
2023-11-21T11:54:01Z,0
2023-11-21T12:05:03Z,0
2023-11-21T12:05:33Z,0
2023-11-21T12:08:47Z,0
2023-11-21T12:09:29Z,0
2023-11-21T13:10:03Z,0
2023-11-21T13:10:50Z,0
2023-11-21T13:10:52Z,0
2023-11-21T13:12:48Z,0
2023-11-21T13:12:52Z,0
2023-11-21T13:17:21Z,0
2023-11-21T13:18:41Z,0
2023-11-21T13:23:20Z,0
2023-11-21T13:33:37Z,0
2023-11-21T13:35:14Z,0
2023-11-21T13:35:19Z,0
2023-11-21T13:35:45Z,0
2023-11-21T13:35:49Z,0
2023-11-21T13:36:07Z,0
2023-11-21T13:36:25Z,0
2023-11-21T13:38:09Z,0
2023-11-21T13:39:00Z,0
2023-11-21T13:39:34Z,0
2023-11-21T13:46:49Z,0
2023-11-21T13:47:01Z,0
2023-11-21T13:53:38Z,0
2023-11-21T13:53:50Z,0
2023-11-21T14:25:08Z,1
2023-11-21T14:26:10Z,2
2023-11-21T14:37:40Z,1
2023-11-21T15:52:57Z,0
2023-11-21T15:59:19Z,0
2023-11-21T16:20:37Z,0
2023-11-21T18:15:54Z,0
2023-11-21T18:16:06Z,0
2023-11-21T18:18:42Z,0
2023-11-21T18:18:54Z,0
2023-11-21T19:16:00Z,0
2023-11-21T19:16:12Z,0
2023-11-21T19:55:54Z,0
2023-11-21T19:56:06Z,0
2023-11-21T20:45:46Z,0
2023-11-21T20:45:58Z,0
2023-11-21T20:54:35Z,0
2023-11-21T20:54:47Z,0
2023-11-21T21:58:12Z,0
2023-11-21T21:58:25Z,0
2023-11-21T23:13:19Z,0
2023-11-21T23:13:31Z,0
2023-11-21T23:42:40Z,0
2023-11-21T23:42:52Z,0
2023-11-22T00:42:50Z,0
2023-11-22T00:43:02Z,0
2023-11-22T02:04:54Z,0
2023-11-22T02:05:06Z,0
2023-11-22T05:05:57Z,0
2023-11-22T05:06:09Z,0
2023-11-22T06:08:21Z,0
2023-11-22T06:08:32Z,0
2023-11-22T06:17:07Z,0
2023-11-22T06:17:27Z,0
2023-11-22T06:19:03Z,0
2023-11-22T06:19:15Z,0
2023-11-22T06:20:20Z,0
2023-11-22T06:20:32Z,0
2023-11-22T06:20:56Z,0
2023-11-22T06:21:17Z,0
2023-11-22T06:21:33Z,0
2023-11-22T06:21:45Z,0
2023-11-22T06:25:05Z,0
2023-11-22T06:25:16Z,0
2023-11-22T06:26:55Z,0
2023-11-22T06:27:07Z,0
2023-11-22T06:28:24Z,0
2023-11-22T06:28:36Z,0
2023-11-22T06:31:47Z,0
2023-11-22T06:32:00Z,0
2023-11-22T06:34:53Z,0
2023-11-22T06:35:04Z,0
2023-11-22T06:38:15Z,0
2023-11-22T06:38:30Z,0
2023-11-22T06:42:28Z,0
2023-11-22T06:42:39Z,0
2023-11-22T06:47:35Z,0
2023-11-22T06:47:46Z,0
2023-11-22T06:55:26Z,0
2023-11-22T06:55:38Z,0
2023-11-22T06:58:14Z,0
2023-11-22T06:58:26Z,0
2023-11-22T07:02:33Z,0
2023-11-22T07:02:45Z,0
2023-11-22T07:06:15Z,0
2023-11-22T07:06:27Z,0
2023-11-22T07:12:14Z,0
2023-11-22T07:12:26Z,0
2023-11-22T07:12:40Z,0
2023-11-22T07:12:52Z,0
2023-11-22T07:21:19Z,0
2023-11-22T07:21:31Z,0
2023-11-22T07:55:19Z,0
2023-11-22T07:55:31Z,0
2023-11-22T08:22:10Z,0
2023-11-22T08:22:22Z,0
2023-11-22T08:57:23Z,0
2023-11-22T08:58:45Z,0
2023-11-22T09:07:06Z,0
2023-11-22T09:07:53Z,0
2023-11-22T09:08:12Z,0
2023-11-22T09:09:58Z,0
2023-11-22T09:10:11Z,0
2023-11-22T09:11:19Z,0
2023-11-22T09:12:56Z,0
2023-11-22T09:13:08Z,0
2023-11-22T09:15:06Z,0
2023-11-22T09:15:36Z,0
2023-11-22T09:15:59Z,0
2023-11-22T09:16:38Z,0
2023-11-22T09:24:04Z,0
2023-11-22T09:24:16Z,0
2023-11-22T09:26:40Z,0
2023-11-22T09:30:46Z,0
2023-11-22T09:32:32Z,0
2023-11-22T09:33:05Z,0
2023-11-22T09:33:20Z,0
2023-11-22T09:41:59Z,0
2023-11-22T09:44:38Z,0
2023-11-22T09:46:46Z,0
2023-11-22T10:20:42Z,0
2023-11-22T10:21:15Z,0
2023-11-22T10:23:33Z,0
2023-11-22T10:24:49Z,0
2023-11-22T10:30:58Z,0
2023-11-22T10:32:54Z,0
2023-11-22T10:38:08Z,0
2023-11-22T10:38:47Z,0
2023-11-22T10:47:38Z,0
2023-11-22T10:49:33Z,0
2023-11-22T10:56:44Z,0
2023-11-22T10:56:55Z,0
2023-11-22T11:09:54Z,0
2023-11-22T11:10:06Z,0
2023-11-22T11:35:21Z,0
2023-11-22T11:35:50Z,0
2023-11-22T11:36:00Z,0
2023-11-22T11:36:32Z,0
2023-11-22T11:37:02Z,0
2023-11-22T11:37:31Z,0
2023-11-22T11:37:34Z,0
2023-11-22T11:38:01Z,0
2023-11-22T13:00:32Z,0
2023-11-22T13:01:18Z,0
2023-11-22T13:02:02Z,0
2023-11-22T13:02:23Z,0
2023-11-22T13:06:42Z,0
2023-11-22T13:07:31Z,0
2023-11-22T13:08:04Z,0
2023-11-22T13:08:18Z,0
2023-11-22T13:09:02Z,0
2023-11-22T13:10:00Z,0
2023-11-22T13:11:33Z,0
2023-11-22T13:13:50Z,0
2023-11-22T14:02:00Z,0
2023-11-22T14:02:12Z,0
2023-11-22T14:03:21Z,0
2023-11-22T14:04:04Z,0
2023-11-22T14:07:14Z,0
2023-11-22T14:07:47Z,0
2023-11-22T14:08:22Z,0
2023-11-22T14:09:02Z,0
2023-11-22T14:37:43Z,0
2023-11-22T14:39:47Z,0
2023-11-22T14:41:35Z,0
2023-11-22T14:42:17Z,0
2023-11-22T14:42:21Z,0
2023-11-22T14:43:48Z,0
2023-11-22T14:46:27Z,0
2023-11-22T14:59:12Z,0
2023-11-22T14:59:35Z,0
2023-11-22T14:59:52Z,0
2023-11-22T15:00:23Z,0
2023-11-22T15:01:29Z,0
2023-11-22T15:02:17Z,0
2023-11-22T15:10:09Z,0
2023-11-22T15:10:10Z,0
2023-11-22T15:12:41Z,0
2023-11-22T15:12:44Z,0
2023-11-22T15:13:21Z,0
2023-11-22T15:15:02Z,0
2023-11-22T15:15:38Z,0
2023-11-22T15:18:08Z,0
2023-11-22T15:19:00Z,0
2023-11-22T16:20:02Z,0
2023-11-22T16:20:14Z,0
2023-11-22T16:32:04Z,0
2023-11-22T16:32:16Z,0
2023-11-22T16:34:47Z,0
2023-11-22T16:35:41Z,0
2023-11-22T16:38:08Z,0
2023-11-22T16:38:20Z,0
2023-11-22T16:46:41Z,0
2023-11-22T16:46:53Z,0
2023-11-22T16:46:53Z,0
2023-11-22T16:47:05Z,0
2023-11-22T16:48:24Z,0
2023-11-22T16:48:36Z,0
2023-11-22T16:49:31Z,0
2023-11-22T16:49:43Z,0
2023-11-22T16:57:28Z,0
2023-11-22T16:57:40Z,0
2023-11-22T18:45:43Z,0
2023-11-22T18:45:55Z,0
2023-11-22T19:22:37Z,0
2023-11-22T19:22:49Z,0
2023-11-22T20:25:04Z,0
2023-11-22T20:25:16Z,0
2023-11-22T21:15:08Z,0
2023-11-22T21:15:19Z,0
2023-11-22T22:05:55Z,0
2023-11-22T22:06:06Z,0
2023-11-22T23:04:17Z,0
2023-11-22T23:04:29Z,0
2023-11-22T23:11:47Z,0
2023-11-22T23:11:59Z,0
2023-11-22T23:28:31Z,0
2023-11-22T23:28:43Z,0
`
var sampleNow = time.Date(2023, 11, 23, 0, 0, 0, 0, time.Local)

func parseSampleData() []*traits.PullOccupancyResponse_Change {
	var records []*traits.PullOccupancyResponse_Change
	scanner := bufio.NewScanner(strings.NewReader(sampleData))
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}
		parts := strings.Split(line, ",")
		t, err := time.Parse(time.RFC3339, parts[0])
		if err != nil {
			panic(err)
		}
		pc, err := strconv.Atoi(parts[1])
		if err != nil {
			panic(err)
		}
		records = append(records, &traits.PullOccupancyResponse_Change{
			ChangeTime: timestamppb.New(t),
			Occupancy:  &traits.Occupancy{PeopleCount: int32(pc)},
		})
	}
	return records
}
