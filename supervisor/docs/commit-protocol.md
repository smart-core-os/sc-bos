Supervisor Commit Protocol — application-confirmed updates
==========================================================

**TL;DR**: instead of the Supervisor inferring update success from a Podman health probe, BOS confirms
it explicitly by calling a `Commit` RPC over the Unix socket once it considers itself healthy on the new
version. Until a matching `Commit` arrives within a deadline, the new version is unconfirmed and the
Supervisor rolls back on its own. This is the application-layer form of the "commit-on-success" model
used by Android A/B and rpm-ostree, and it is the same on Rocky/Podman and FreeBSD.

Companion to [`state-model.md`](./state-model.md) (the state machine and crash-safety invariants this
refines) and [`../../cmd/supervisor/docs/supervisor-mechanism-rocky.md`](../../cmd/supervisor/docs/supervisor-mechanism-rocky.md)
(the tag-swap mechanism).

The shape
---------

Store the goal, derive the status by observing what is running, and make a single idempotent
`reconcile` the only operation. `Commit` is what feeds the two observations (`running()`, `healthy()`)
portably, in place of platform-specific host inspection.

```go
// The ONLY persisted state: what we want. Written once, atomically, before any work.
type Target struct{ Version, URL, SHA256 string }

// Status is DERIVED, never stored. running()/healthy() come from BOS's Commit, not host inspection.
func status(t *Target) State {
    if t == nil { return IDLE }
    switch {
    case running() == t.Version && healthy(): return COMPLETED
    case running() == t.Version:              return INSTALLING // applied, awaiting commit
    default:                                  return FAILED     // never took / rolled back
    }
}

// The ONLY operation. Every step is "ensure X" — a no-op if already true.
func reconcile(ctx, t Target) error {
    if !imageExists(t.Version)             { download(t); load(t) }      // idempotent: skip if present
    if resolve("current") != id(t.Version) { tag("current","previous"); tag(t.Version,"current") }
    if running() != resolve("current")     { restart() }                 // idempotent: skip if matched
    if err := awaitCommit(ctx, t.Version); err != nil {                  // was: confirmHealthy
        return reconcile(ctx, previousTarget())                          // roll toward previous
    }
    return nil
}
```

Startup and a fresh request are the same call: `reconcile(ctx, loadTarget())`. There is no separate
recovery path, and because the running version asserts itself via `Commit`, `COMPLETED(v)` cannot be
reported while running != v. The rest of this document specifies the `Commit` RPC those two observations
rely on.

Motivation
----------

The current confirm step polls `podman inspect --format '{{.State.Health.Status}}'` plus
`systemctl is-active`. That has four problems:

1. **It is Rocky/Podman only.** FreeBSD (rc.d + binary tarball) has no container health concept, so the
   `Installer.Confirm` step has no portable implementation. The confirm signal is bound to the platform.
2. **It is a weak signal.** A health probe (`/bos healthz` returning 200) reports health-of-process. It
   does not know whether config loaded, the database is reachable, listeners bound — the things that
   make the new version *actually* good. BOS knows all of this; the probe does not.
3. **It cannot tell which version answered.** `healthy()` checks health, not identity. As
   [`state-model.md`](./state-model.md) section 5 shows (crash points P2–P4, RB2), a Supervisor that
   confirms health alone can report `COMPLETED(v)` while the host is actually running the *old* version
   — a false success. Closing this needs a separate version-identity check (obligation 2/3 in that doc).
4. **It couples the Supervisor to the unit file.** The Supervisor depends on `HealthCmd` being present
   and correct in every `bos.container`, one more thing to standardise across sites.

Confirmation belongs to BOS. BOS is the authority on its own health, it knows its own version, and it
already speaks to the Supervisor over the socket. Moving the confirm signal there fixes all four.

The protocol
------------

### New RPC

```proto
// Commit is called by BOS on every startup, once it considers itself healthy, to tell the Supervisor
// which version is now running and good. The Supervisor uses it to confirm an in-progress update (or a
// rollback) and to learn the running version on platforms with no container health concept.
rpc Commit(CommitRequest) returns (CommitResponse);

message CommitRequest {
  // The version of BOS making the call. The Supervisor confirms the update only if this matches the
  // version it is awaiting.
  string version = 1;
}
message CommitResponse {}
```

### The one rule that makes it work

**BOS calls `Commit(its-own-version)` on every startup — not only after an update.** `Commit` does not
mean "the update worked"; it means "BOS version V is up and healthy." The Supervisor interprets it
relative to what it is currently expecting:

- **Awaiting commit for version X**, and `Commit(X)` arrives -> promote X to the committed/stable
  version; the update is `COMPLETED`.
- **Awaiting commit for X**, deadline expires with no `Commit(X)` -> roll back to `:previous`, restart;
  the old version boots and calls `Commit(old)`, confirming the rollback; the update is `FAILED`.
- **Idle** (no update in flight), `Commit(V)` arrives -> a routine heartbeat. The Supervisor records V
  as the running version and otherwise does nothing.

The every-boot rule is load-bearing: it is how the Supervisor confirms a *rollback* succeeded (the old
version commits on its way back up), and how the Supervisor learns `running()` without inspecting the
platform.

### Install sequence with Commit

```
BOS -> Supervisor : InstallUpdate(v, url, sha)          [existing]
Supervisor        : download, verify, load, tag-swap, restart bos   (state: awaiting-commit for v)
Supervisor        : start commit deadline (= commitDeadline config option)
new BOS boots     : self-check; when healthy -> Commit(v)  (with retry/backoff until deadline)
  Commit(v) before deadline   -> Supervisor commits v          -> COMPLETED
  deadline with no Commit(v)  -> Supervisor rolls back, restart -> old BOS Commit(old) -> FAILED
```

