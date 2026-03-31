package log

import (
	"archive/zip"
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

	// Wire download URL generation if an HTTP mux is available.
	if s.services.HTTPMux != nil && cfg.LogFilePath != "" {
		// Update the allowed directory on every config apply so the handler
		// always enforces the current log root, even after a reload.
		allowedDir := logAllowedDir(cfg.LogFilePath, cfg.LogDir)
		s.downloadAllowedDir.Store(&allowedDir)

		// Register the HTTP handler exactly once. Subsequent applyConfig calls
		// (e.g. on config reload via MonoApply) reuse the same registration so
		// that http.ServeMux does not panic on duplicate patterns. The HMAC key
		// is generated once in NewSystem so previously issued URLs remain valid.
		s.httpOnce.Do(func() {
			s.registeredDLPath = cfg.DownloadPath()
			s.services.HTTPMux.HandleFunc(s.registeredDLPath, func(w http.ResponseWriter, r *http.Request) {
				var dir string
				if p := s.downloadAllowedDir.Load(); p != nil {
					dir = *p
				}
				serveLogDownload(w, r, s.downloadKey, dir)
			})
		})

		ttl := time.Duration(cfg.URLTTLSecondsOrDefault()) * time.Second
		urlBase := cfg.HTTPDownloadURLBase
		if urlBase == "" && s.services.HTTPEndpoint != "" {
			urlBase = "https://" + s.services.HTTPEndpoint
		}
		key := s.downloadKey
		dlPath := s.registeredDLPath

		srv.GetDownloadLogUrlFunc = func(ctx context.Context, req *logpb.GetDownloadLogUrlRequest) (*logpb.GetDownloadLogUrlResponse, error) {
			return getDownloadLogURL(req, cfg.LogFilePath, cfg.LogDir, urlBase, dlPath, key, ttl)
		}
	}

	// Periodically refresh LogMetadata by walking the log files.
	if cfg.LogFilePath != "" {
		go func() {
			ticker := time.NewTicker(30 * time.Second)
			defer ticker.Stop()
			refreshMetadata(model, cfg.LogFilePath, cfg.LogDir, s.logger)
			for {
				select {
				case <-ctx.Done():
					return
				case <-ticker.C:
					refreshMetadata(model, cfg.LogFilePath, cfg.LogDir, s.logger)
				}
			}
		}()
	}

	// Announce the trait.
	announcer.Announce(s.name, node.HasTrait(logpb.TraitName, node.WithClients(logpb.WrapApi(srv))))

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

// --------------------------------------------------------------------------
// Log file metadata
// --------------------------------------------------------------------------

func refreshMetadata(model *logpb.Model, glob, logDir string, logger *zap.Logger) {
	var totalSize int64
	var fileCount int

	if logDir != "" {
		entries, err := os.ReadDir(logDir)
		if err != nil {
			logger.Warn("failed to read log dir", zap.String("dir", logDir), zap.Error(err))
			return
		}
		for _, e := range entries {
			if !e.Type().IsRegular() {
				continue
			}
			info, err := e.Info()
			if err != nil {
				continue
			}
			totalSize += info.Size()
			fileCount++
		}
	} else {
		files, err := filepath.Glob(glob)
		if err != nil {
			logger.Warn("failed to glob log files", zap.String("glob", glob), zap.Error(err))
			return
		}
		for _, f := range files {
			info, err := os.Stat(f)
			if err != nil {
				continue
			}
			totalSize += info.Size()
			fileCount++
		}
	}

	_, _ = model.UpdateLogMetadata(&logpb.LogMetadata{
		TotalSizeBytes: totalSize,
		FileCount:      int32(fileCount),
	})
}

// --------------------------------------------------------------------------
// Download URL generation and HTTP handler
// --------------------------------------------------------------------------

type downloadClaims struct {
	// FilePath is the absolute path to a single log file to download.
	FilePath string `json:"fp,omitempty"`
	// ZipFiles is a list of absolute paths to include in a zip archive download.
	ZipFiles []string `json:"zf,omitempty"`
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
	glob, logDir, urlBase, downloadPath string,
	key []byte,
	ttl time.Duration,
) (*logpb.GetDownloadLogUrlResponse, error) {
	if req.IncludeRotated {
		// Collect all files and return a single zip download URL.
		var files []string
		if logDir != "" {
			entries, err := os.ReadDir(logDir)
			if err != nil {
				return nil, status.Errorf(codes.Internal, "read log dir: %v", err)
			}
			for _, e := range entries {
				if !e.Type().IsRegular() {
					continue
				}
				files = append(files, filepath.Join(logDir, e.Name()))
			}
		} else {
			var err error
			files, err = filepath.Glob(glob)
			if err != nil {
				return nil, status.Errorf(codes.Internal, "glob log files: %v", err)
			}
		}
		if len(files) == 0 {
			return &logpb.GetDownloadLogUrlResponse{}, nil
		}
		tokenStr, err := signDownloadToken(downloadClaims{ZipFiles: files}, key, time.Now().Add(ttl))
		if err != nil {
			return nil, status.Errorf(codes.Internal, "sign token: %v", err)
		}
		u, err := buildDownloadURL(urlBase, downloadPath, tokenStr)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "build url: %v", err)
		}
		return &logpb.GetDownloadLogUrlResponse{Files: []*logpb.GetDownloadLogUrlResponse_LogFile{{
			Url:         u,
			Filename:    "logs.zip",
			ContentType: "application/zip",
		}}}, nil
	}

	// include_rotated=false: single most recent file.
	files, err := filepath.Glob(glob)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "glob log files: %v", err)
	}
	if len(files) == 0 {
		return &logpb.GetDownloadLogUrlResponse{}, nil
	}
	fp := files[len(files)-1]
	info, err := os.Stat(fp)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "stat log file: %v", err)
	}
	tokenStr, err := signDownloadToken(downloadClaims{FilePath: fp}, key, time.Now().Add(ttl))
	if err != nil {
		return nil, status.Errorf(codes.Internal, "sign token: %v", err)
	}
	u, err := buildDownloadURL(urlBase, downloadPath, tokenStr)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "build url: %v", err)
	}
	return &logpb.GetDownloadLogUrlResponse{Files: []*logpb.GetDownloadLogUrlResponse_LogFile{{
		Url:         u,
		Filename:    filepath.Base(fp),
		SizeBytes:   info.Size(),
		ContentType: detectContentType(fp),
	}}}, nil
}

