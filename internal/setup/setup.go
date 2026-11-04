// Package setup manages the setup code that lets an installer claim a fresh node, or a hub enrol it, while the node's
// access policy is on.
//
// Presenting the code proves access to the node's own filesystem: the code is only ever written to a file in its data
// directory, readable by the node's user alone, and never to its log, which the node serves to read-only callers.
// The code is accepted while the node is waiting to be set up, which means local accounts are enabled but none exist,
// there is no other way to log in, and the node is not enrolled with a hub. So nobody the node already lets in is
// waiting alongside the code. This is checked every time the code is used, so the code stops working as soon as the
// node is claimed or enrolled.
package setup

import (
	"context"
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/google/renameio/v2/maybe"
	"go.uber.org/zap"
	"golang.org/x/time/rate"

	"github.com/smart-core-os/sc-bos/internal/account"
	"github.com/smart-core-os/sc-bos/pkg/auth/token"
)

// Role is the system role granted to a caller presenting the setup code as a bearer token.
const Role = "setup"

const (
	stateFileName = "state.json"
	codeFileName  = "setup-code.txt"
)

var (
	ErrWrongCode       = errors.New("wrong setup code")
	ErrTooManyAttempts = errors.New("too many wrong setup codes, try again later")
	ErrNotPending      = errors.New("node is not waiting to be set up")
)

// Config configures Open.
type Config struct {
	// Dir holds the setup state, typically <data>/setup.
	Dir string
	// CodeFile, if set, supplies the setup code instead of generating one, for scripted installs.
	CodeFile string
	// Accounts is the local account store, nil if local accounts are disabled.
	// Without local accounts nothing can be claimed with a code, so no code is issued.
	Accounts *account.Store
	// OtherLogins is true if people or services can log in to the node without a local account, for example through
	// Keycloak or an identities file. Those logins are out of sight of the account store, so the node counts as
	// claimed and no code is issued.
	OtherLogins bool
	// Enrolled reports whether the node is enrolled with a hub.
	Enrolled func() bool
	Logger   *zap.Logger
}

// Codes checks setup codes presented to the node.
// The zero value is not usable; use Open.
type Codes struct {
	accounts    *account.Store
	otherLogins bool
	enrolled    func() bool
	logger      *zap.Logger

	// limiter is shared by every way of presenting a code, so callers can't multiply their guesses by spreading them
	// over different RPCs. It is node-wide rather than per client so that guessing from many addresses doesn't help.
	// The cost is that while someone keeps guessing faster than the limit refills, even the right code is refused,
	// so the installer and the hub are locked out until the guessing stops. The node is on a site LAN at this point,
	// and a lockout that ends with the flood was preferred to a per-client limit that many addresses could beat.
	limiter *rate.Limiter
	guessMu sync.Mutex

	hash []byte // nil if no code was issued; set by Open and never changed
}

// Open returns Codes for the node described by cfg.
// If the node is waiting to be set up, the setup code is loaded from cfg.Dir or cfg.CodeFile, or generated, and the
// log says where the installer can read it.
func Open(ctx context.Context, cfg Config) (*Codes, error) {
	if cfg.Logger == nil {
		cfg.Logger = zap.NewNop()
	}
	if cfg.Enrolled == nil {
		cfg.Enrolled = func() bool { return false }
	}
	c := &Codes{
		accounts:    cfg.Accounts,
		otherLogins: cfg.OtherLogins,
		enrolled:    cfg.Enrolled,
		logger:      cfg.Logger,
		limiter:     rate.NewLimiter(rate.Every(time.Second), 10),
	}

	pending, err := c.pending(ctx)
	if err != nil {
		return nil, err
	}
	if !pending {
		return c, nil
	}

	hash, readable, err := loadOrCreate(cfg.Dir, cfg.CodeFile)
	if err != nil {
		return nil, err
	}
	c.hash = hash

	switch {
	case cfg.CodeFile != "":
		c.logger.Info("This node is waiting to be set up. Use the setup code from the setup code file to claim it, or to enrol it with a hub.",
			zap.String("setupCodeFile", cfg.CodeFile))
	case readable:
		c.logger.Info("This node is waiting to be set up. Read its setup code from the setup code file to claim it, or to enrol it with a hub.",
			zap.String("setupCodeFile", filepath.Join(cfg.Dir, codeFileName)))
	default:
		// The installer was shown the code when it was made, and has since removed the code file.
		c.logger.Info("This node is waiting to be set up with the setup code it was given when installed. To issue a new code, delete the setup state file and restart.",
			zap.String("stateFile", filepath.Join(cfg.Dir, stateFileName)))
	}
	return c, nil
}

// Claimed reports whether the node has finished setup: it no longer accepts a setup code.
func (c *Codes) Claimed(ctx context.Context) (bool, error) {
	if c.hash == nil {
		return true, nil
	}
	pending, err := c.pending(ctx)
	return !pending, err
}

// Check returns nil if code is the node's setup code and the node is still waiting to be set up.
// Wrong codes count towards a node-wide limit on guesses; correct ones do not.
func (c *Codes) Check(ctx context.Context, code string) error {
	code, err := normaliseCode(code)
	if err != nil {
		return ErrWrongCode
	}
	return c.check(ctx, code)
}

