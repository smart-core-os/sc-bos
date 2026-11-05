package remote_test

import (
	"context"
	"slices"
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
		pin      bool // pass the target's certs as remoteRoots
		wantCode codes.Code
		wantMsg  string
	}{
		{name: "policy on, right code", policyOn: true, code: testtarget.Code, pin: true, wantCode: codes.OK},
		{name: "policy on, code as typed", policyOn: true, code: " abcd efgh ", pin: true, wantCode: codes.OK},
		{name: "policy on, no code", policyOn: true, pin: true, wantCode: codes.PermissionDenied, wantMsg: requiresCodeMsg},
		{name: "policy on, wrong code", policyOn: true, code: wrongCode, pin: true, wantCode: codes.PermissionDenied, wantMsg: rejectedCodeMsg},
		{name: "policy off, no code", policyOn: false, wantCode: codes.OK},
		{name: "policy off, no code, pinned", policyOn: false, pin: true, wantCode: codes.OK},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tgt := testtarget.New(t, tt.policyOn)
			var roots []string
			if tt.pin {
				roots = tgt.Certs
			}
			en, err := remote.Enroll(testContext(t), newEnrollment(tgt), testtarget.Authority(t), tt.code, roots...)
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

// TestEnroll_codeNeedsCerts checks a code without certs to verify the target against never leaves the hub, so it
// can't be captured on the way to whoever answers at the target's address.
func TestEnroll_codeNeedsCerts(t *testing.T) {
	tests := []struct {
		name  string
		certs []string
	}{
		{name: "no certs"},
		{name: "not PEM", certs: []string{"not a certificate"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tgt := testtarget.New(t, true)
			_, err := remote.Enroll(testContext(t), newEnrollment(tgt), testtarget.Authority(t), testtarget.Code, tt.certs...)
			checkStatus(t, err, codes.InvalidArgument, "public certs")
			if calls := tgt.Calls(); len(calls) != 0 {
				t.Errorf("target received %v, want no calls", calls)
			}
		})
	}
}

// TestEnroll_otherCerts checks the code goes nowhere when the node at the address isn't the one that was inspected.
func TestEnroll_otherCerts(t *testing.T) {
	tgt := testtarget.New(t, true)
	other := testtarget.New(t, true)
	_, err := remote.Enroll(testContext(t), newEnrollment(tgt), testtarget.Authority(t), testtarget.Code, other.Certs...)
	if code := status.Code(err); code != codes.Unavailable {
		t.Errorf("Enroll against a target presenting other certs = %v, want Unavailable", err)
	}
	for _, c := range tgt.Calls() {
		if c.Bearer || strings.HasSuffix(c.Method, "/CreateEnrollment") {
			t.Errorf("target received %v", c)
		}
	}
	if tgt.Enrollment() != nil {
		t.Error("target enrolled")
	}
}

// TestEnroll_codeOnlyOnCreate checks only CreateEnrollment carries the code: every wrong code a target sees uses up
// one of its limited guesses, so the code must not ride along on GetEnrollment as well.
func TestEnroll_codeOnlyOnCreate(t *testing.T) {
	tgt := testtarget.New(t, true)
	if _, err := remote.Enroll(testContext(t), newEnrollment(tgt), testtarget.Authority(t), testtarget.Code, tgt.Certs...); err != nil {
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
			if _, err := remote.Enroll(testContext(t), newEnrollment(tgt), authority, testtarget.Code, tgt.Certs...); err != nil {
				t.Fatal(err)
			}
			first := tgt.Enrollment()

			_, err := remote.Enroll(testContext(t), newEnrollment(tgt), authority, testtarget.Code, tgt.Certs...)
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
		wantName string // empty for no metadata
	}{
		{name: "policy on", policyOn: true},
		{name: "policy off", policyOn: false, wantName: testtarget.Name},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tgt := testtarget.New(t, tt.policyOn)
			got, err := remote.Inspect(testContext(t), tgt.Addr)
			if err != nil {
				t.Fatal(err)
			}
			if tt.wantName == "" && got.GetMetadata() != nil {
				t.Errorf("metadata = %v, want none", got.GetMetadata())
			}
			if got.GetMetadata().GetName() != tt.wantName {
				t.Errorf("metadata name = %q, want %q", got.GetMetadata().GetName(), tt.wantName)
			}
			if !slices.Equal(got.GetPublicCerts(), tgt.Certs) {
				t.Errorf("public certs = %q, want %q", got.GetPublicCerts(), tgt.Certs)
			}
			calls := tgt.Calls()
			if len(calls) != 1 || calls[0].Method != metadatapb.MetadataApi_GetMetadata_FullMethodName || calls[0].Bearer {
				t.Errorf("calls = %v, want one GetMetadata without a bearer token", calls)
			}
		})
	}
}

// TestInspect_pinsEnroll checks the certs Inspect returns are the ones Enroll needs to send the setup code.
func TestInspect_pinsEnroll(t *testing.T) {
	tgt := testtarget.New(t, true)
	got, err := remote.Inspect(testContext(t), tgt.Addr)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := remote.Enroll(testContext(t), newEnrollment(tgt), testtarget.Authority(t), testtarget.Code, got.GetPublicCerts()...); err != nil {
		t.Fatal(err)
	}
	if tgt.Enrollment() == nil {
		t.Error("target not enrolled")
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
