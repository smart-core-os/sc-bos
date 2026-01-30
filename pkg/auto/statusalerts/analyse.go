package statusalerts

import (
	"context"
	"time"

	"go.uber.org/zap"

	"github.com/smart-core-os/sc-bos/pkg/auto/statusalerts/config"
	"github.com/smart-core-os/sc-bos/pkg/proto/alertpb"
	"github.com/smart-core-os/sc-bos/pkg/proto/statuspb"
)

func analyseStatusLogs(ctx context.Context, source config.Source, c <-chan *statuspb.StatusLog, name string, client alertpb.AlertAdminApiClient, logger *zap.Logger) error {
	var failedLog *statuspb.StatusLog
	var failedCount int

	retryTimer := newStoppedTimer()
	nextAttemptDelay := 200 * time.Millisecond
	var firstAttemptTime time.Time
	const nextAttemptScale = 1.2
	const maxAttemptDelay = 10 * time.Second

	debounceTimer := newStoppedTimer()
	debounceDelay := source.DebounceOrDefault()
	var debouncedLog *statuspb.StatusLog

	recordResult := func(msg *statuspb.StatusLog, err error) {
		switch {
		case err == nil && failedLog == nil: // last attempt worked, this attempt worked too
		case err == nil && failedLog != nil:
			if failedCount > 5 {
				logger.Debug("alert saved successfully after previous attempt", zap.Int("attempts", failedCount))
			}

			failedLog = nil
			failedCount = 0
			if !retryTimer.Stop() {
				<-retryTimer.C
			}
			nextAttemptDelay = 200 * time.Millisecond
		case err != nil:
			if failedLog == nil {
				firstAttemptTime = time.Now()
			}
			if !retryTimer.Stop() && failedLog != nil {
				<-retryTimer.C
			}
			retryTimer.Reset(nextAttemptDelay)

			failedLog = msg
			failedCount++
			// setup the next attempt to send the msg
			nextAttemptDelay = time.Duration(float64(nextAttemptDelay) * nextAttemptScale)
			if nextAttemptDelay > maxAttemptDelay {
				nextAttemptDelay = maxAttemptDelay
			}

			switch {
			case failedCount == 5:
				logger.Warn("failed to save alert, will retry", zap.Int("attempts", failedCount), zap.Error(err))
			case failedCount == 20:
				logger.Warn("failed to save alert, reducing logging", zap.Int("attempts", failedCount), zap.Error(err))
			case failedCount%100 == 0:
				logger.Debug("failed to save alert, will retry", zap.Int("attempts", failedCount), zap.Time("since", firstAttemptTime))
			}
		}
	}

	for {
		var msg *statuspb.StatusLog
		select {
		case <-retryTimer.C:
			msg = failedLog
			failedLog = nil
		case <-debounceTimer.C:
			msg = debouncedLog
			debouncedLog = nil
		case m, ok := <-c:
			if !ok {
				return ctx.Err()
			}
			if debounceDelay > 0 {
				// Checking for level changes means that devices that send a constant stream of description changes
				// don't cause an infinite debounce where we never actually store the alert
				if debouncedLog == nil || m.Level != debouncedLog.Level {
					if !debounceTimer.Stop() && debouncedLog != nil {
						<-debounceTimer.C
					}
					debounceTimer.Reset(debounceDelay)
				}
				debouncedLog = m
				continue
			}
			msg = m
		}

		switch {
		case msg.Level == statuspb.StatusLog_NOMINAL:
			_, err := client.ResolveAlert(ctx, &alertpb.ResolveAlertRequest{
				Name:         name,
				Alert:        &alertpb.Alert{Source: source.Name},
				AllowMissing: true,
			})
			recordResult(msg, err)
		default:
			_, err := client.CreateAlert(ctx, &alertpb.CreateAlertRequest{
				Name: name,
				Alert: &alertpb.Alert{
					Description: logToDescription(msg),
					Severity:    levelToSeverity(msg.Level),
					Floor:       source.Floor,
					Zone:        source.Zone,
					Subsystem:   source.Subsystem,
					Source:      source.Name,
				},
				MergeSource: true,
			})
			recordResult(msg, err)
		}
	}
}

func newStoppedTimer() *time.Timer {
	t := time.NewTimer(0)
	if !t.Stop() {
		<-t.C
	}
	return t
}

func levelToSeverity(level statuspb.StatusLog_Level) alertpb.Alert_Severity {
	switch level {
	case statuspb.StatusLog_NOMINAL:
		return alertpb.Alert_SEVERITY_UNSPECIFIED
	case statuspb.StatusLog_NOTICE:
		return alertpb.Alert_INFO
	case statuspb.StatusLog_REDUCED_FUNCTION:
		return alertpb.Alert_WARNING
	case statuspb.StatusLog_NON_FUNCTIONAL, statuspb.StatusLog_OFFLINE:
		return alertpb.Alert_SEVERE
	default:
		return alertpb.Alert_WARNING
	}
}

func logToDescription(log *statuspb.StatusLog) string {
	return log.Description
}