func (c *Codes) check(ctx context.Context, code string) error {
	if c.hash == nil {
		return ErrNotPending
	}
	pending, err := c.pending(ctx)
	if err != nil {
		return err
	}
	if !pending {
		return ErrNotPending
	}

	// Only wrong guesses use up the limit, and rate.Limiter can't refund a token, so look before taking one.
	// Holding guessMu makes the look and the take one step; otherwise simultaneous guesses could all see the same
	// spare token. Comparing a hash is quick, so serialising guesses costs nothing noticeable.
	c.guessMu.Lock()
	defer c.guessMu.Unlock()
	if c.limiter.Tokens() < 1 {
		return ErrTooManyAttempts
	}
	if subtle.ConstantTimeCompare(hashCode(code), c.hash) != 1 {
		c.limiter.Allow()
		return ErrWrongCode
	}
	return nil
}

// ValidateAccessToken implements token.Validator, accepting the setup code as a bearer token.
// The claims carry only the setup role.
func (c *Codes) ValidateAccessToken(ctx context.Context, tkn string) (*token.Claims, error) {
	code, err := normaliseCode(tkn)
	if err != nil {
		// Most tokens are something else, a JWT for example. Don't count those as guesses.
		return nil, err
	}
	if err := c.check(ctx, code); err != nil {
		return nil, err
	}
	return &token.Claims{
		Subject:     "setup-code",
		Name:        "setup code",
		SystemRoles: []string{Role},
	}, nil
}

// pending reports whether the node is waiting to be set up, whether or not a code has been issued.
func (c *Codes) pending(ctx context.Context) (bool, error) {
	if c.accounts == nil || c.otherLogins || c.enrolled() {
		return false, nil
	}
	// Any account, whatever its roles, means someone has already been let in, and the code must not be waiting
	// within their reach: even a viewer enrolling the node into a hub they control would gain full control of it.
	var n int64
	err := c.accounts.Read(ctx, func(tx *account.Tx) error {
		var err error
		n, err = tx.CountAccounts(ctx)
		return err
	})
	if err != nil {
		return false, fmt.Errorf("count accounts: %w", err)
	}
	return n == 0, nil
}

type state struct {
	// CodeHash is the hex SHA-256 of the normalised setup code.
	CodeHash string `json:"codeHash"`
}

// loadOrCreate returns the hash of the setup code to accept, and whether the code can be read from the code file in
// dir.
//
// The code is kept across restarts, so the code an installer was shown stays valid when the node restarts.
// It is not consumed when used: it stops working while the node is claimed or enrolled, and works again if the node
// becomes unclaimed (unenrolled with no accounts), so a node can be re-enrolled after its hub is rebuilt.
// Only its hash is kept in the state file; the code file exists so the installer can read the code, and may be
// deleted once they have.
func loadOrCreate(dir, codeFile string) (hash []byte, readable bool, err error) {
	if codeFile != "" {
		raw, err := os.ReadFile(codeFile)
		if err != nil {
			return nil, false, fmt.Errorf("read setup code file: %w", err)
		}
		code, err := normaliseCode(strings.TrimSpace(string(raw)))
		if err != nil {
			return nil, false, fmt.Errorf("setup code file %s: want %d characters from %s", codeFile, codeLen, codeAlphabet)
		}
		hash = hashCode(code)
		if err := writeState(dir, hash); err != nil {
			return nil, false, err
		}
		return hash, false, nil
	}

	hash, err = readState(dir)
	switch {
	case errors.Is(err, fs.ErrNotExist):
	case err != nil:
		return nil, false, err
	default:
		return hash, codeFileMatches(dir, hash), nil
	}

	code, err := newCode()
	if err != nil {
		return nil, false, fmt.Errorf("generate setup code: %w", err)
	}
	hash = hashCode(code)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return nil, false, fmt.Errorf("create setup dir: %w", err)
	}
	if err := maybe.WriteFile(filepath.Join(dir, codeFileName), []byte(displayCode(code)+"\n"), 0o600); err != nil {
		return nil, false, fmt.Errorf("write setup code file: %w", err)
	}
	if err := writeState(dir, hash); err != nil {
		return nil, false, err
	}
	return hash, true, nil
}

func readState(dir string) ([]byte, error) {
	raw, err := os.ReadFile(filepath.Join(dir, stateFileName))
	if err != nil {
		return nil, err
	}
	var s state
	if err := json.Unmarshal(raw, &s); err != nil {
		return nil, fmt.Errorf("read setup state: %w", err)
	}
	hash, err := hex.DecodeString(s.CodeHash)
	if err != nil || len(hash) == 0 {
		return nil, fmt.Errorf("read setup state: invalid code hash")
	}
	return hash, nil
}

func writeState(dir string, hash []byte) error {
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return fmt.Errorf("create setup dir: %w", err)
	}
	raw, err := json.Marshal(state{CodeHash: hex.EncodeToString(hash)})
	if err != nil {
		return err
	}
	if err := maybe.WriteFile(filepath.Join(dir, stateFileName), raw, 0o600); err != nil {
		return fmt.Errorf("write setup state: %w", err)
	}
	return nil
}

// codeFileMatches reports whether the code file in dir is still there and holds the code with hash.
func codeFileMatches(dir string, hash []byte) bool {
	raw, err := os.ReadFile(filepath.Join(dir, codeFileName))
	if err != nil {
		return false
	}
	code, err := normaliseCode(strings.TrimSpace(string(raw)))
	return err == nil && subtle.ConstantTimeCompare(hashCode(code), hash) == 1
}
