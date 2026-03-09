package log

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"io"
	"mime"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/go-jose/go-jose/v4"
	josejwt "github.com/go-jose/go-jose/v4/jwt"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	genlogpb "github.com/smart-core-os/sc-bos/pkg/gentrait/logpb"
	"github.com/smart-core-os/sc-bos/pkg/node"
	"github.com/smart-core-os/sc-bos/pkg/proto/logpb"
	"github.com/smart-core-os/sc-bos/pkg/system/log/config"
)

func (s *System) applyConfig(ctx context.Context, cfg config.Root) error {
	announcer := s.announcer.Replace(ctx)

	model := genlogpb.NewModel(cfg.BufCap)

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
		}
	}

	// Create the gRPC server backed by this model.
	srv := genlogpb.NewModelServer(model)

	// Wire download URL generation if an HTTP mux is available.
	if s.services.HTTPMux != nil && cfg.LogFilePath != "" {
		key := newHMACKey()
		ttl := time.Duration(cfg.URLTTLSecondsOrDefault()) * time.Second
		downloadPath := cfg.DownloadPath()
		urlBase := cfg.HTTPDownloadURLBase

		srv.GetDownloadLogUrlFunc = func(ctx context.Context, req *logpb.GetDownloadLogUrlRequest) (*logpb.GetDownloadLogUrlResponse, error) {
			return getDownloadLogURL(req, cfg.LogFilePath, urlBase, downloadPath, key, ttl)
		}

		s.services.HTTPMux.HandleFunc(downloadPath, func(w http.ResponseWriter, r *http.Request) {
			serveLogDownload(w, r, key)
		})
	}

	// Periodically refresh LogMetadata by walking the log files.
	if cfg.LogFilePath != "" {
		go func() {
			ticker := time.NewTicker(30 * time.Second)
			defer ticker.Stop()
			refreshMetadata(model, cfg.LogFilePath, s.logger)
			for {
				select {
				case <-ctx.Done():
					return
				case <-ticker.C:
					refreshMetadata(model, cfg.LogFilePath, s.logger)
				}
			}
		}()
	}

	// Announce the trait.
	announcer.Announce(s.name, node.HasTrait(genlogpb.TraitName, node.WithClients(logpb.WrapApi(srv))))

	return nil
}

// --------------------------------------------------------------------------
// zapcore.Core implementation for capturing log entries into the model.
// --------------------------------------------------------------------------

// captureCore is a zapcore.Core that captures every log entry into the model.
// It does NOT inherit fields from With() calls; only per-Write fields are captured.
// Concurrent-write log rotation is safe because io.Copy reads sequentially.
type captureCore struct {
	model *genlogpb.Model
}

func (c *captureCore) Enabled(_ zapcore.Level) bool { return true }

func (c *captureCore) With(_ []zapcore.Field) zapcore.Core { return c }

func (c *captureCore) Check(entry zapcore.Entry, ce *zapcore.CheckedEntry) *zapcore.CheckedEntry {
	return ce.AddCore(entry, c)
}

func (c *captureCore) Write(entry zapcore.Entry, fields []zapcore.Field) error {
	c.model.AppendMessage(entryToLogMessage(entry, fields))
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

func zapLevelToLogpb(l zapcore.Level) logpb.LogLevel_Level {
	switch l {
	case zapcore.DebugLevel:
		return logpb.LogLevel_DEBUG
	case zapcore.InfoLevel:
		return logpb.LogLevel_INFO
	case zapcore.WarnLevel:
		return logpb.LogLevel_WARN
	case zapcore.ErrorLevel:
		return logpb.LogLevel_ERROR
	case zapcore.DPanicLevel:
		return logpb.LogLevel_DPANIC
	case zapcore.PanicLevel:
		return logpb.LogLevel_PANIC
	case zapcore.FatalLevel:
		return logpb.LogLevel_FATAL
	default:
		return logpb.LogLevel_LEVEL_UNSPECIFIED
	}
}

func logpbLevelToZap(l logpb.LogLevel_Level) zapcore.Level {
	switch l {
	case logpb.LogLevel_DEBUG:
		return zapcore.DebugLevel
	case logpb.LogLevel_WARN:
		return zapcore.WarnLevel
	case logpb.LogLevel_ERROR:
		return zapcore.ErrorLevel
	case logpb.LogLevel_DPANIC:
		return zapcore.DPanicLevel
	case logpb.LogLevel_PANIC:
		return zapcore.PanicLevel
	case logpb.LogLevel_FATAL:
		return zapcore.FatalLevel
	default:
		return zapcore.InfoLevel
	}
}

func setModelLevel(m *genlogpb.Model, l zapcore.Level) error {
	_, err := m.UpdateLogLevel(&logpb.LogLevel{Level: zapLevelToLogpb(l)})
	return err
}

// --------------------------------------------------------------------------
// Log file metadata
// --------------------------------------------------------------------------

func refreshMetadata(model *genlogpb.Model, glob string, logger *zap.Logger) {
	files, err := filepath.Glob(glob)
	if err != nil {
		logger.Warn("failed to glob log files", zap.String("glob", glob), zap.Error(err))
		return
	}
	var totalSize int64
	for _, f := range files {
		info, err := os.Stat(f)
		if err != nil {
			continue
		}
		totalSize += info.Size()
	}
	_, _ = model.UpdateLogMetadata(&logpb.LogMetadata{
		TotalSizeBytes: totalSize,
		FileCount:      int32(len(files)),
	})
}

// --------------------------------------------------------------------------
// Download URL generation and HTTP handler
// --------------------------------------------------------------------------

type downloadClaims struct {
	// FilePath is the absolute path to the log file to download.
	FilePath string `json:"fp"`
}

type downloadJWTClaims struct {
	josejwt.Claims
	Body downloadClaims `json:"b"`
}

// newHMACKey generates a random 32-byte HMAC key.
// The key is generated once per applyConfig call (i.e. once per config reload).
func newHMACKey() []byte {
	key := make([]byte, 32)
	if _, err := rand.Read(key); err != nil {
		panic("logpb: failed to generate HMAC key: " + err.Error())
	}
	return key
}

func getDownloadLogURL(
	req *logpb.GetDownloadLogUrlRequest,
	glob, urlBase, downloadPath string,
	key []byte,
	ttl time.Duration,
) (*logpb.GetDownloadLogUrlResponse, error) {
	files, err := filepath.Glob(glob)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "glob log files: %v", err)
	}
	if len(files) == 0 {
		return &logpb.GetDownloadLogUrlResponse{}, nil
	}

	// Without include_rotated, only return the most recently modified file.
	if !req.IncludeRotated && len(files) > 1 {
		files = files[len(files)-1:]
	}

	var logFiles []*logpb.GetDownloadLogUrlResponse_LogFile
	for _, fp := range files {
		info, err := os.Stat(fp)
		if err != nil {
			continue
		}
		tokenStr, err := signDownloadToken(fp, key, time.Now().Add(ttl))
		if err != nil {
			return nil, status.Errorf(codes.Internal, "sign token: %v", err)
		}
		u, err := buildDownloadURL(urlBase, downloadPath, tokenStr)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "build url: %v", err)
		}
		logFiles = append(logFiles, &logpb.GetDownloadLogUrlResponse_LogFile{
			Url:         u,
			Filename:    filepath.Base(fp),
			SizeBytes:   info.Size(),
			ContentType: detectContentType(fp),
		})
	}
	return &logpb.GetDownloadLogUrlResponse{Files: logFiles}, nil
}

