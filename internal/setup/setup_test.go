package setup

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zaptest/observer"
	"golang.org/x/time/rate"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/smart-core-os/sc-bos/internal/account"
	"github.com/smart-core-os/sc-bos/internal/account/queries"
	"github.com/smart-core-os/sc-bos/pkg/proto/accountpb"
	"github.com/smart-core-os/sc-bos/pkg/proto/setuppb"
)

type fixture struct {
	dir      string
	accounts *account.Store
	enrolled atomic.Bool
	logs     *observer.ObservedLogs
}

func newFixture(t *testing.T) *fixture {
	t.Helper()
	accounts := account.NewMemoryStore(zap.NewNop())
	t.Cleanup(func() { _ = accounts.Close() })
	return &fixture{dir: filepath.Join(t.TempDir(), "setup"), accounts: accounts}
}

func (f *fixture) open(t *testing.T, codeFile string) (*Codes, error) {
	t.Helper()
	core, logs := observer.New(zap.InfoLevel)
	f.logs = logs
	return Open(context.Background(), Config{
		Dir:      f.dir,
		CodeFile: codeFile,
		Accounts: f.accounts,
		Enrolled: f.enrolled.Load,
		Logger:   zap.New(core),
	})
}

func (f *fixture) mustOpen(t *testing.T) *Codes {
	t.Helper()
	c, err := f.open(t, "")
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	return c
}

// shownCode returns the code as the installer reads it from the code file, or "" if there is no code file.
func (f *fixture) shownCode() string {
	raw, err := os.ReadFile(filepath.Join(f.dir, codeFileName))
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(raw))
}

// logged reports whether any field or message logged by the last open contains s.
func (f *fixture) logged(s string) bool {
	for _, e := range f.logs.All() {
		if strings.Contains(e.Message, s) {
			return true
		}
		for _, v := range e.ContextMap() {
			if strings.Contains(fmt.Sprint(v), s) {
				return true
			}
		}
	}
	return false
}

func (f *fixture) addAccount(t *testing.T, legacyRole string) {
	t.Helper()
	ctx := context.Background()
	err := f.accounts.Write(ctx, func(tx *account.Tx) error {
		acc, err := tx.CreateAccount(ctx, queries.CreateAccountParams{
			DisplayName: "Someone",
			Type:        accountpb.Account_USER_ACCOUNT.String(),
		})
		if err != nil {
			return err
		}
		roles, err := tx.ListRolesWithLegacyRole(ctx, sql.NullString{String: legacyRole, Valid: true})
		if err != nil {
			return err
		}
		if len(roles) == 0 {
			return errors.New("no role for " + legacyRole)
		}
		_, err = tx.CreateRoleAssignment(ctx, queries.CreateRoleAssignmentParams{AccountID: acc.ID, RoleID: roles[0].ID})
		return err
	})
	if err != nil {
		t.Fatalf("add %s account: %v", legacyRole, err)
	}
}

