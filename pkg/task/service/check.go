package service

import (
	"context"
	"errors"
)

// SystemCheck is a service's top-level health indicator, created by the framework
// and passed via the SystemCheck field of driver, auto and zone Services, see ManagedSystemCheck.
// Services call MarkFailed when they can't do their job, for example when connectivity is
// lost, and MarkRunning when they can again.
// May be nil if the check could not be registered; always nil-check before use.
type SystemCheck interface {
	// Dispose removes the check.
	// The checks the framework passes to services are disposed for them when the service stops.
	Dispose()
	MarkRunning()
	MarkFailed(err error)
}

// UpdateSystemCheck calls MarkFailed(err) or MarkRunning() on check based on err.
// Safe when check is nil. Ignores context.Canceled and context.DeadlineExceeded — these
// indicate intentional shutdown, not a connectivity failure.
func UpdateSystemCheck(check SystemCheck, err error) {
	if check == nil {
		return
	}
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return
	}
	if err != nil {
		check.MarkFailed(err)
	} else {
		check.MarkRunning()
	}
}
