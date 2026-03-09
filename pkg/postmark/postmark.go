// Package postmark provides a minimal Postmark email client for sending reports.
package postmark

import (
	"bytes"
	"context"
	_ "embed"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

//go:embed powered-by-dark.png
var defaultLogoPNG []byte

// Config holds the Postmark configuration. ServerTokenPath is read from the
// JSON config; ServerToken is populated at parse time by calling ReadToken.
type Config struct {
	// ServerTokenPath is a path to a file containing the Postmark server token.
	ServerTokenPath string `json:"serverTokenPath,omitempty"`
	// ServerToken is populated at parse time by reading ServerTokenPath. Not serialised.
	ServerToken string   `json:"-"`
	From        string   `json:"from"`
	Recipients  []string `json:"recipients"`
	// LogoPath is an optional path to a PNG logo file to embed in report emails.
	// If empty, the default logo is used.
	LogoPath string `json:"logoPath,omitempty"`
	// APIURL is the Postmark API endpoint. Defaults to "https://api.postmarkapp.com/email" if empty.
	APIURL string `json:"apiUrl,omitempty"`
}

// ReadToken reads the server token from ServerTokenPath and stores it in ServerToken.
func (c *Config) ReadToken() error {
	if c.ServerTokenPath == "" {
		c.ServerTokenPath = "/run/secrets/postmark_token"
	}
	b, err := os.ReadFile(c.ServerTokenPath)
	if err != nil {
		return fmt.Errorf("postmark: failed to read server token from %s: %w", c.ServerTokenPath, err)
	}
	c.ServerToken = strings.TrimSpace(string(b))
	return nil
}

type emailPayload struct {
	From          string       `json:"From"`
	To            string       `json:"To"`
	Subject       string       `json:"Subject"`
	HtmlBody      string       `json:"HtmlBody"`
	MessageStream string       `json:"MessageStream"`
	Attachments   []attachment `json:"Attachments,omitempty"`
}

type attachment struct {
	Name        string `json:"Name"`
	Content     string `json:"Content"`
	ContentType string `json:"ContentType"`
	ContentID   string `json:"ContentID,omitempty"`
}

// LogoHTML is the HTML snippet to embed the Smart Core logo in an email body.
// Place it wherever the logo should appear.
const LogoHTML = `<img src="cid:smartcore-logo.png" alt="Smart Core" style="height:60px;">`

// SendReportEmail emails one or more xlsx files as attachments to all configured recipients.
// The htmlBody may reference the logo via <img src="cid:smartcore-logo.png">.
// All errors are accumulated and returned as a joined error.
func SendReportEmail(ctx context.Context, cfg *Config, filePaths []string, subject, htmlBody string) error {
	if cfg == nil || len(cfg.Recipients) == 0 {
		return nil
	}

	logoPNG := defaultLogoPNG
	if cfg.LogoPath != "" {
		data, err := os.ReadFile(cfg.LogoPath)
		if err != nil {
			return fmt.Errorf("postmark: failed to read logo file: %w", err)
		}
		logoPNG = data
	}

	attachments := []attachment{
		{
			Name:        "smartcore-logo.png",
			Content:     base64.StdEncoding.EncodeToString(logoPNG),
			ContentType: "image/png",
			ContentID:   "cid:smartcore-logo.png",
		},
	}

	var errs []error

	for _, filePath := range filePaths {
		data, err := os.ReadFile(filePath)
		if err != nil {
			errs = append(errs, fmt.Errorf("postmark: failed to read report file %s: %w", filePath, err))
			continue
		}
		attachments = append(attachments, attachment{
			Name:        filepath.Base(filePath),
			Content:     base64.StdEncoding.EncodeToString(data),
			ContentType: "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet",
		})
	}

	apiURL := cfg.APIURL
	if apiURL == "" {
		apiURL = "https://api.postmarkapp.com/email"
	}

	for _, recipient := range cfg.Recipients {
		payload := emailPayload{
			From:          cfg.From,
			To:            recipient,
			Subject:       subject,
			HtmlBody:      htmlBody,
			MessageStream: "outbound",
			Attachments:   attachments,
		}

		body, err := json.Marshal(payload)
		if err != nil {
			errs = append(errs, fmt.Errorf("postmark: failed to marshal payload for %s: %w", recipient, err))
			continue
		}

		req, err := http.NewRequestWithContext(ctx, http.MethodPost, apiURL, bytes.NewReader(body))
		if err != nil {
			errs = append(errs, fmt.Errorf("postmark: failed to create request for %s: %w", recipient, err))
			continue
		}
		req.Header.Set("Accept", "application/json")
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-Postmark-Server-Token", cfg.ServerToken)

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			errs = append(errs, fmt.Errorf("postmark: failed to send email to %s: %w", recipient, err))
			continue
		}
		resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			errs = append(errs, fmt.Errorf("postmark: non-200 response for %s: status %d", recipient, resp.StatusCode))
		}
	}

	return errors.Join(errs...)
}