func TestOpen_pending(t *testing.T) {
	f := newFixture(t)
	c := f.mustOpen(t)

	shown := f.shownCode()
	if shown == "" {
		t.Fatal("no setup code file")
	}
	code, err := normaliseCode(shown)
	if err != nil {
		t.Fatalf("code file holds %q: %v", shown, err)
	}
	// The log is served to read-only callers, so it must say where the code is without giving it away.
	if f.logged(shown) || f.logged(code) {
		t.Error("setup code was logged")
	}
	if !f.logged(filepath.Join(f.dir, codeFileName)) {
		t.Error("log doesn't say where the setup code is")
	}
	state, err := os.ReadFile(filepath.Join(f.dir, stateFileName))
	if err != nil {
		t.Fatalf("state file: %v", err)
	}
	if strings.Contains(string(state), code) {
		t.Errorf("state file holds the code itself: %s", state)
	}
	if runtime.GOOS != "windows" { // no POSIX mode bits on Windows
		for _, name := range []string{codeFileName, stateFileName} {
			info, err := os.Stat(filepath.Join(f.dir, name))
			if err != nil {
				t.Fatal(err)
			}
			if perm := info.Mode().Perm(); perm != 0o600 {
				t.Errorf("%s mode %v, want 0600", name, perm)
			}
		}
	}

	claimed, err := c.Claimed(context.Background())
	if err != nil || claimed {
		t.Errorf("Claimed = %v, %v; want false", claimed, err)
	}

	// as a person might type it
	for _, tkn := range []string{shown, code, strings.ToLower(shown), " " + code[:4] + " " + code[4:]} {
		claims, err := c.ValidateAccessToken(context.Background(), tkn)
		if err != nil {
			t.Fatalf("ValidateAccessToken(%q): %v", tkn, err)
		}
		if !slices.Equal(claims.SystemRoles, []string{Role}) || len(claims.Permissions) != 0 {
			t.Errorf("claims %+v, want only the setup role", claims)
		}
	}
}

func TestOpen_noAccounts(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "setup")
	c, err := Open(context.Background(), Config{Dir: dir})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(dir); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("setup dir created without local accounts: %v", err)
	}
	if claimed, _ := c.Claimed(context.Background()); !claimed {
		t.Error("Claimed = false without local accounts")
	}
	if _, err := c.ValidateAccessToken(context.Background(), "ABCD-EFGH"); !errors.Is(err, ErrNotPending) {
		t.Errorf("ValidateAccessToken = %v, want ErrNotPending", err)
	}
}

func TestOpen_alreadyClaimed(t *testing.T) {
	for _, tc := range []struct {
		name  string
		setup func(f *fixture)
	}{
		{"admin", func(f *fixture) { f.addAccount(t, "admin") }},
		// Anyone with an account could read the code if it were within reach, and enrol the node to take it over.
		{"viewer", func(f *fixture) { f.addAccount(t, "viewer") }},
		{"operator", func(f *fixture) { f.addAccount(t, "operator") }},
		{"enrolled", func(f *fixture) { f.enrolled.Store(true) }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			f := newFixture(t)
			tc.setup(f)
			c := f.mustOpen(t)
			if len(f.logs.All()) != 0 {
				t.Errorf("logged setup for a claimed node: %v", f.logs.All())
			}
			if _, err := os.Stat(f.dir); !errors.Is(err, os.ErrNotExist) {
				t.Errorf("setup dir created for a claimed node: %v", err)
			}
			if claimed, _ := c.Claimed(context.Background()); !claimed {
				t.Error("Claimed = false")
			}
		})
	}
}

func TestOpen_otherLogins(t *testing.T) {
	f := newFixture(t)
	core, logs := observer.New(zap.InfoLevel)
	f.logs = logs
	c, err := Open(context.Background(), Config{
		Dir:         f.dir,
		Accounts:    f.accounts,
		OtherLogins: true,
		Enrolled:    f.enrolled.Load,
		Logger:      zap.New(core),
	})
	if err != nil {
		t.Fatal(err)
	}
	if f.shownCode() != "" {
		t.Error("setup code issued for a node with other logins")
	}
	if claimed, _ := c.Claimed(context.Background()); !claimed {
		t.Error("Claimed = false for a node with other logins")
	}
}

