package http

import "go.uber.org/zap"

// logWrapper for intercepting go-retryablehttp Printf logs
type logWrapper struct {
	*zap.SugaredLogger
}

func (lw *logWrapper) Printf(msg string, args ...any) {
	lw.Debugf(msg, args...)
}
