package bms

import (
	"strings"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"go.uber.org/zap"
	"google.golang.org/protobuf/testing/protocmp"

	"github.com/smart-core-os/sc-bos/pkg/proto/airtemperaturepb"
	"github.com/smart-core-os/sc-bos/pkg/proto/modepb"
)

func logWrites(logger *zap.Logger, rs *ReadState, ws *WriteState, counts *ActionCounts, ttl time.Duration, err error) {
	switch {
	case err != nil:
		logger.Warn("processReadState failed; scheduling retry",
			zap.Error(err),
			zap.Stringer("time", formattedDuration(ws.T1.Sub(ws.T0))),
			zap.Stringer("retryAfter", formattedDuration(ttl)),
		)
	case counts.TotalWrites > 0 || rs.Config.LogDuplicateChanges:
		// something changed, work out what
		logger.Debug("processReadState complete",
			zap.Strings("changes", counts.Changes()),
			zap.Strings("reasons", ws.Reasons),
			zap.Stringer("time", formattedDuration(ws.T1.Sub(ws.T0))),
			zap.Stringer("ttl", formattedDuration(ttl)))
	case ttl > 0 && rs.Config.LogTTLDelays:
		logger.Debug("processReadState made no changes",
			zap.Strings("reasons", ws.Reasons),
			zap.Stringer("time", formattedDuration(ws.T1.Sub(ws.T0))),
			zap.Stringer("ttl", formattedDuration(ttl)))
	}
}

func formatDuration(d time.Duration) string {
	d2 := d
	switch {
	case d > time.Hour:
		d2 = d.Round(time.Hour)
	case d > time.Minute:
		d2 = d.Round(time.Minute)
	case d > time.Second:
		d2 = d.Round(time.Second)
	case d > time.Millisecond:
		d2 = d.Round(time.Millisecond)
	case d > time.Microsecond:
		d2 = d.Round(time.Microsecond)
	}

	s := d2.String()
	if d2 != d {
		s = "~" + s
	}
	if strings.HasSuffix(s, "m0s") {
		s = s[:len(s)-2]
	}
	if strings.HasSuffix(s, "h0m") {
		s = s[:len(s)-2]
	}
	return s
}

type formattedDuration time.Duration

func (f formattedDuration) String() string {
	return formatDuration(time.Duration(f))
}

func logReads(logger *zap.Logger, oldState, newState *ReadState) {
	if oldState == nil {
		logger.Debug("processReadState first run")
		return
	}
	if diff := cmp.Diff(oldState, newState,
		cmpopts.IgnoreFields(ReadState{}, "Now", "Config", "StartTime"),
		cmpopts.IgnoreFields(Value[*airtemperaturepb.AirTemperature]{}, "At", "Hit"),
		cmpopts.IgnoreFields(Value[string]{}, "At", "Hit"),
		cmpopts.IgnoreFields(Value[*modepb.ModeValues]{}, "At", "Hit"),
		protocmp.Transform()); diff != "" {
		logger.Debug("processReadState read state changed", zap.String("diff", diff))
		return
	}
}