func TestCodes_stopsWorkingOnceClaimed(t *testing.T) {
	ctx := context.Background()

	t.Run("account created", func(t *testing.T) {
		f := newFixture(t)
		c := f.mustOpen(t)
		code := f.shownCode()
		f.addAccount(t, "viewer")
		if _, err := c.ValidateAccessToken(ctx, code); !errors.Is(err, ErrNotPending) {
			t.Errorf("ValidateAccessToken = %v, want ErrNotPending", err)
		}
		if claimed, _ := c.Claimed(ctx); !claimed {
			t.Error("Claimed = false after an account was created")
		}
	})

	t.Run("enrolled then unenrolled", func(t *testing.T) {
		f := newFixture(t)
		c := f.mustOpen(t)
		code := f.shownCode()
		f.enrolled.Store(true)
		if _, err := c.ValidateAccessToken(ctx, code); !errors.Is(err, ErrNotPending) {
			t.Errorf("enrolled: ValidateAccessToken = %v, want ErrNotPending", err)
		}
		// leaving the hub puts the node back to waiting, so a hub can enrol it again
		f.enrolled.Store(false)
		if _, err := c.ValidateAccessToken(ctx, code); err != nil {
			t.Errorf("unenrolled: ValidateAccessToken = %v", err)
		}
	})
}

func TestCodes_wrongCode(t *testing.T) {
	f := newFixture(t)
	c := f.mustOpen(t)
	code, _ := normaliseCode(f.shownCode())
	wrong := strings.Repeat("2", codeLen)
	if code == wrong {
		wrong = strings.Repeat("3", codeLen)
	}
	if _, err := c.ValidateAccessToken(context.Background(), wrong); !errors.Is(err, ErrWrongCode) {
		t.Errorf("ValidateAccessToken = %v, want ErrWrongCode", err)
	}
}

func TestCodes_limitsWrongGuesses(t *testing.T) {
	ctx := context.Background()
	f := newFixture(t)
	c := f.mustOpen(t)
	c.limiter = rate.NewLimiter(rate.Every(time.Hour), 3)
	code, _ := normaliseCode(f.shownCode())
	wrong := strings.Repeat("2", codeLen)
	if code == wrong {
		wrong = strings.Repeat("3", codeLen)
	}

	// Correct codes and other kinds of token don't use up guesses.
	for range 10 {
		if _, err := c.ValidateAccessToken(ctx, code); err != nil {
			t.Fatalf("correct code: %v", err)
		}
		if _, err := c.ValidateAccessToken(ctx, "eyJhbGciOiJIUzI1NiJ9.e30.sig"); !errors.Is(err, errMalformedCode) {
			t.Fatalf("jwt: %v, want errMalformedCode", err)
		}
	}

	for i := range 3 {
		if err := c.Check(ctx, wrong); !errors.Is(err, ErrWrongCode) {
			t.Fatalf("guess %d: %v, want ErrWrongCode", i, err)
		}
	}
	// Bearer tokens and CompleteSetup share the limit.
	if _, err := c.ValidateAccessToken(ctx, wrong); !errors.Is(err, ErrTooManyAttempts) {
		t.Errorf("bearer guess after limit: %v, want ErrTooManyAttempts", err)
	}
	if err := c.Check(ctx, code); !errors.Is(err, ErrTooManyAttempts) {
		t.Errorf("correct code after limit: %v, want ErrTooManyAttempts", err)
	}
}

func TestOpen_restartKeepsCode(t *testing.T) {
	ctx := context.Background()
	f := newFixture(t)
	f.mustOpen(t)
	first := f.shownCode()

	c := f.mustOpen(t)
	if got := f.shownCode(); got != first {
		t.Errorf("after restart code is %q, want %q", got, first)
	}
	if _, err := c.ValidateAccessToken(ctx, first); err != nil {
		t.Errorf("after restart: %v", err)
	}

	// An installer may delete the code file once it has shown the code.
	if err := os.Remove(filepath.Join(f.dir, codeFileName)); err != nil {
		t.Fatal(err)
	}
	c = f.mustOpen(t)
	if got := f.shownCode(); got != "" {
		t.Errorf("code file rewritten with %q", got)
	}
	if f.logged(codeFileName) || !f.logged(stateFileName) {
		t.Errorf("without the code file, want the log to point at the state file: %v", f.logs.All())
	}
	if _, err := c.ValidateAccessToken(ctx, first); err != nil {
		t.Errorf("after code file removed: %v", err)
	}
}

