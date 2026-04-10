package config

import (
	"strings"
	"testing"
)

func TestReadBytes_InvalidJSON(t *testing.T) {
	_, err := ReadBytes([]byte(`not valid json`))
	if err == nil {
		t.Fatal("expected error for invalid JSON, got nil")
	}
	// The error should come directly from json.Unmarshal, not be masked by a
	// later validation check or Destination.Parse().
	if strings.HasPrefix(err.Error(), "destination.") {
		t.Errorf("invalid JSON should return a parse error, not a validation/config error: %v", err)
	}
}

func TestReadBytes_MissingHost(t *testing.T) {
	data := []byte(`{"destination": {"to": ["user@example.com"]}}`)
	_, err := ReadBytes(data)
	if err == nil {
		t.Fatal("expected error for missing host, got nil")
	}
	if !strings.Contains(err.Error(), "destination.host") {
		t.Errorf("expected host error, got: %v", err)
	}
}

func TestReadBytes_MissingRecipients(t *testing.T) {
	data := []byte(`{"destination": {"host": "smtp.example.com"}}`)
	_, err := ReadBytes(data)
	if err == nil {
		t.Fatal("expected error for missing recipients, got nil")
	}
	if !strings.Contains(err.Error(), "destination.recipients") {
		t.Errorf("expected recipients error, got: %v", err)
	}
}