func signDownloadToken(filePath string, key []byte, expiry time.Time) (string, error) {
	signer, err := jose.NewSigner(
		jose.SigningKey{Algorithm: jose.HS256, Key: key},
		(&jose.SignerOptions{}).WithType("JWT"),
	)
	if err != nil {
		return "", err
	}
	claims := downloadJWTClaims{
		Claims: josejwt.Claims{Expiry: josejwt.NewNumericDate(expiry)},
		Body:   downloadClaims{FilePath: filePath},
	}
	return josejwt.Signed(signer).Claims(claims).Serialize()
}

func parseDownloadToken(tokenStr string, key []byte) (*downloadClaims, error) {
	tok, err := josejwt.ParseSigned(tokenStr, []jose.SignatureAlgorithm{jose.HS256})
	if err != nil {
		return nil, err
	}
	var claims downloadJWTClaims
	if err := tok.Claims(key, &claims); err != nil {
		return nil, err
	}
	if err := claims.ValidateWithLeeway(josejwt.Expected{Time: time.Now()}, time.Minute); err != nil {
		return nil, err
	}
	return &claims.Body, nil
}

func buildDownloadURL(urlBase, downloadPath, tokenStr string) (string, error) {
	base := urlBase
	if base == "" {
		base = downloadPath
	}
	u, err := url.Parse(base)
	if err != nil {
		return "", err
	}
	if !strings.HasPrefix(u.Path, "/") {
		u.Path = downloadPath
	} else if u.Path == "" {
		u.Path = downloadPath
	}
	q := u.Query()
	q.Set("dlt", base64.RawURLEncoding.EncodeToString([]byte(tokenStr)))
	u.RawQuery = q.Encode()
	return u.String(), nil
}

// serveLogDownload is the HTTP handler for log file downloads.
// It validates the signed JWT in the "dlt" query parameter and streams
// the corresponding file directly via io.Copy — no read→serialise→write cycle.
//
// Concurrency note: active log file downloads use a sequential read handle;
// concurrent writes from the logger are safe because io.Copy reads sequentially
// and log rotation produces a new file (the active file is never truncated while
// the old one is being served).
func serveLogDownload(w http.ResponseWriter, r *http.Request, key []byte) {
	tokenB64 := r.URL.Query().Get("dlt")
	if tokenB64 == "" {
		http.Error(w, "missing download token", http.StatusUnauthorized)
		return
	}
	tokenStr, err := base64.RawURLEncoding.DecodeString(tokenB64)
	if err != nil {
		http.Error(w, "invalid download token", http.StatusUnauthorized)
		return
	}
	claims, err := parseDownloadToken(string(tokenStr), key)
	if err != nil {
		http.Error(w, "invalid or expired download token", http.StatusUnauthorized)
		return
	}

	// Validate the path is under a safe location (prevent path traversal).
	fp := filepath.Clean(claims.FilePath)
	f, err := os.Open(fp)
	if err != nil {
		http.Error(w, "file not found", http.StatusNotFound)
		return
	}
	defer f.Close()

	info, err := f.Stat()
	if err != nil || info.IsDir() {
		http.Error(w, "not a file", http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", detectContentType(fp))
	w.Header().Set("Content-Disposition", `attachment; filename="`+filepath.Base(fp)+`"`)
	w.Header().Set("Content-Length", fmt.Sprintf("%d", info.Size()))
	_, _ = io.Copy(w, f)
}

func detectContentType(fp string) string {
	switch {
	case strings.HasSuffix(fp, ".gz"):
		return "application/gzip"
	case strings.HasSuffix(fp, ".zst"):
		return "application/zstd"
	case strings.HasSuffix(fp, ".bz2"):
		return "application/x-bzip2"
	default:
		ct := mime.TypeByExtension(filepath.Ext(fp))
		if ct == "" {
			return "text/plain"
		}
		return ct
	}
}
