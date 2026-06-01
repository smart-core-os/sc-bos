package logdownload

import (
	"net/http"
	"os"
	"path/filepath"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"

	"github.com/smart-core-os/sc-bos/internal/download"
	"github.com/smart-core-os/sc-bos/pkg/proto/logpb"
)

// URLGenerator signs and returns a URL embedding payload under typ. A
// *internal/download.Router satisfies this interface.
type URLGenerator interface {
	GenerateURL(typ string, payload []byte) (string, time.Time, error)
}

// FileDownloadHandler implements download.Handler for log-style file
// downloads. AllowedDir supplies the directory under which the streamed file
// must reside; it is called per request so callers with dynamic state (e.g.
// the log system's config reload via atomic.Pointer) can return current
// values via closure. An AllowedDir that returns "" disables the check.
type FileDownloadHandler struct {
	AllowedDir func() string
}

// ServeHTTP decodes the DownloadToken payload (placed on r's context by
// download.Router) and streams the referenced file or zip archive. The router
// has already verified signature and expiry before dispatching here.
func (h *FileDownloadHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	payload := download.PayloadFromContext(r.Context())
	token := &DownloadToken{}
	if err := proto.Unmarshal(payload, token); err != nil {
		http.Error(w, "corrupt download token", http.StatusBadRequest)
		return
	}
	var dir string
	if h.AllowedDir != nil {
		dir = h.AllowedDir()
	}

	if len(token.GetZipFiles()) > 0 {
		ServeZip(w, token.GetZipFiles(), dir)
		return
	}
	ServeFile(w, r, token.GetFilePath(), dir)
}

// GenerateFileDownloadURL resolves files per req, marshals a DownloadToken,
// signs the URL via urlGen, and returns the gRPC response shape used by both
// LogApi consumers (system-log and audit-log).
func GenerateFileDownloadURL(
	req *logpb.GetDownloadLogUrlRequest,
	glob, logDir string,
	urlGen URLGenerator,
	downloadType string,
) (*logpb.GetDownloadLogUrlResponse, error) {
	if urlGen == nil {
		return nil, status.Error(codes.Unavailable, "download URL generation is not configured")
	}

	if req.IncludeRotated {
		files, err := ResolveRotatedLogFiles(glob, logDir)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "resolve log files: %v", err)
		}
		if len(files) == 0 {
			return &logpb.GetDownloadLogUrlResponse{}, nil
		}
		payload, err := proto.Marshal(&DownloadToken{ZipFiles: files})
		if err != nil {
			return nil, status.Errorf(codes.Internal, "marshal token: %v", err)
		}
		u, _, err := urlGen.GenerateURL(downloadType, payload)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "generate url: %v", err)
		}
		return &logpb.GetDownloadLogUrlResponse{Files: []*logpb.GetDownloadLogUrlResponse_LogFile{{
			Url:         u,
			Filename:    "logs.zip",
			ContentType: "application/zip",
		}}}, nil
	}

	fp, err := ResolveLatestLogFile(glob)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "resolve log file: %v", err)
	}
	if fp == "" {
		return &logpb.GetDownloadLogUrlResponse{}, nil
	}
	info, err := os.Stat(fp)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "stat log file: %v", err)
	}
	payload, err := proto.Marshal(&DownloadToken{FilePath: fp})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "marshal token: %v", err)
	}
	u, _, err := urlGen.GenerateURL(downloadType, payload)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "generate url: %v", err)
	}
	return &logpb.GetDownloadLogUrlResponse{Files: []*logpb.GetDownloadLogUrlResponse_LogFile{{
		Url:         u,
		Filename:    filepath.Base(fp),
		SizeBytes:   info.Size(),
		ContentType: DetectContentType(fp),
	}}}, nil
}
