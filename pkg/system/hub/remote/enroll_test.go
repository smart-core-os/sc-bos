package remote_test

import (
	"context"
	"strings"
	"testing"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/smart-core-os/sc-bos/pkg/proto/enrollmentpb"
	"github.com/smart-core-os/sc-bos/pkg/proto/metadatapb"
	"github.com/smart-core-os/sc-bos/pkg/system/hub/internal/testtarget"
	"github.com/smart-core-os/sc-bos/pkg/system/hub/remote"
)

const (
	wrongCode       = "2222-2222"
	requiresCodeMsg = "give its setup code"
	rejectedCodeMsg = "rejected the setup code"
)

func TestEnroll(t *testing.T) {
	tests := []struct {
		name     string
		policyOn bool
		code     string
		wantCode codes.Code
		wantMsg  string
	}{
		{name: "policy on, right code", policyOn: true, code: testtarget.Code, wantCode: codes.OK},
		{name: "policy on, code as typed", policyOn: true, code: " abcd efgh ", wantCode: codes.OK},
		{name: "policy on, no code", policyOn: true, wantCode: codes.PermissionDenied, wantMsg: requiresCodeMsg},
		{name: "policy on, wrong code", policyOn: true, code: wrongCode, wantCode: codes.PermissionDenied, wantMsg: rejectedCodeMsg},
		{name: "policy off, no code", policyOn: false, wantCode: codes.OK},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tgt := testtarget.New(t, tt.policyOn)
			en, err := remote.Enroll(testContext(t), newEnrollment(tgt), testtarget.Authority(t), tt.code)
			checkStatus(t, err, tt.wantCode, tt.wantMsg)

			got := tgt.Enrollment()
			if tt.wantCode != codes.OK {
				if got != nil {
					t.Errorf("target enrolled after a failed Enroll: %v", got)
				}
				return
			}
			if got == nil {
				t.Fatal("target not enrolled")
			}
			if got.ManagerName != "hub" || en.ManagerName != "hub" {
				t.Errorf("ManagerName: target has %q, Enroll returned %q, want %q", got.ManagerName, en.ManagerName, "hub")
			}
		})
	}
}

// TestEnroll_codeOnlyOnCreate checks only CreateEnrollment carries the code: every wrong code a target sees uses up
// one of its limited guesses, so the code must not ride along on GetEnrollment as well.
func TestEnroll_codeOnlyOnCreate(t *testing.T) {
	tgt := testtarget.New(t, true)
	if _, err := remote.Enroll(testContext(t), newEnrollment(tgt), testtarget.Authority(t), testtarget.Code); err != nil {
		t.Fatal(err)
	}
	for _, c := range tgt.Calls() {
		wantBearer := strings.HasSuffix(c.Method, "/CreateEnrollment")
		if c.Bearer != wantBearer {
			t.Errorf("%s: carried a bearer token = %v, want %v", c.Method, c.Bearer, wantBearer)
		}
	}
}

// TestEnroll_alreadyEnrolled checks a hub learns the node is already enrolled, not that its code was rejected, even
// though a node with policy on stops accepting its code once enrolled.
func TestEnroll_alreadyEnrolled(t *testing.T) {
	for _, policyOn := range []bool{true, false} {
		name := "policy off"
		if policyOn {
			name = "policy on"
		}
		t.Run(name, func(t *testing.T) {
			tgt := testtarget.New(t, policyOn)
			authority := testtarget.Authority(t)
			if _, err := remote.Enroll(testContext(t), newEnrollment(tgt), authority, testtarget.Code); err != nil {
				t.Fatal(err)
			}
			first := tgt.Enrollment()

			_, err := remote.Enroll(testContext(t), newEnrollment(tgt), authority, testtarget.Code)
			checkStatus(t, err, codes.AlreadyExists, "")
			if tgt.Enrollment() != first {
				t.Error("target's enrollment changed")
			}
		})
	}
}

func TestInspect(t *testing.T) {
	tests := []struct {
		name     string
		policyOn bool
		code     string
		wantCode codes.Code
		wantMsg  string
	}{
		{name: "policy on, right code", policyOn: true, code: testtarget.Code, wantCode: codes.OK},
		{name: "policy on, no code", policyOn: true, wantCode: codes.PermissionDenied, wantMsg: requiresCodeMsg},
		{name: "policy on, wrong code", policyOn: true, code: wrongCode, wantCode: codes.PermissionDenied, wantMsg: rejectedCodeMsg},
		{name: "policy off, no code", policyOn: false, wantCode: codes.OK},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tgt := testtarget.New(t, tt.policyOn)
			got, err := remote.Inspect(testContext(t), tgt.Addr, tt.code)
			checkStatus(t, err, tt.wantCode, tt.wantMsg)
			if tt.wantCode != codes.OK {
				return
			}
			if got.GetMetadata().GetName() != testtarget.Name {
				t.Errorf("metadata name = %q, want %q", got.GetMetadata().GetName(), testtarget.Name)
			}
			if len(got.GetPublicCerts()) == 0 {
				t.Error("no public certs")
			}
		})
	}
}

// TestInspect_nameless checks Inspect asks for the node's own metadata, the only metadata the setup code may read.
func TestInspect_nameless(t *testing.T) {
	tgt := testtarget.New(t, true)
	if _, err := remote.Inspect(testContext(t), tgt.Addr, testtarget.Code); err != nil {
		t.Fatal(err)
	}
	calls := tgt.Calls()
	if len(calls) != 1 || calls[0].Method != metadatapb.MetadataApi_GetMetadata_FullMethodName {
		t.Errorf("calls = %v, want one GetMetadata", calls)
	}
}

func newEnrollment(tgt *testtarget.Target) *enrollmentpb.Enrollment {
	return &enrollmentpb.Enrollment{
		TargetName:     "target",
		TargetAddress:  tgt.Addr,
		ManagerName:    "hub",
		ManagerAddress: "hub.example:23557",
	}
}

func testContext(t *testing.T) context.Context {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	t.Cleanup(cancel)
	return ctx
}

// checkStatus fails t unless err has code want and a message containing msg.
// The hub's caller authenticated with the hub, so a target's Unauthenticated must never reach them.
func checkStatus(t *testing.T, err error, want codes.Code, msg string) {
	t.Helper()
	got := status.Convert(err)
	if got.Code() == codes.Unauthenticated {
		t.Fatalf("got Unauthenticated, which would log the hub's caller out: %v", err)
	}
	if got.Code() != want {
		t.Fatalf("code = %v, want %v (err %v)", got.Code(), want, err)
	}
	if !strings.Contains(got.Message(), msg) {
		t.Errorf("message = %q, want it to contain %q", got.Message(), msg)
	}
}