`Confirm` (poll health) is replaced by "await a matching `Commit` within the deadline." `Apply` and
`Rollback` are unchanged. The platform-specific `Installer` no longer needs a `Confirm`/`healthy()`
implementation at all — confirmation is platform-independent.

How this refines the state model
---------------------------------

This maps directly onto the desired-state framing recommended in
[`state-model.md`](./state-model.md), and tightens two invariants from being *enforced* to being
*structural*:

- **I3 (no optimistic lies) becomes free.** The caller of `Commit` *is* the running BOS and it states
  its own version, so the Supervisor can only commit a version that is genuinely running. There is no
  way to confirm the wrong version — the image-identity check (obligation 2/3 in `state-model.md`) is no
  longer needed, because identity is carried by the signal.
- **`running()` and `healthy()` stop being host observations.** They are reported by BOS over the
  socket, identically on Rocky and FreeBSD. The Supervisor needs no `podman inspect` / health-probe
  introspection for the confirm decision.

Persisted state (per the desired-state model): the goal `target = {version, url, sha}`, plus a
`committed` version and the awaiting-commit deadline. Derived status:

| Condition | Reported state |
|---|---|
| running == committed | `COMPLETED` (stable) |
| running == target, != committed, within deadline | `INSTALLING` (awaiting commit) |
| running == target, != committed, past deadline | roll back -> `FAILED` |
| no target | `IDLE` |

Crash safety carries over unchanged: a Supervisor restart re-derives the awaiting-commit state from the
persisted goal and either keeps waiting (BOS will re-`Commit` on its own next heartbeat) or rolls back
past the deadline. There is still no separate recovery path.

What BOS must do
----------------

1. On **every** startup, after its own readiness self-check passes, call `Commit(version)` over the
   Supervisor socket.
2. **Retry with backoff** until it succeeds or the Supervisor's deadline would have elapsed: a healthy
   BOS that fails to reach the socket would otherwise be rolled back (see Risks).
3. Define "ready enough to commit" — at minimum process up and listeners bound; ideally config applied
   and the datastore reachable. This policy is BOS's to own, which is the point: BOS is the authority.

Risks and trade-offs
--------------------

- **Healthy-but-unreachable rollback.** A genuinely healthy BOS that cannot reach the socket (mount
  missing, Supervisor restarting) misses its deadline and is rolled back unnecessarily. This is
  symmetric with today's "health probe flaps -> rollback" and arguably less likely over a local socket,
  but it is a real failure mode. Mitigations: BOS retry/backoff; the existing socket-*directory* mount
  (so the path survives the Supervisor recreating its socket); a deadline comfortably longer than BOS
  startup.
- **BOS gains a startup dependency on the Supervisor API.** The socket and client already exist for
  `InstallUpdate`, so this is incremental, but BOS now does meaningful work against the Supervisor on
  every boot. The `Commit` must be best-effort and non-fatal to BOS startup — BOS must come up and serve
  even if the Supervisor is briefly unavailable; only the *confirmation* waits.
- **Defining BOS readiness** is a new policy decision. Too strict and good updates get rolled back; too
  loose and a subtly-broken version commits itself. Start conservative (process + listeners) and tighten
  with evidence.
- **Supervisor self-update is still out of scope.** `Commit` confirms *BOS* updates. Updating the
  Supervisor binary needs its own confirm story regardless; it does not get one for free here.

Relationship to the SCC check-in
---------------------------------

`Commit` is the **local analog of the SCC check-in**. Both are BOS asserting its running version and
health; SCC uses it to advance the deployment lifecycle, the Supervisor uses it to decide commit vs.
rollback. The Supervisor needs its own copy precisely because rollback must be local and fast — it
cannot wait for a cloud round-trip. The two stay consistent because they carry the same fact (BOS's
self-assessed running version) from the same authority.

Recommendation
--------------

**Adopt the `Commit` RPC and retire the Podman-health confirm.** It is the single change that:

- makes the confirm step portable to FreeBSD (the stated requirement) with no platform introspection;
- upgrades the confirm signal from process-health to application-health;
- makes invariant I3 structural rather than a hand-written version check; and
- removes the Supervisor's dependency on the unit's `HealthCmd`.

Specifics:

1. Add `Commit(CommitRequest)` to `SupervisorApi`; carry the version. Keep request/response messages
   unique per RPC.
2. Replace the `Installer.Confirm` health poll with an await-matching-`Commit`-within-deadline step in
   the Service; the `commitDeadline` config option bounds the wait. Keep `Apply`/`Rollback` as they are. This
   collapses the platform-specific surface of `Installer` to just apply/rollback/restart.
3. Make BOS call `Commit` on every startup, best-effort with retry/backoff, after its readiness check.
4. Keep `HealthCmd` in `bos.container` for systemd's own restart and observability, but remove it from
   the Supervisor's decision path; drop the "standardise HealthCmd across sites" coupling from the
   mechanism doc.
5. Document the healthy-but-unreachable rollback as an accepted failure mode, mitigated by retry and the
   socket-directory mount.

Defer: the boot-counter / candidate-promotion variant (unit boots `:candidate`, a one-shot service
promotes it) — it would let a dead Supervisor leave the old version booting with zero action, but it
changes the unit's boot contract and is more invasive than warranted now. `Commit` gets most of the
benefit without touching how the unit boots.
