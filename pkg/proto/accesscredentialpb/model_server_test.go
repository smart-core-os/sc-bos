package accesscredentialpb

import (
	stdcmp "cmp"
	"context"
	"slices"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/testing/protocmp"
	"google.golang.org/protobuf/types/known/fieldmaskpb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func TestModelServer_crud(t *testing.T) {
	ctx := context.Background()
	srv := NewModelServer(NewModel(nil))

	created, err := srv.CreateCredential(ctx, &CreateCredentialRequest{Credential: &Credential{
		Id:    "ignored",
		Type:  "fob",
		Value: "1234",
		Kind:  Credential_PIN, // output only, derived from type
	}})
	if err != nil {
		t.Fatalf("CreateCredential: %v", err)
	}
	want := &Credential{Id: "1", Type: "fob", Kind: Credential_FOB, Value: "1234", State: Credential_ACTIVE}
	if diff := cmp.Diff(want, created, protocmp.Transform()); diff != "" {
		t.Fatalf("CreateCredential (-want,+got)\n%s", diff)
	}

	got, err := srv.GetCredential(ctx, &GetCredentialRequest{Id: "1"})
	if err != nil {
		t.Fatalf("GetCredential: %v", err)
	}
	if diff := cmp.Diff(want, got, protocmp.Transform()); diff != "" {
		t.Fatalf("GetCredential (-want,+got)\n%s", diff)
	}

	updated, err := srv.UpdateCredential(ctx, &UpdateCredentialRequest{
		Credential: &Credential{Id: "1", State: Credential_LOST, Value: "not applied"},
		UpdateMask: &fieldmaskpb.FieldMask{Paths: []string{"state"}},
	})
	if err != nil {
		t.Fatalf("UpdateCredential: %v", err)
	}
	want.State = Credential_LOST
	if diff := cmp.Diff(want, updated, protocmp.Transform()); diff != "" {
		t.Fatalf("UpdateCredential (-want,+got)\n%s", diff)
	}

	if _, err := srv.DeleteCredential(ctx, &DeleteCredentialRequest{Id: "1"}); err != nil {
		t.Fatalf("DeleteCredential: %v", err)
	}
	if _, err := srv.GetCredential(ctx, &GetCredentialRequest{Id: "1"}); status.Code(err) != codes.NotFound {
		t.Fatalf("GetCredential after delete: want NotFound, got %v", err)
	}
	if _, err := srv.DeleteCredential(ctx, &DeleteCredentialRequest{Id: "1"}); status.Code(err) != codes.NotFound {
		t.Fatalf("DeleteCredential missing: want NotFound, got %v", err)
	}
	if _, err := srv.DeleteCredential(ctx, &DeleteCredentialRequest{Id: "1", AllowMissing: true}); err != nil {
		t.Fatalf("DeleteCredential allow_missing: %v", err)
	}
}

func TestModelServer_CreateCredential_errors(t *testing.T) {
	tests := []struct {
		name string
		c    *Credential
		code codes.Code
	}{
		{"no type", &Credential{Value: "1"}, codes.InvalidArgument},
		{"unknown type", &Credential{Type: "nope", Value: "1"}, codes.InvalidArgument},
		{"caller supplied without value", &Credential{Type: "fob"}, codes.InvalidArgument},
		{"system allocated with value", &Credential{Type: "mobile", Value: "1", Invitation: &Credential_Invitation{Email: "a@b"}}, codes.InvalidArgument},
		{"missing invitation", &Credential{Type: "mobile"}, codes.InvalidArgument},
		{"unwritable state", &Credential{Type: "plate", Value: "AB12CDE", State: Credential_LOST}, codes.InvalidArgument},
		{"duplicate", &Credential{Type: "fob", Value: "existing"}, codes.AlreadyExists},
		{"expires before active", &Credential{Type: "fob", Value: "1", ActiveTime: timestamppb.New(time.Unix(200, 0)), ExpireTime: timestamppb.New(time.Unix(100, 0))}, codes.InvalidArgument},
		{"nil", nil, codes.InvalidArgument},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srv := NewModelServer(NewModel(nil))
			if _, err := srv.CreateCredential(context.Background(), &CreateCredentialRequest{Credential: &Credential{Type: "fob", Value: "existing"}}); err != nil {
				t.Fatalf("seed: %v", err)
			}
			_, err := srv.CreateCredential(context.Background(), &CreateCredentialRequest{Credential: tt.c})
			if got := status.Code(err); got != tt.code {
				t.Fatalf("want %v, got %v", tt.code, err)
			}
		})
	}
}

