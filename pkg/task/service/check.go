package service

import (
	"context"
	"errors"
)

// SystemCheck is the driver's persistent top-level health indicator, created by the framework
// and passed via driver.Services.SystemCheck. Drivers call MarkFailed when connectivity is
// lost and MarkRunning when it is restored. Drivers must call Dispose in their stop handler.
// May be nil if the check could not be registered; always nil-check before use.
type SystemCheck interface {
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
