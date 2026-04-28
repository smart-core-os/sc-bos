package service

// SystemCheck is the driver's persistent top-level health indicator, created by the framework
// and passed via driver.Services.SystemCheck. Drivers call MarkFailed when connectivity is
// lost and MarkRunning when it is restored. Drivers must call Dispose in their stop handler.
// May be nil if the check could not be registered; always nil-check before use.
type SystemCheck interface {
	Dispose()
	MarkRunning()
	MarkFailed(err error)
}