func TestModelServer_CreateCredential_allocated(t *testing.T) {
	srv := NewModelServer(NewModel(nil))
	got, err := srv.CreateCredential(context.Background(), &CreateCredentialRequest{Credential: &Credential{
		Type:       "mobile",
		Invitation: &Credential_Invitation{Email: "someone@example.com", Status: "accepted"},
	}})
	if err != nil {
		t.Fatalf("CreateCredential: %v", err)
	}
	if got.Value == "" {
		t.Errorf("want an allocated value, got none")
	}
	if got.Invitation.GetStatus() != "" {
		t.Errorf("invitation.status is output only, got %q", got.Invitation.GetStatus())
	}
}

func TestModelServer_UpdateCredential_errors(t *testing.T) {
	ctx := context.Background()
	srv := NewModelServer(NewModel(nil))
	for _, v := range []string{"a", "b"} {
		if _, err := srv.CreateCredential(ctx, &CreateCredentialRequest{Credential: &Credential{Type: "fob", Value: v}}); err != nil {
			t.Fatalf("seed: %v", err)
		}
	}

	tests := []struct {
		name string
		req  *UpdateCredentialRequest
		code codes.Code
	}{
		{"no id", &UpdateCredentialRequest{Credential: &Credential{Value: "c"}}, codes.InvalidArgument},
		{"missing", &UpdateCredentialRequest{Credential: &Credential{Id: "99", Type: "fob", Value: "c"}}, codes.NotFound},
		{"read only field", &UpdateCredentialRequest{
			Credential: &Credential{Id: "1", Kind: Credential_CARD},
			UpdateMask: &fieldmaskpb.FieldMask{Paths: []string{"kind"}},
		}, codes.InvalidArgument},
		{"duplicate", &UpdateCredentialRequest{
			Credential: &Credential{Id: "1", Value: "b"},
			UpdateMask: &fieldmaskpb.FieldMask{Paths: []string{"value"}},
		}, codes.AlreadyExists},
		{"clear value", &UpdateCredentialRequest{
			Credential: &Credential{Id: "1"},
			UpdateMask: &fieldmaskpb.FieldMask{Paths: []string{"value"}},
		}, codes.InvalidArgument},
		{"clear required invitation", &UpdateCredentialRequest{
			Credential: &Credential{Id: "3"},
			UpdateMask: &fieldmaskpb.FieldMask{Paths: []string{"invitation"}},
		}, codes.InvalidArgument},
		{"change to a system allocated type", &UpdateCredentialRequest{
			Credential: &Credential{Id: "1", Type: "mobile", Invitation: &Credential_Invitation{Email: "a@b"}},
			UpdateMask: &fieldmaskpb.FieldMask{Paths: []string{"type", "invitation"}},
		}, codes.InvalidArgument},
		{"expires before active", &UpdateCredentialRequest{
			Credential: &Credential{Id: "1", ActiveTime: timestamppb.New(time.Unix(200, 0)), ExpireTime: timestamppb.New(time.Unix(100, 0))},
			UpdateMask: &fieldmaskpb.FieldMask{Paths: []string{"active_time", "expire_time"}},
		}, codes.InvalidArgument},
	}
	if _, err := srv.CreateCredential(ctx, &CreateCredentialRequest{Credential: &Credential{Type: "mobile", Invitation: &Credential_Invitation{Email: "a@b"}}}); err != nil {
		t.Fatalf("seed mobile: %v", err)
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := srv.UpdateCredential(ctx, tt.req)
			if got := status.Code(err); got != tt.code {
				t.Fatalf("want %v, got %v", tt.code, err)
			}
		})
	}
}

// Leaving state out of an update keeps the state, rather than re-activating a lost credential.
func TestModelServer_UpdateCredential_unsetStateKeepsLost(t *testing.T) {
	ctx := context.Background()
	srv := NewModelServer(NewModel(nil))
	if _, err := srv.CreateCredential(ctx, &CreateCredentialRequest{Credential: &Credential{Type: "fob", Value: "1", State: Credential_LOST}}); err != nil {
		t.Fatalf("seed: %v", err)
	}
	for _, req := range []*UpdateCredentialRequest{
		{Credential: &Credential{Id: "1", Type: "card", Value: "1"}},
		{Credential: &Credential{Id: "1", Type: "fob"}, UpdateMask: &fieldmaskpb.FieldMask{Paths: []string{"type", "state"}}},
	} {
		got, err := srv.UpdateCredential(ctx, req)
		if err != nil {
			t.Fatalf("UpdateCredential(%v): %v", req, err)
		}
		if got.State != Credential_LOST {
			t.Errorf("UpdateCredential(%v): want LOST, got %v", req, got.State)
		}
	}
}

