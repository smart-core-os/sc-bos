package bolthub

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/smart-core-os/sc-bos/pkg/proto/enrollmentpb"
	"github.com/smart-core-os/sc-bos/pkg/proto/hubpb"
	"github.com/smart-core-os/sc-bos/pkg/system/hub/internal/testtarget"
	"github.com/smart-core-os/sc-bos/pkg/system/hub/remote"
)

func TestServer_EnrollHubNode_setupCode(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	dbPath := filepath.Join(t.TempDir(), "hub.db")
	s, err := NewServer(dbPath, zap.NewNop())
	if err != nil {
		t.Fatal(err)
	}
	closed := false
	t.Cleanup(func() {
		if !closed {
			_ = s.Close()
		}
	})
	s.ManagerName = "hub"
	s.ManagerAddr = "hub.example:23557"
	s.Authority = testtarget.Authority(t)

	tgt := testtarget.New(t, true)
	enrol := func(code string) error {
		_, err := s.EnrollHubNode(ctx, &hubpb.EnrollHubNodeRequest{
			Node:      &hubpb.HubNode{Address: tgt.Addr, Name: "target"},
			SetupCode: code,
		})
		return err
	}
	checkNotStored := func() {
		t.Helper()
		if _, err := s.GetHubNode(ctx, &hubpb.GetHubNodeRequest{Address: tgt.Addr}); status.Code(err) != codes.NotFound {
			t.Errorf("GetHubNode after a failed enrol: err = %v, want NotFound", err)
		}
	}

	for _, code := range []string{"", "2222-2222"} {
		err := enrol(code)
		if status.Code(err) != codes.PermissionDenied || !strings.Contains(status.Convert(err).Message(), "setup code") {
			t.Errorf("EnrollHubNode(code %q) = %v, want PermissionDenied about the setup code", code, err)
		}
		checkNotStored()
	}

	if err := enrol(testtarget.Code); err != nil {
		t.Fatalf("EnrollHubNode with the right code: %v", err)
	}
	if _, err := s.GetHubNode(ctx, &hubpb.GetHubNodeRequest{Address: tgt.Addr}); err != nil {
		t.Errorf("GetHubNode after enrolling: %v", err)
	}
	if tgt.Enrollment() == nil {
		t.Error("target not enrolled")
	}

	if err := enrol(testtarget.Code); status.Code(err) != codes.AlreadyExists {
		t.Errorf("enrolling again = %v, want AlreadyExists", err)
	}

	if err := s.Close(); err != nil {
		t.Fatal(err)
	}
	closed = true
	raw, err := os.ReadFile(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	for _, form := range []string{testtarget.Code, strings.ReplaceAll(testtarget.Code, "-", "")} {
		if bytes.Contains(raw, []byte(form)) {
			t.Errorf("hub db contains the setup code %q", form)
		}
	}
}

// TestServer_EnrollHubNode_enrolledElsewhere checks a node already enrolled with another hub is reported as such,
// not as a rejected setup code.
func TestServer_EnrollHubNode_enrolledElsewhere(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	s, err := NewServer(filepath.Join(t.TempDir(), "hub.db"), zap.NewNop())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = s.Close() })
	s.ManagerName = "hub"
	s.Authority = testtarget.Authority(t)

	tgt := testtarget.New(t, true)
	_, err = remote.Enroll(ctx, &enrollmentpb.Enrollment{TargetAddress: tgt.Addr, ManagerName: "other-hub"},
		testtarget.Authority(t), testtarget.Code)
	if err != nil {
		t.Fatal(err)
	}

	_, err = s.EnrollHubNode(ctx, &hubpb.EnrollHubNodeRequest{
		Node:      &hubpb.HubNode{Address: tgt.Addr},
		SetupCode: testtarget.Code,
	})
	if status.Code(err) != codes.AlreadyExists {
		t.Errorf("EnrollHubNode = %v, want AlreadyExists", err)
	}
}