func signDownloadToken(body downloadClaims, key []byte, expiry time.Time) (string, error) {
	signer, err := jose.NewSigner(
		jose.SigningKey{Algorithm: jose.HS256, Key: key},
		(&jose.SignerOptions{}).WithType("JWT"),
	)
	if err != nil {
		return "", err
	}
	claims := downloadJWTClaims{
		Claims: josejwt.Claims{Expiry: josejwt.NewNumericDate(expiry)},
		Body:   body,
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
// It validates the signed JWT in the "dlt" (download token) query parameter and
// streams the corresponding file directly via io.Copy — no read→serialise→write cycle.
//
// The "dlt" parameter is a base64url-encoded HS256-signed JWT containing either
// a single file path (downloadClaims.FilePath) or a list of paths to zip
// (downloadClaims.ZipFiles). Tokens are short-lived and signed with a per-config
// HMAC key; see signDownloadToken / parseDownloadToken.
//
// Concurrency note: active log file downloads use a sequential read handle;
// concurrent writes from the logger are safe because io.Copy reads sequentially
// and log rotation produces a new file (the active file is never truncated while
// the old one is being served).
func serveLogDownload(w http.ResponseWriter, r *http.Request, key []byte, allowedDir string) {
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("Cache-Control", "no-store")

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

	if len(claims.ZipFiles) > 0 {
		serveZipDownload(w, claims.ZipFiles, allowedDir)
		return
	}

	// Single file download.
	fp := filepath.Clean(claims.FilePath)
	if !isUnderDir(fp, allowedDir) {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
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

// serveZipDownload streams a zip archive containing the listed files.
// Files that do not reside under allowedDir are silently skipped.
func serveZipDownload(w http.ResponseWriter, files []string, allowedDir string) {
	w.Header().Set("Content-Type", "application/zip")
	w.Header().Set("Content-Disposition", `attachment; filename="logs.zip"`)

	zw := zip.NewWriter(w)
	defer zw.Close()

	for _, fp := range files {
		fp = filepath.Clean(fp)
		if !isUnderDir(fp, allowedDir) {
			continue
		}
		f, err := os.Open(fp)
		if err != nil {
			continue
		}
		info, err := f.Stat()
		if err != nil || info.IsDir() {
			f.Close()
			continue
		}
		fh, err := zip.FileInfoHeader(info)
		if err != nil {
			f.Close()
			continue
		}
		fh.Name = filepath.Base(fp)
		fh.Method = zip.Deflate
		fw, err := zw.CreateHeader(fh)
		if err != nil {
			f.Close()
			continue
		}
		_, _ = io.Copy(fw, f)
		f.Close()
	}
}

// logAllowedDir returns the directory under which log files are expected,
// derived from config. The result is used by isUnderDir to validate paths
// embedded in download tokens before opening them.
func logAllowedDir(logFilePath, logDir string) string {
	if logDir != "" {
		return filepath.Clean(logDir)
	}
	if logFilePath != "" {
		return filepath.Clean(filepath.Dir(logFilePath))
	}
	return ""
}

// isUnderDir reports whether fp resides directly under dir.
// An empty dir disables the check (returns true), so callers that have no
// configured log directory remain unaffected.
func isUnderDir(fp, dir string) bool {
	if dir == "" {
		return true
	}
	// Use dir+Sep as prefix to prevent "/var/log" from matching "/var/logmalicious".
	return strings.HasPrefix(fp, dir+string(filepath.Separator))
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
