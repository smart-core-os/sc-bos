// Package audit provides the audit log subsystem used by Bootstrap.
// It writes security-relevant write operations to a rotating file and an
// in-memory ring buffer, both accessible via the LogApi gRPC trait.
package audit

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/natefinch/lumberjack.v2"

	"github.com/smart-core-os/sc-bos/internal/logdownload"
	"github.com/smart-core-os/sc-bos/pkg/proto/logpb"
)

// Setup holds the initialised audit log subsystem. Create with NewSetup;
// always call Close on shutdown (after calling Logger.Sync if non-nil).
type Setup struct {
	// Logger writes structured JSON audit entries to a rotating file.
	// Nil when no filename was configured or the file could not be opened.
	Logger *zap.Logger
	// Model holds the in-memory ring buffer of audit entries. Always non-nil.
	Model *logpb.Model

	downloadKey []byte
	filename    string
	closer      io.Closer
}

// NewSetup creates the audit log subsystem.
// When filename is non-empty a rotating JSON file logger is also created.
// A non-nil error means the file could not be opened — the returned *Setup
// is still valid for in-memory use. The caller must call Close.
func NewSetup(filename string, maxSizeMB, maxAgeDays, maxBackups int, compress bool) (*Setup, error) {
	a := &Setup{Model: logpb.NewModel(0)}
	if filename == "" {
		return a, nil
	}
	maxSize := maxSizeMB
	if maxSize == 0 {
		maxSize = 100
	}
	logger, closer, err := newFileLogger(filename, maxSize, maxAgeDays, maxBackups, compress)
	if err != nil {
		return a, err
	}
	a.Logger = logger
	a.closer = closer
	a.downloadKey = logdownload.NewHMACKey()
	a.filename = filename
	return a, nil
}

// Write records msg to both the file logger (if configured) and the in-memory
// model, guaranteeing both sinks always contain identical data.
// Implements policy.AuditSink.
func (a *Setup) Write(msg *logpb.LogMessage) {
	if a.Logger != nil {
		fields := make([]zap.Field, 0, len(msg.Fields))
		for k, v := range msg.Fields {
			fields = append(fields, zap.String(k, v))
		}
		switch msg.Level {
		case logpb.Level_LEVEL_WARN, logpb.Level_LEVEL_ERROR:
			a.Logger.Warn(msg.Message, fields...)
		default:
			a.Logger.Info(msg.Message, fields...)
		}
	}
	a.Model.AppendMessage(msg)
}

// RegisterHTTP installs the audit-log file download handler on mux at dlPath.
// Does nothing when no filename was configured.
func (a *Setup) RegisterHTTP(mux *http.ServeMux, dlPath string) {
	if a.filename == "" {
		return
	}
	allowedDir := filepath.Dir(a.filename)
	key := a.downloadKey
	mux.HandleFunc(dlPath, func(w http.ResponseWriter, r *http.Request) {
		logdownload.ServeLogDownload(w, r, key, allowedDir)
	})
}

// NewModelServer returns a LogApi ModelServer backed by this Setup's model.
// When dlPath is non-empty and a file was configured, GetDownloadLogUrl is
// wired up with a 15-minute token TTL.
func (a *Setup) NewModelServer(urlBase, dlPath string) *logpb.ModelServer {
	srv := logpb.NewModelServer(a.Model)
	if a.filename != "" && dlPath != "" {
		ttl := 15 * time.Minute
		key, filename := a.downloadKey, a.filename
		srv.GetDownloadLogUrlFunc = func(ctx context.Context, req *logpb.GetDownloadLogUrlRequest) (*logpb.GetDownloadLogUrlResponse, error) {
			return logdownload.GetDownloadLogURL(req, filename, "", urlBase, dlPath, key, ttl)
		}
	}
	return srv
}

// StartMetadataRefresh launches a background goroutine that scans the audit
// log files every 30 seconds and updates the model's LogMetadata.
// Does nothing when no filename was configured.
func (a *Setup) StartMetadataRefresh(ctx context.Context, logger *zap.Logger) {
	if a.filename == "" {
		return
	}
	filename := a.filename
	go func() {
		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()
		logdownload.RefreshMetadata(a.Model, filename, "", logger)
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				logdownload.RefreshMetadata(a.Model, filename, "", logger)
			}
		}
	}()
}

// Close releases file resources held by the Setup.
// The caller should call Logger.Sync() before Close.
func (a *Setup) Close() error {
	if a.closer != nil {
		return a.closer.Close()
	}
	return nil
}

// newFileLogger creates a rotating JSON file logger and returns a zap.Logger
// that writes to it, plus an io.Closer for cleanup.
func newFileLogger(filename string, maxSizeMB, maxAgeDays, maxBackups int, compress bool) (*zap.Logger, io.Closer, error) {
	if err := os.MkdirAll(filepath.Dir(filename), 0o755); err != nil {
		return nil, nil, fmt.Errorf("create audit log dir: %w", err)
	}
	// Open eagerly to surface config errors at startup rather than on first write.
	f, err := os.OpenFile(filename, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o640)
	if err != nil {
		return nil, nil, fmt.Errorf("open audit log: %w", err)
	}
	f.Close()
	w := &lumberjack.Logger{
		Filename:   filename,
		MaxSize:    maxSizeMB,
		MaxAge:     maxAgeDays,
		MaxBackups: maxBackups,
		Compress:   compress,
	}
	enc := zap.NewProductionEncoderConfig()
	enc.EncodeTime = zapcore.ISO8601TimeEncoder
	core := zapcore.NewCore(
		zapcore.NewJSONEncoder(enc),
		zapcore.AddSync(w),
		zapcore.InfoLevel,
	)
	return zap.New(core), w, nil
}