func TestOpen_codeFile(t *testing.T) {
	ctx := context.Background()
	f := newFixture(t)
	codeFile := filepath.Join(t.TempDir(), "code")
	if err := os.WriteFile(codeFile, []byte("abcd-efgh\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	c, err := f.open(t, codeFile)
	if err != nil {
		t.Fatal(err)
	}
	if f.shownCode() != "" {
		t.Error("code from a code file was logged")
	}
	if _, err := os.Stat(filepath.Join(f.dir, codeFileName)); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("wrote %s for a supplied code: %v", codeFileName, err)
	}
	if _, err := c.ValidateAccessToken(ctx, "ABCDEFGH"); err != nil {
		t.Errorf("supplied code: %v", err)
	}

	t.Run("malformed", func(t *testing.T) {
		bad := filepath.Join(t.TempDir(), "code")
		if err := os.WriteFile(bad, []byte("0OIL1234"), 0o600); err != nil {
			t.Fatal(err)
		}
		if _, err := newFixture(t).open(t, bad); err == nil {
			t.Error("Open accepted a malformed setup code file")
		}
	})
	t.Run("missing", func(t *testing.T) {
		if _, err := newFixture(t).open(t, filepath.Join(t.TempDir(), "nope")); err == nil {
			t.Error("Open accepted a missing setup code file")
		}
	})
}

func TestOpen_corruptState(t *testing.T) {
	f := newFixture(t)
	if err := os.MkdirAll(f.dir, 0o700); err != nil {
		t.Fatal(err)
	}
	raw, _ := json.Marshal(state{CodeHash: "not hex"})
	if err := os.WriteFile(filepath.Join(f.dir, stateFileName), raw, 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := f.open(t, ""); err == nil {
		t.Error("Open accepted a corrupt state file")
	}
}

func TestNewCode(t *testing.T) {
	seen := map[string]bool{}
	for range 100 {
		code, err := newCode()
		if err != nil {
			t.Fatal(err)
		}
		if got, err := normaliseCode(code); err != nil || got != code {
			t.Fatalf("newCode() = %q, not a normalised code (%q, %v)", code, got, err)
		}
		seen[code] = true
	}
	if len(seen) < 100 {
		t.Errorf("only %d distinct codes in 100", len(seen))
	}
}

func TestServer(t *testing.T) {
	ctx := context.Background()
	f := newFixture(t)
	c := f.mustOpen(t)
	code := f.shownCode()
	s := NewServer(c, "ac-01", "v1.2.3")

	state, err := s.GetSetupState(ctx, &setuppb.GetSetupStateRequest{})
	if err != nil {
		t.Fatal(err)
	}
	if state.Claimed || state.NodeName != "ac-01" || state.Version != "v1.2.3" {
		t.Errorf("GetSetupState = %v", state)
	}

	wrong := "2222-2222"
	if code == wrong {
		wrong = "3333-3333"
	}
	assertCode(t, "wrong code", s, wrong, codes.PermissionDenied)
	assertCode(t, "malformed code", s, "x", codes.PermissionDenied)
	assertCode(t, "right code", s, code, codes.Unimplemented)

	c.limiter = rate.NewLimiter(rate.Every(time.Hour), 0)
	assertCode(t, "throttled", s, wrong, codes.ResourceExhausted)

	f.addAccount(t, "admin")
	assertCode(t, "claimed", s, code, codes.FailedPrecondition)
	state, err = s.GetSetupState(ctx, &setuppb.GetSetupStateRequest{})
	if err != nil || !state.Claimed {
		t.Errorf("GetSetupState after claim = %v, %v", state, err)
	}
}

func assertCode(t *testing.T, name string, s *Server, code string, want codes.Code) {
	t.Helper()
	_, err := s.CompleteSetup(context.Background(), &setuppb.CompleteSetupRequest{SetupCode: code})
	if got := status.Code(err); got != want {
		t.Errorf("%s: CompleteSetup = %v, want %v", name, err, want)
	}
}
