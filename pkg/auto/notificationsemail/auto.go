// Package notificationsemail has been made to tick a box and needs improvement.
// provides an automation that send a CSV list of all notifications recorded by smart core
// in the previous month. This should be run on the 1st day of the month as it fetches the notifications for
// the previous month, not the previous ~30 days
// Test program for meter reading automation is in 'cmd/tools/test-notificationsemail/main.go'.
package notificationsemail

import (
	"bytes"
	"context"
	"fmt"
	"sort"
	"time"

	"go.uber.org/zap"

	"github.com/smart-core-os/sc-bos/pkg/auto"
	"github.com/smart-core-os/sc-bos/pkg/auto/notificationsemail/config"
	"github.com/smart-core-os/sc-bos/pkg/proto/alertpb"
	"github.com/smart-core-os/sc-bos/pkg/task"
	"github.com/smart-core-os/sc-bos/pkg/task/service"
)

const AutoName = "notificationsemail"

var Factory auto.Factory = factory{}

type factory struct{}

type autoImpl struct {
	*service.Service[config.Root]
	auto.Services
}

func (f factory) New(services auto.Services) service.Lifecycle {
	a := &autoImpl{Services: services}
	a.Service = service.New(service.MonoApply(a.applyConfig), service.WithParser(config.ReadBytes))
	a.Logger = a.Logger.Named(AutoName)
	return a
}

func isAlertInPreviousMonth(t, now time.Time) bool {
	return t.Month() == prevMonth(now, 1)
}

func isAlert2MonthsAgo(t, now time.Time) bool {
	return t.Month() == prevMonth(now, 2)
}

func prevMonth(t time.Time, n time.Month) time.Month {
	y, m, _ := t.Date()
	_, m, _ = time.Date(y, m-n, 1, 0, 0, 0, 0, time.UTC).Date()
	return m
}

// getAlertsInLastMonth gets the alerts that have happened in the previous month (not the last 30 days)
// this automation is intended to be run on the 1st of the month
func (a *autoImpl) getAlertsInLastMonth(ctx context.Context, alertClient alertpb.AlertApiClient, name string, t time.Time) []*alertpb.Alert {
	var lastMonth []*alertpb.Alert

	listAlerts := alertpb.ListAlertsRequest{
		Name: name,
	}

	res, err := alertClient.ListAlerts(ctx, &listAlerts)

	for _, a := range res.Alerts {
		if isAlertInPreviousMonth(a.CreateTime.AsTime(), t) {
			lastMonth = append(lastMonth, a)
		}
	}

	// this is not great, if there are a lot of alerts then this could take a while
	// the alerts/notification system is going to be overhauled, so until that is done we can just do this
	for res.NextPageToken != "" {
		listAlerts = alertpb.ListAlertsRequest{
			Name:      name,
			PageToken: res.NextPageToken,
		}

		res, err = alertClient.ListAlerts(ctx, &listAlerts)
		if err != nil {
			a.Logger.Warn("failed to get alerts for "+name, zap.Error(err))
			continue
		}

		for _, a := range res.Alerts {
			if isAlertInPreviousMonth(a.CreateTime.AsTime(), t) {
				lastMonth = append(lastMonth, a)
			} else if isAlert2MonthsAgo(a.CreateTime.AsTime(), t) {
				// assuming that the alerts are given in descending chronological order we can stop looking here
				res.NextPageToken = ""
				break
			}
		}
	}
	return lastMonth
}

// createNotificationsFile creates a CSV file which lists notifications
//
//goland:noinspection GoUnhandledErrorResult
func (a *autoImpl) createNotificationsFile(alerts *[]*alertpb.Alert) []byte {
	buf := bytes.NewBuffer(nil)
	fmt.Fprintf(buf, "Notifications\n")

	// group by floors->zone so we have a map[map][Alert]
	byZone := make(map[string]map[string][]*alertpb.Alert)
	for _, a := range *alerts {
		if _, ok := byZone[a.Floor]; !ok {
			byZone[a.Floor] = make(map[string][]*alertpb.Alert)
		}

		byZone[a.Floor][a.Zone] = append(byZone[a.Floor][a.Zone], a)
	}

	floorKeys := make([]string, len(byZone))

	i := 0
	for k := range byZone {
		floorKeys[i] = k
		i++
	}

	sort.Strings(floorKeys)

	for _, floor := range floorKeys {

		zonesInFloor := byZone[floor]
		zoneKeys := make([]string, len(zonesInFloor))

		i := 0
		for k := range zonesInFloor {
			zoneKeys[i] = k
			i++
		}
		sort.Strings(zoneKeys)

		fmt.Fprintf(buf, "\n\nFloor: %s\n", floor)

		for _, zoneKey := range zoneKeys {
			fmt.Fprintf(buf, "Zone: %s\n", zoneKey)
			fmt.Fprintf(buf, "Create Time, Resolve Time, Source, Floor, Zone, Severity, Description, Acknowledged\n")
			for _, a := range zonesInFloor[zoneKey] {
				acked := "No"
				if a.Acknowledgement != nil {
					acked = a.Acknowledgement.String()
				}
				resolveTime := "N/A"
				if a.ResolveTime != nil {
					resolveTime = a.ResolveTime.AsTime().Format("2006-01-02 15:04:05")
				}

				fmt.Fprintf(buf, "%s,%s,%s,%s,%s,%s,%s,%s\n", a.CreateTime.AsTime().Format("2006-01-02 15:04:05"),
					resolveTime, a.Source, a.Floor,
					a.Zone, a.Severity.String(), a.Description, acked)
			}
		}
	}
	return buf.Bytes()
}

func (a *autoImpl) applyConfig(ctx context.Context, cfg config.Root) error {
	logger := a.Logger
	logger = logger.With(zap.String("snmp.addr", cfg.Destination.Addr()))

	alertClient := alertpb.NewAlertApiClient(a.Node.ClientConn())

	sendTime := cfg.Destination.SendTime
	now := cfg.Now
	if now == nil {
		now = a.Now
	}
	if now == nil {
		now = time.Now
	}
	go func() {
		t := now()
		for {
			next := sendTime.Next(t)
			select {
			case <-ctx.Done():
				return
			case <-time.After(time.Until(next)):
				// Use the time we were planning on running instead of the current time.
				// We do this to make output more predictable
				t = next
			}

			timeNow := now()
			timeout := cfg.Timeout

			if timeout == 0 {
				timeout = 10 * time.Second
			}

			lastMonth := a.getAlertsInLastMonth(ctx, alertClient, cfg.AlertHubName, timeNow)
			// generate the notifications CSV attachment file
			attachmentName := "notifications-" + timeNow.Format("2006-01-02") + ".csv"
			file := a.createNotificationsFile(&lastMonth)
			attachmentCfg := config.AttachmentCfg{
				AttachmentName: attachmentName,
				Attachment:     file,
			}

			err := retry(ctx, func(ctx context.Context) error {
				return sendEmail(cfg.Destination, attachmentCfg, cfg.Subject, cfg.TemplateArgs, logger)
			})
			if err != nil {
				logger.Warn("failed to send email", zap.Error(err))
			} else {
				logger.Info("email sent")
			}
		}
	}()

	return nil
}
func retry(ctx context.Context, f func(context.Context) error) error {
	return task.Run(ctx, func(ctx context.Context) (task.Next, error) {
		return 0, f(ctx)
	}, task.WithBackoff(10*time.Second, 10*time.Minute), task.WithRetry(40))
}
