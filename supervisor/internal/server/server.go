// Package server implements the SupervisorApi gRPC service.
//
// The Supervisor stores the goal of an update - the target version and where to fetch it - and drives a
// single idempotent reconcile toward it: download, apply, then await BOS asserting the new version is
// healthy via Commit. Reported status is derived from the goal plus the last Commit, never stored as a
// separate progress enum. Startup recovery and a fresh InstallUpdate are the same reconcile, so there is
// no separate recovery path. See supervisor/docs/commit-protocol.md and state-model.md.
package server

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"sync"
	"time"

	"github.com/google/renameio/v2/maybe"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/smart-core-os/sc-bos/pkg/proto/supervisorpb"
	"github.com/smart-core-os/sc-bos/supervisor/internal/install"
	"github.com/smart-core-os/sc-bos/supervisor/internal/selfupdate"
)

// stateFileName holds the durable update record (the goal plus the last committed version) under the
// staging dir, so the in-flight goal or last outcome survives a Supervisor restart.
const stateFileName = "state.json"

// tagPattern matches a valid OCI image tag, per the reference grammar.
var tagPattern = regexp.MustCompile(`^[a-zA-Z0-9_][a-zA-Z0-9._-]{0,127}$`)

// target is the goal of an update: the version to run and where to fetch its artefact. It is the only
// thing the Supervisor needs to persist before mutating the host, and it carries the URL/sha so recovery
// can resume (re-fetch) rather than abandon.
type target struct {
	Version string `json:"version"`
	URL     string `json:"url"`
	SHA256  string `json:"sha256"`
}

// record is the durable state. The goal (Target) drives reconcile and recovery; Committed is the last
// version BOS asserted running+healthy (the rollback baseline); FinishTime marks the goal's reconcile as
// settled so a restart does not re-drive a finished update. Reported status is derived from this.
type record struct {
	Target     *target    `json:"target,omitempty"`
	Committed  string     `json:"committed,omitempty"`
	StartTime  *time.Time `json:"startTime,omitempty"`
	FinishTime *time.Time `json:"finishTime,omitempty"`
	Error      string     `json:"error,omitempty"`
}

// phase is the in-memory progress of an active reconcile, used only to distinguish DOWNLOADING from
// INSTALLING while it runs. It is not persisted: after a crash, recovery re-runs the reconcile.
type phase int

const (
	phaseIdle phase = iota
	phaseDownloading
	phaseInstalling
	phaseRollingBack
)

// Service implements supervisorpb.SupervisorApiServer.
type Service struct {
	supervisorpb.UnimplementedSupervisorApiServer

	installer              install.Installer
	stagingDir             string
	stateFile              string
	httpClient             *http.Client
	commitDeadline         time.Duration
	allowInsecureDownloads bool
	selfUpdate             SelfUpdater // nil when supervisor self-update is not configured
	logger                 *zap.Logger

	wg sync.WaitGroup // tracks the in-flight reconcile goroutine for shutdown

	mu          sync.Mutex
	rec         record
	phase       phase
	reconciling bool          // single-flight: at most one reconcile in flight
	lastCommit  string        // version of the most recent Commit
	commitGen   uint64        // bumped on each Commit; lets awaitCommit ignore stale commits
	commitCh    chan struct{} // lazily created; closed (and cleared) on each Commit to wake awaitCommit
}

// New returns a Service that installs updates with the given installer, keeping durable state under
// stateDir: the update record at stateDir/state.json and downloaded artefacts under stateDir/staging
// (created if absent). httpClient downloads artefacts (nil uses http.DefaultClient); commitDeadline
// bounds how long reconcile waits for BOS to Commit the new version before rolling back; logger may be
// nil.
//
// If a durable record from a previous run is present it is loaded, so GetUpdateStatus reports the last
// outcome (and Reconcile can resume an interrupted goal) rather than IDLE.
//
// allowInsecureDownloads permits non-HTTPS artefact URLs; it is a development-only escape hatch and
// defaults to false in production.
func New(installer install.Installer, stateDir string, httpClient *http.Client, commitDeadline time.Duration, allowInsecureDownloads bool, logger *zap.Logger) *Service {
	if logger == nil {
		logger = zap.NewNop()
	}
	s := &Service{
		installer:              installer,
		stagingDir:             filepath.Join(stateDir, "staging"),
		stateFile:              filepath.Join(stateDir, stateFileName),
		httpClient:             httpClient,
		commitDeadline:         commitDeadline,
		allowInsecureDownloads: allowInsecureDownloads,
		logger:                 logger,
	}
	// Ensure the staging dir exists (this also creates stateDir) so downloads and the state file can be
	// written. A failure here is logged, not fatal: it surfaces later as a download/persist error.
	if err := os.MkdirAll(s.stagingDir, 0o700); err != nil {
		s.logger.Warn("create staging dir", zap.Error(err))
	}
	s.loadState()
	return s
}