func TestModelServer_UpdateCredential_invitationStatus(t *testing.T) {
	ctx := context.Background()
	srv := NewModelServer(NewModel(nil))
	if _, err := srv.CreateCredential(ctx, &CreateCredentialRequest{Credential: &Credential{Type: "fob", Value: "1"}}); err != nil {
		t.Fatalf("seed: %v", err)
	}
	got, err := srv.UpdateCredential(ctx, &UpdateCredentialRequest{
		Credential: &Credential{Id: "1", Invitation: &Credential_Invitation{Email: "a@b", Status: "accepted"}},
		UpdateMask: &fieldmaskpb.FieldMask{Paths: []string{"invitation"}},
	})
	if err != nil {
		t.Fatalf("UpdateCredential: %v", err)
	}
	if got.Invitation.GetEmail() != "a@b" || got.Invitation.GetStatus() != "" {
		t.Errorf("invitation.status is output only, want only the email set, got %v", got.Invitation)
	}
}

func TestModelServer_readMask(t *testing.T) {
	ctx := context.Background()
	srv := NewModelServer(NewModel(nil))
	if _, err := srv.CreateCredential(ctx, &CreateCredentialRequest{Credential: &Credential{Type: "fob", Value: "1", More: map[string]string{"x": "y"}}}); err != nil {
		t.Fatalf("seed: %v", err)
	}
	// a path into the more map used to panic, taking the process down with it
	for _, paths := range [][]string{{"more.x"}, {"no_such_field"}} {
		mask := &fieldmaskpb.FieldMask{Paths: paths}
		if _, err := srv.GetCredential(ctx, &GetCredentialRequest{Id: "1", ReadMask: mask}); status.Code(err) != codes.InvalidArgument {
			t.Errorf("GetCredential %v: want InvalidArgument, got %v", paths, err)
		}
		if _, err := srv.ListCredentials(ctx, &ListCredentialsRequest{ReadMask: mask}); status.Code(err) != codes.InvalidArgument {
			t.Errorf("ListCredentials %v: want InvalidArgument, got %v", paths, err)
		}
	}
}

func TestModelServer_emptyID(t *testing.T) {
	ctx := context.Background()
	srv := NewModelServer(NewModel(nil))
	if _, err := srv.GetCredential(ctx, &GetCredentialRequest{}); status.Code(err) != codes.InvalidArgument {
		t.Errorf("GetCredential: want InvalidArgument, got %v", err)
	}
	for _, allowMissing := range []bool{false, true} {
		if _, err := srv.DeleteCredential(ctx, &DeleteCredentialRequest{AllowMissing: allowMissing}); status.Code(err) != codes.InvalidArgument {
			t.Errorf("DeleteCredential allow_missing=%v: want InvalidArgument, got %v", allowMissing, err)
		}
	}
}

func TestModelServer_ListCredentials_paging(t *testing.T) {
	ctx := context.Background()
	srv := NewModelServer(NewModel(nil))
	for i := range 12 {
		if _, err := srv.CreateCredential(ctx, &CreateCredentialRequest{Credential: &Credential{Type: "card"}}); err != nil {
			t.Fatalf("seed %d: %v", i, err)
		}
	}

	var ids []string
	req := &ListCredentialsRequest{PageSize: 5, ReadMask: &fieldmaskpb.FieldMask{Paths: []string{"id"}}}
	for page := 0; ; page++ {
		if page > 5 {
			t.Fatalf("too many pages")
		}
		res, err := srv.ListCredentials(ctx, req)
		if err != nil {
			t.Fatalf("ListCredentials: %v", err)
		}
		if res.TotalSize != 12 {
			t.Errorf("total_size: want 12, got %d", res.TotalSize)
		}
		for _, c := range res.Credentials {
			if c.Value != "" {
				t.Errorf("read_mask not applied: %v", c)
			}
			ids = append(ids, c.Id)
		}
		if res.NextPageToken == "" {
			break
		}
		req.PageToken = res.NextPageToken
	}
	want := []string{"1", "2", "3", "4", "5", "6", "7", "8", "9", "10", "11", "12"}
	if diff := cmp.Diff(want, ids); diff != "" {
		t.Fatalf("ids (-want,+got)\n%s", diff)
	}

	if _, err := srv.ListCredentials(ctx, &ListCredentialsRequest{PageToken: "!!"}); status.Code(err) != codes.InvalidArgument {
		t.Fatalf("bad page token: want InvalidArgument, got %v", err)
	}
}

func TestCompareIDs(t *testing.T) {
	ids := []string{"b", "10", "a", "9", "1"}
	want := []string{"1", "9", "10", "a", "b"}
	for i := range ids {
		for j := range ids {
			a, b := ids[i], ids[j]
			wantCmp := slices.Index(want, a) - slices.Index(want, b)
			if got := CompareIDs(a, b); sign(got) != sign(wantCmp) {
				t.Errorf("CompareIDs(%q, %q) = %d, want sign %d", a, b, got, sign(wantCmp))
			}
		}
	}
}

func sign(i int) int {
	return stdcmp.Compare(i, 0)
}
