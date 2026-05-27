package log

import (
	"context"
	"fmt"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/smart-core-os/sc-bos/internal/logdownload"
	"github.com/smart-core-os/sc-bos/pkg/node"
	"github.com/smart-core-os/sc-bos/pkg/proto/logpb"
	"github.com/smart-core-os/sc-bos/pkg/system/log/config"
)

func (s *System) applyConfig(ctx context.Context, cfg config.Root) error {
	announcer := s.announcer.Replace(ctx)

	model := logpb.NewModel(cfg.BufCapOrDefault())

	// Install log capture: hook every zap entry into the model's ring buffer.
	if s.services.AddLogCore != nil {
		captureCore := &captureCore{model: model}
		removeCapture := s.services.AddLogCore(captureCore)
		context.AfterFunc(ctx, removeCapture)
	}

	// Sync log level: if the controller exposes an AtomicLevel, keep the
	// model's level in sync with it and vice-versa.
	if s.services.LogLevel != nil {
		al := s.services.LogLevel
		// seed the model with the current level
		_ = setModelLevel(model, al.Level())

		// when UpdateLogLevel is called, also change the real zap level
		model.OnUpdateLogLevel = func(lvl *logpb.LogLevel) {
			al.SetLevel(logpbLevelToZap(lvl.Level))
			s.logger.Info("log level changed", zap.String("level", lvl.Level.String()))
		}
	}

	// Create the gRPC server backed by this model.
	srv := logpb.NewModelServer(model)

	// Wire download URL generation against the shared signed-URL router. The
	// handler is (re-)registered each apply with a closure capturing the
	// current config's allowed dir; Router.Handle replaces the previous
	// handler in place.
	if s.services.DownloadRouter != nil && cfg.LogFilePath != "" {
		allowedDir := logdownload.LogAllowedDir(cfg.LogFilePath, cfg.LogDir)
		glob, logDir := cfg.LogFilePath, cfg.LogDir
		s.services.DownloadRouter.Handle(DownloadType, &logdownload.FileDownloadHandler{
			AllowedDir: func() string { return allowedDir },
		})
		srv.GetDownloadLogUrlFunc = func(_ context.Context, req *logpb.GetDownloadLogUrlRequest) (*logpb.GetDownloadLogUrlResponse, error) {
			return logdownload.GenerateFileDownloadURL(req, glob, logDir, s.services.DownloadRouter, DownloadType)
		}
	}

	// Periodically refresh LogMetadata by walking the log files.
	if cfg.LogFilePath != "" {
		go func() {
			ticker := time.NewTicker(30 * time.Second)
			defer ticker.Stop()
			logdownload.RefreshMetadata(model, cfg.LogFilePath, cfg.LogDir, s.logger)
			for {
				select {
				case <-ctx.Done():
					return
				case <-ticker.C:
					logdownload.RefreshMetadata(model, cfg.LogFilePath, cfg.LogDir, s.logger)
				}
			}
		}()
	}

	// Announce the trait.
	announcer.Announce(s.name,
		node.HasServer(logpb.RegisterLogApiServer, logpb.LogApiServer(srv)),
		node.HasTrait(logpb.TraitName),
	)

	return nil
}

// --------------------------------------------------------------------------
// zapcore.Core implementation for capturing log entries into the model.
// --------------------------------------------------------------------------

// captureCore is a zapcore.Core that captures every log entry into the model.
// withFields accumulates fields added via With() so they appear on every Write.
type captureCore struct {
	model      *logpb.Model
	withFields []zapcore.Field
}

func (c *captureCore) Enabled(_ zapcore.Level) bool { return true }

func (c *captureCore) With(fields []zapcore.Field) zapcore.Core {
	merged := make([]zapcore.Field, len(c.withFields)+len(fields))
	copy(merged, c.withFields)
	copy(merged[len(c.withFields):], fields)
	return &captureCore{model: c.model, withFields: merged}
}

func (c *captureCore) Check(entry zapcore.Entry, ce *zapcore.CheckedEntry) *zapcore.CheckedEntry {
	return ce.AddCore(entry, c)
}

func (c *captureCore) Write(entry zapcore.Entry, fields []zapcore.Field) error {
	all := append(c.withFields, fields...)
	c.model.AppendMessage(entryToLogMessage(entry, all))
	return nil
}

func (c *captureCore) Sync() error { return nil }

func entryToLogMessage(entry zapcore.Entry, fields []zapcore.Field) *logpb.LogMessage {
	msg := &logpb.LogMessage{
		Timestamp: timestamppb.New(entry.Time),
		Level:     zapLevelToLogpb(entry.Level),
		Logger:    entry.LoggerName,
		Message:   entry.Message,
	}
	if entry.Caller.Defined {
		msg.SourceLocation = &logpb.SourceLocation{
			File:     entry.Caller.File,
			Line:     int32(entry.Caller.Line),
			Function: entry.Caller.Function,
		}
	}
	if entry.Stack != "" {
		msg.StackTrace = entry.Stack
	}
	if len(fields) > 0 {
		msg.Fields = make(map[string]string, len(fields))
		enc := zapcore.NewMapObjectEncoder()
		for _, f := range fields {
			f.AddTo(enc)
		}
		for k, v := range enc.Fields {
			msg.Fields[k] = fmt.Sprintf("%v", v)
		}
	}
	return msg
}

func zapLevelToLogpb(l zapcore.Level) logpb.Level {
	switch {
	case l <= zapcore.DebugLevel:
		return logpb.Level_LEVEL_DEBUG
	case l <= zapcore.InfoLevel:
		return logpb.Level_LEVEL_INFO
	case l <= zapcore.WarnLevel:
		return logpb.Level_LEVEL_WARN
	default: // ErrorLevel, DPanicLevel, PanicLevel, FatalLevel → ERROR
		return logpb.Level_LEVEL_ERROR
	}
}

func logpbLevelToZap(l logpb.Level) zapcore.Level {
	switch l {
	case logpb.Level_LEVEL_DEBUG:
		return zapcore.DebugLevel
	case logpb.Level_LEVEL_WARN:
		return zapcore.WarnLevel
	case logpb.Level_LEVEL_ERROR:
		return zapcore.ErrorLevel
	default:
		return zapcore.InfoLevel
	}
}

func setModelLevel(m *logpb.Model, l zapcore.Level) error {
	_, err := m.UpdateLogLevel(&logpb.LogLevel{Level: zapLevelToLogpb(l)})
	return err
}