func (s *Service) InstallUpdate(_ context.Context, req *supervisorpb.InstallUpdateRequest) (*supervisorpb.InstallUpdateResponse, error) {
	if req.GetVersion() == "" || req.GetDownloadUrl() == "" || req.GetSha256() == "" {
		return nil, status.Error(codes.InvalidArgument, "version, download_url and sha256 are required")
	}
	// version becomes an image tag; reject anything that isn't a valid OCI tag at the boundary, so a
	// malformed request fails loudly rather than retargeting the repo via a confusing podman error.
	if !tagPattern.MatchString(req.GetVersion()) {
		return nil, status.Error(codes.InvalidArgument, "version is not a valid image tag")
	}

	s.mu.Lock()
	if s.reconciling {
		s.mu.Unlock()
		return nil, status.Error(codes.FailedPrecondition, "an update is already in progress")
	}
	now := time.Now()
	// Replace the goal, keeping the last committed version as the rollback baseline. Persist it before
	// any host mutation (I2); the URL/sha it carries let recovery resume by re-fetching.
	s.rec = record{
		Target:    &target{Version: req.GetVersion(), URL: req.GetDownloadUrl(), SHA256: req.GetSha256()},
		Committed: s.rec.Committed,
		StartTime: &now,
	}
	s.beginReconcileLocked()
	t := *s.rec.Target
	snapshot := s.rec
	accepted := s.deriveStatusLocked()
	s.mu.Unlock()

	s.persistState(snapshot)
	s.startReconcile(t)
	return &supervisorpb.InstallUpdateResponse{Status: accepted}, nil
}

// startReconcile spawns the single reconcile goroutine, detached from any request context: applying an
// update recreates the BOS container, severing the caller's connection. The WaitGroup lets shutdown wait
// for it to reach a terminal state.
func (s *Service) startReconcile(t target) {
	s.wg.Add(1)
	go s.reconcile(t)
}

// reconcile is the single idempotent operation that drives the host to the goal: download + verify the
// artefact, apply it, then await BOS asserting the new version via Commit. On a missing commit within
// the deadline it rolls back to the previous version. A fresh InstallUpdate and startup recovery both
// run it.
func (s *Service) reconcile(t target) {
	defer s.wg.Done()
	defer s.endReconcile()

	ctx := context.Background()
	s.mu.Lock()
	sinceGen := s.commitGen // only a commit after this point confirms the new version
	s.mu.Unlock()

	s.setPhase(phaseDownloading)
	artefact, err := install.DownloadAndVerify(ctx, s.httpClient, t.URL, t.SHA256, s.stagingDir, install.MaxArtefactBytes, s.allowInsecureDownloads)
	if err != nil {
		s.settleFailed(fmt.Sprintf("download: %v", err))
		return
	}
	defer func() { _ = os.Remove(artefact) }()

	s.setPhase(phaseInstalling)
	if err := s.installer.Apply(ctx, artefact, t.Version); err != nil {
		s.settleFailed(fmt.Sprintf("apply: %v", err))
		return
	}

	if s.awaitCommit(ctx, t.Version, sinceGen) {
		s.settleCompleted(t.Version)
		return
	}

	s.rollback(ctx, fmt.Sprintf("update %s not confirmed within %s", t.Version, s.commitDeadline))
}

