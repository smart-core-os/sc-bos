package status

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/smart-core-os/gobacnet/enum/errorcode"
	bactypes "github.com/smart-core-os/gobacnet/types"
	"github.com/smart-core-os/sc-bos/pkg/driver/bacnet/comm"
	"github.com/smart-core-os/sc-bos/pkg/gentrait/statuspb"
	gen_statuspb "github.com/smart-core-os/sc-bos/pkg/proto/statuspb"
)

func UpdatePollErrorStatus(statuses *statuspb.Map, name, task string, requests []string, errs []error) {
	problemName := fmt.Sprintf("%s:%s", name, task)
	level, desc := SummariseRequestErrors(task, requests, errs)
	if level == gen_statuspb.StatusLog_LEVEL_UNDEFINED {
		return
	}

	statuses.UpdateProblem(name, &gen_statuspb.StatusLog_Problem{
		Name:        problemName,
		Level:       level,
		Description: desc,
	})
}

func SummariseRequestErrors(name string, requests []string, errs []error) (gen_statuspb.StatusLog_Level, string) {
	if len(errs) == 0 || len(requests) == 0 {
		return gen_statuspb.StatusLog_NOMINAL, fmt.Sprintf("%s nominal", name)
	}
	var readPropErrs []comm.ErrReadProperty
	var miscErrs []error
	for _, err := range errs {
		var readErr comm.ErrReadProperty
		if errors.As(err, &readErr) {
			readPropErrs = append(readPropErrs, readErr)
			continue
		}
		miscErrs = append(miscErrs, err)
	}

	cancelled, notFoundCount, timeoutCount, otherCount := countErrTypes(errs...)
	switch {
	case cancelled > 0:
		return gen_statuspb.StatusLog_LEVEL_UNDEFINED, fmt.Sprintf("%s request cancelled", name)
	case notFoundCount == len(requests):
		return gen_statuspb.StatusLog_NOTICE, fmt.Sprintf("%s points not found on device", name)
	case timeoutCount == len(requests):
		return gen_statuspb.StatusLog_REDUCED_FUNCTION, fmt.Sprintf("%s points timed out", name)
	default:
		failedPropNames := failedPropReads(errs...)
		var desc strings.Builder
		desc.WriteString(name)
		switch len(errs) {
		case 1:
			desc.WriteString(" point")
		default:
			desc.WriteString(" points")
		}
		if len(failedPropNames) > 0 {
			skipped := len(failedPropNames) - 5
			if skipped > 0 {
				failedPropNames = failedPropNames[:5]
			}
			fmt.Fprintf(&desc, " %s", strings.Join(failedPropNames, ", "))
			if skipped > 0 {
				fmt.Fprintf(&desc, " and %d more", skipped)
			}
		}
		level := gen_statuspb.StatusLog_REDUCED_FUNCTION
		switch {
		case notFoundCount > 0 && timeoutCount == 0 && otherCount == 0:
			fmt.Fprintf(&desc, " not found on device")
			level = gen_statuspb.StatusLog_NOTICE
		case notFoundCount == 0 && timeoutCount > 0 && otherCount == 0:
			fmt.Fprintf(&desc, " timed out")
		case notFoundCount == 0 && timeoutCount == 0 && otherCount > 0:
			fmt.Fprintf(&desc, " errored")
		default:
			desc.WriteString(" read,")
			if notFoundCount > 0 {
				fmt.Fprintf(&desc, " %d not found", notFoundCount)
			}
			if timeoutCount > 0 {
				fmt.Fprintf(&desc, " %d timed out", timeoutCount)
			}
			if otherCount > 0 {
				fmt.Fprintf(&desc, " %d error", otherCount)
			}
		}
		return level, desc.String()
	}
}

func countErrTypes(errs ...error) (cancelled, notFound, timeout, other int) {
	countBacErr := func(bacErr bactypes.Error) {
		switch bacErr.Code {
		case errorcode.UnknownObject, errorcode.UnknownProperty:
			notFound++
		case errorcode.Timeout:
			timeout++
		default:
			other++
		}
	}

	for _, err := range errs {
		var propErr bactypes.PropertyAccessError
		var bacErr bactypes.Error
		switch {
		case errors.Is(err, context.Canceled):
			cancelled++
		case errors.Is(err, comm.ErrPropNotFound):
			notFound++
		case errors.Is(err, context.DeadlineExceeded):
			timeout++
		case errors.As(err, &propErr):
			countBacErr(propErr.Err)
		case errors.As(err, &bacErr):
			countBacErr(bacErr)
		default:
			other++
		}
	}
	return
}

func failedPropReads(errs ...error) []string {
	var failed []string
	for _, err := range errs {
		var readErr comm.ErrReadProperty
		if errors.As(err, &readErr) {
			failed = append(failed, readErr.PropName())
		}
	}
	return failed
}