// rollback returns BOS to the previous version after the new one failed to commit, then waits for that
// previous version to assert itself before settling FAILED. Awaiting a fresh commit (newer than sinceGen)
// is what stops a terminal FAILED being reported while the bad version is still the one running (RB2).
func (s *Service) rollback(ctx context.Context, reason string) {
	s.setPhase(phaseRollingBack)
	s.logger.Warn("update not confirmed, rolling back", zap.String("reason", reason))

	s.mu.Lock()
	sinceGen := s.commitGen
	previous := s.rec.Committed
	s.mu.Unlock()

	if err := s.installer.Rollback(ctx); err != nil {
		s.settleFailed(fmt.Sprintf("%s; rollback failed: %v", reason, err))
		return
	}
	if previous != "" && !s.awaitCommit(ctx, previous, sinceGen) {
		s.settleFailed(fmt.Sprintf("%s; rolled back but %s not confirmed within %s", reason, previous, s.commitDeadline))
		return
	}
	s.settleFailed(reason)
}

// awaitCommit blocks until BOS Commits version with a generation newer than sinceGen, the commit
// deadline elapses, or ctx is done. The generation guard ignores a stale pre-update commit, so a
// rollback waits for a genuinely fresh post-rollback commit (the RB2 guarantee).
func (s *Service) awaitCommit(ctx context.Context, version string, sinceGen uint64) bool {
	timer := time.NewTimer(s.commitDeadline)
	defer timer.Stop()
	for {
		s.mu.Lock()
		confirmed := s.lastCommit == version && s.commitGen > sinceGen
		ch := s.commitChLocked()
		s.mu.Unlock()
		if confirmed {
			return true
		}
		select {
		case <-ch:
		case <-timer.C:
			return false
		case <-ctx.Done():
			return false
		}
	}
}

// Commit is BOS asserting that version is now running and healthy. The Supervisor uses it to confirm an
// in-progress update or rollback (an awaiting reconcile is woken via awaitCommit), and to learn the
// running version. With no reconcile in flight it is a routine heartbeat that records version as the
// running, good version - the rollback baseline for the next update. See commit-protocol.md.
func (s *Service) Commit(_ context.Context, req *supervisorpb.CommitRequest) (*supervisorpb.CommitResponse, error) {
	if req.GetVersion() == "" {
		return nil, status.Error(codes.InvalidArgument, "version is required")
	}

	s.mu.Lock()
	s.noteCommitLocked(req.GetVersion())
	var snapshot *record
	if !s.reconciling && s.rec.Committed != req.GetVersion() {
		// Heartbeat outside an update: record the running version as the rollback baseline. (A reconcile
		// in flight owns Committed itself: it promotes on success, after awaitCommit returns.)
		s.rec.Committed = req.GetVersion()
		cp := s.rec
		snapshot = &cp
	}
	s.mu.Unlock()

	if snapshot != nil {
		s.persistState(*snapshot)
	}
	return &supervisorpb.CommitResponse{}, nil
}

// noteCommitLocked records the latest commit and wakes any awaitCommit waiter. The generation counter
// lets a waiter tell a fresh commit from a stale one (e.g. a rollback must wait for a post-rollback
// commit, not the pre-update one). Caller holds s.mu.
func (s *Service) noteCommitLocked(version string) {
	s.lastCommit = version
	s.commitGen++
	close(s.commitChLocked())
	s.commitCh = nil // a later waiter (or commit) lazily recreates it
}

// commitChLocked returns the broadcast channel awaitCommit waits on, creating it on demand. It is made
// lazily rather than in New so its whole lifecycle stays within the caller's goroutine context (which
// matters under testing/synctest, where a channel created outside the test bubble is not durable).
// Caller holds s.mu.
func (s *Service) commitChLocked() chan struct{} {
	if s.commitCh == nil {
		s.commitCh = make(chan struct{})
	}
	return s.commitCh
}

// Reconcile drives the persisted goal to completion. It runs at startup (recovery) and is the same
// operation a fresh InstallUpdate spawns - there is no separate recovery path. A goal interrupted by a
// crash (no finish time) is resumed: the artefact is re-fetched and re-applied (Apply is idempotent),
// then BOS is awaited as usual, rolling back on timeout. A settled goal, or no goal, is a no-op.
//
// It takes the single-flight exclusion synchronously, so a recovery in flight is in effect before the
// Supervisor serves requests. Call once after New, before serving.
func (s *Service) Reconcile() {
	s.mu.Lock()
	if s.reconciling || s.rec.Target == nil || s.rec.FinishTime != nil {
		s.mu.Unlock()
		return
	}
	t := *s.rec.Target
	s.beginReconcileLocked()
	s.mu.Unlock()

	s.logger.Warn("resuming an update interrupted by restart", zap.String("version", t.Version))
	s.startReconcile(t)
}

// Wait blocks until any in-flight reconcile finishes, or ctx is done. It returns ctx.Err() if the wait
// is cut short, letting shutdown bound how long it waits for a reconcile to reach a terminal state.
func (s *Service) Wait(ctx context.Context) error {
	done := make(chan struct{})
	go func() {
		s.wg.Wait()
		close(done)
	}()
	select {
	case <-done:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (s *Service) GetUpdateStatus(_ context.Context, _ *supervisorpb.GetUpdateStatusRequest) (*supervisorpb.GetUpdateStatusResponse, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return &supervisorpb.GetUpdateStatusResponse{Status: s.deriveStatusLocked()}, nil
}

// SelfUpdater is the supervisor self-update entry point the SupervisorApi delegates to. It is satisfied
// by *selfupdate.Updater.
type SelfUpdater interface {
	// Install records the intent to self-update to version and launches the out-of-process applier.
	Install(ctx context.Context, version, url, sha256 string) (selfupdate.State, error)
	// Version is the running Supervisor's own version.
	Version() string
	// LastUpdate returns the most recent self-update state, or false if none.
	LastUpdate() (selfupdate.State, bool)
}

// SetSelfUpdater wires the supervisor self-update entry point. Until set, the self-update RPCs report
// the feature as unconfigured. Call before serving.
func (s *Service) SetSelfUpdater(u SelfUpdater) { s.selfUpdate = u }

func (s *Service) InstallSupervisorUpdate(ctx context.Context, req *supervisorpb.InstallSupervisorUpdateRequest) (*supervisorpb.InstallSupervisorUpdateResponse, error) {
	if s.selfUpdate == nil {
		return nil, status.Error(codes.Unimplemented, "supervisor self-update is not configured")
	}
	if req.GetVersion() == "" || req.GetDownloadUrl() == "" || req.GetSha256() == "" {
		return nil, status.Error(codes.InvalidArgument, "version, download_url and sha256 are required")
	}
	// version names the RPM's last-known-good file; reject anything that isn't a safe tag/filename.
	if !tagPattern.MatchString(req.GetVersion()) {
		return nil, status.Error(codes.InvalidArgument, "version is not a valid version tag")
	}
	st, err := s.selfUpdate.Install(ctx, req.GetVersion(), req.GetDownloadUrl(), req.GetSha256())
	if errors.Is(err, selfupdate.ErrInProgress) {
		return nil, status.Error(codes.FailedPrecondition, err.Error())
	}
	if err != nil {
		return nil, status.Errorf(codes.Internal, "begin self-update: %v", err)
	}
	return &supervisorpb.InstallSupervisorUpdateResponse{Status: selfUpdateStatus(st)}, nil
}

func (s *Service) GetSupervisorInfo(_ context.Context, _ *supervisorpb.GetSupervisorInfoRequest) (*supervisorpb.GetSupervisorInfoResponse, error) {
	if s.selfUpdate == nil {
		return &supervisorpb.GetSupervisorInfoResponse{
			SelfUpdate: &supervisorpb.UpdateStatus{State: supervisorpb.UpdateStatus_IDLE},
		}, nil
	}
	resp := &supervisorpb.GetSupervisorInfoResponse{Version: s.selfUpdate.Version()}
	if st, ok := s.selfUpdate.LastUpdate(); ok {
		resp.SelfUpdate = selfUpdateStatus(st)
	} else {
		resp.SelfUpdate = &supervisorpb.UpdateStatus{State: supervisorpb.UpdateStatus_IDLE}
	}
	return resp, nil
}

// selfUpdateStatus maps a selfupdate.State to the wire UpdateStatus BOS relays to Smart Core Connect.
func selfUpdateStatus(st selfupdate.State) *supervisorpb.UpdateStatus {
	out := &supervisorpb.UpdateStatus{Version: st.Target}
	if !st.StartTime.IsZero() {
		out.StartTime = timestamppb.New(st.StartTime)
	}
	if st.FinishTime != nil {
		out.FinishTime = timestamppb.New(*st.FinishTime)
	}
	switch st.Phase {
	case selfupdate.PhaseCompleted:
		out.State = supervisorpb.UpdateStatus_COMPLETED
	case selfupdate.PhaseFailed:
		out.State = supervisorpb.UpdateStatus_FAILED
		out.Error = st.Error
	case selfupdate.PhaseInstalling, selfupdate.PhaseRollingBack:
		out.State = supervisorpb.UpdateStatus_INSTALLING
	default:
		out.State = supervisorpb.UpdateStatus_IDLE
	}
	return out
}

// beginReconcileLocked marks a reconcile in flight (single-flight) and sets its initial phase. Caller
// holds s.mu.
func (s *Service) beginReconcileLocked() {
	s.reconciling = true
	s.phase = phaseDownloading
}

// endReconcile clears the single-flight flag once the reconcile goroutine returns.
func (s *Service) endReconcile() {
	s.mu.Lock()
	s.reconciling = false
	s.mu.Unlock()
}

func (s *Service) setPhase(p phase) {
	s.mu.Lock()
	s.phase = p
	s.mu.Unlock()
}

// settleCompleted promotes version to the committed (stable) version and marks the goal settled, so the
// update is reported COMPLETED and a restart does not re-drive it.
func (s *Service) settleCompleted(version string) {
	now := time.Now()
	s.mu.Lock()
	s.rec.Committed = version
	s.rec.FinishTime = &now
	s.rec.Error = ""
	s.phase = phaseIdle
	snapshot := s.rec
	s.mu.Unlock()
	s.persistState(snapshot)
}

// settleFailed marks the goal settled without promoting it, so the update is reported FAILED with reason
// and a restart does not re-drive it.
func (s *Service) settleFailed(reason string) {
	now := time.Now()
	s.mu.Lock()
	s.rec.FinishTime = &now
	s.rec.Error = reason
	s.phase = phaseIdle
	snapshot := s.rec
	s.mu.Unlock()
	s.persistState(snapshot)
}

// deriveStatusLocked computes the reported UpdateStatus from the durable record plus the in-memory
// reconcile phase. There is no stored status enum: COMPLETED means the goal version has been committed,
// FAILED means the goal settled on a different version. Caller holds s.mu.
func (s *Service) deriveStatusLocked() *supervisorpb.UpdateStatus {
	rec := s.rec
	if rec.Target == nil {
		return &supervisorpb.UpdateStatus{State: supervisorpb.UpdateStatus_IDLE}
	}
	st := &supervisorpb.UpdateStatus{Version: rec.Target.Version, Error: rec.Error}
	if rec.StartTime != nil {
		st.StartTime = timestamppb.New(*rec.StartTime)
	}
	if rec.FinishTime != nil {
		st.FinishTime = timestamppb.New(*rec.FinishTime)
	}
	switch {
	case rec.FinishTime != nil && rec.Committed == rec.Target.Version:
		st.State = supervisorpb.UpdateStatus_COMPLETED
		st.Error = ""
	case rec.FinishTime != nil:
		st.State = supervisorpb.UpdateStatus_FAILED
	case s.phase == phaseInstalling || s.phase == phaseRollingBack:
		st.State = supervisorpb.UpdateStatus_INSTALLING
	default:
		st.State = supervisorpb.UpdateStatus_DOWNLOADING
	}
	return st
}

// persistState atomically writes the record to the state file (temp + fsync + rename). Failures are
// logged, not fatal: a lost record degrades to IDLE after a restart, which is recoverable - BOS
// re-Commits its running version on its next heartbeat.
func (s *Service) persistState(rec record) {
	data, err := json.Marshal(rec)
	if err != nil {
		s.logger.Warn("marshal state for persistence", zap.Error(err))
		return
	}
	if err := maybe.WriteFile(s.stateFile, data, 0o600); err != nil {
		s.logger.Warn("write state file", zap.Error(err))
	}
}

// loadState reads a previously persisted record, if any, into s.rec. A missing file is normal (no prior
// update); a malformed one is logged and ignored (degrades to IDLE, per I6).
func (s *Service) loadState() {
	data, err := os.ReadFile(s.stateFile)
	if err != nil {
		if !os.IsNotExist(err) {
			s.logger.Warn("read state file", zap.Error(err))
		}
		return
	}
	var rec record
	if err := json.Unmarshal(data, &rec); err != nil {
		s.logger.Warn("parse state file", zap.Error(err))
		return
	}
	s.rec = rec
}
