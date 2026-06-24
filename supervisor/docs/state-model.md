Supervisor State Model — requirements, invariants, interruption safety
======================================================================

This document specifies **what the Supervisor must uphold** when applying a BOS update, independently of
how it is built. It is written as requirements, not as a particular code structure: the same model holds
for the current implementation (a persisted progress enum plus a Podman health poll) and for the
reconcile-to-desired-state design with an explicit BOS `Commit` (see
[`commit-protocol.md`](./commit-protocol.md)). Where the two differ, the difference is isolated to one
concept — *how confirmation is obtained* — and to the mapping table in section 9; the body is common to
both.

Companion docs: [`../../cmd/supervisor/docs/supervisor-mechanism-rocky.md`](../../cmd/supervisor/docs/supervisor-mechanism-rocky.md)
(the tag-swap mechanism), [`../../cmd/supervisor/docs/diagram-update-lifecycle.md`](../../cmd/supervisor/docs/diagram-update-lifecycle.md)
(the SCC-facing lifecycle this mirrors), and [`commit-protocol.md`](./commit-protocol.md) (the explicit
confirmation signal).

1. The two state spaces
-----------------------

The Supervisor straddles two state spaces, and correctness is the requirement that the first never lies
about the second — even across a crash.

### 1.1 Reported status `R`

What BOS observes via `GetUpdateStatus`. Each state has a fixed *meaning*; whether an implementation
**stores** it or **derives** it from the host is an implementation choice (section 9), not part of the
model.

| State | Meaning |
|---|---|
| `IDLE` | no update in progress or recorded |
| `DOWNLOADING` | fetching + verifying the artefact; the host has **not** been mutated |
| `INSTALLING` | the host is being or has been mutated; the target version is **not yet confirmed** |
| `COMPLETED` | the target version is running and confirmed good |
| `FAILED` | the target is not running; the system is on a healthy known-good version, or an explicit unrecoverable error is reported |

At every non-`IDLE` state the Supervisor knows the **target version `v`**.

### 1.2 Host state `H` (Rocky / Podman; FreeBSD analogously)

- Images `repo:<v>` (one per applied version) plus two moving tags the Supervisor owns:
  - `repo:current` — what the BOS unit resolves at (re)start.
  - `repo:previous` — the rollback pointer.
- The BOS service, recreated from `repo:current` on restart.

`running(H)` = the version the BOS service is actually running (on Rocky, the version whose image ID
matches the running container's; on FreeBSD, the installed binary's version). May be "unknown/none".

The FreeBSD platform has no container tags, but the model is the same: an applied version, a recorded
previous version, a `running(H)`, and a restart that brings up whatever is currently selected.

2. Confirmation — the one concept that varies
---------------------------------------------

`confirmed(v)` means: **the Supervisor holds evidence that `running(H) = v` and that `v` is functioning
correctly.** This is the hinge of the whole model (it gates `COMPLETED`), and it is the *only* place the
two implementations genuinely diverge. Confirmation can be obtained two ways:

- **Observed** — the Supervisor polls a platform health signal (current Rocky implementation: `podman
  inspect` health + `systemctl is-active`). A health signal reports *function* but **not identity**, so
  an observed implementation MUST independently establish `running(H) = v` as well. Health alone is
  insufficient (it is the cause of the false-success cases in section 7). Observation is also
  platform-bound: FreeBSD has no equivalent signal.
- **Asserted** — the running BOS calls `Commit(v)` over the socket once it considers itself healthy
  ([`commit-protocol.md`](./commit-protocol.md)). Identity is carried by the caller (the running version
  asserts its own version), and the signal is identical on Rocky and FreeBSD.

**Requirement C:** confirmation MUST establish *both* function *and* version identity. How (observe +
identity check, or assert) is the implementation's choice. Everything else in this document is written
in terms of `confirmed(v)` and does not care which mechanism produces it.

3. The host operations
----------------------

Applying an update is the ordered sequence (Rocky naming; FreeBSD has the analogous extract/select/restart):

```
D   download + verify artefact            (no host mutation)
L   apply artefact                        -> image repo:<v> exists
T1  record rollback pointer               (current -> previous; no-op on first install)
T2  select the new version                (version  -> current)   <- the moment :current changes
Rr  restart BOS                           (recreate from :current)
Cf  obtain confirmed(v)                    (observe health + identity, or await Commit) within a deadline
```

Rollback on `Cf` failure is the symmetric tail: select previous (`previous -> current`), restart,
confirm.

4. Invariants
-------------

These are requirements on any implementation.

- **I1 — Single-flight.** At most one update (including its crash-recovery) proceeds at a time; a request
  arriving while one is in flight is rejected (`FailedPrecondition`). Recovery must take this exclusion
  **before** the Supervisor begins serving requests.
- **I2 — Recoverable-before-mutate.** Before the first host mutation (`L`), the Supervisor must durably
  record enough to drive recovery after a crash — at minimum that an update to version `v` is underway.
  (Whether it also records how to re-obtain the artefact determines which recovery outcome is available;
  see the latitude note below.)
- **I3 — Truthful reporting (core).** `COMPLETED(v)` ⇒ `running(H) = v` and `confirmed(v)`. `FAILED` ⇒
  `running(H)` is a healthy known-good version, *or* an explicit unrecoverable error is reported. The
  Supervisor must **never** report `COMPLETED(v)` while `running(H) ≠ v`.
- **I4 — Valid rollback pointer.** Whenever `:current` has been moved to the new version (`T2` done),
  `:previous` resolves to the prior good version (`T1` ran first, or it was a first install — see I3's
  unrecoverable-error clause and example E6).
- **I5 — Convergent recovery.** Recovery is repeatable and converges: re-running it any number of times
  reaches the same terminal state, consistent with `H`. There is no crash point from which the system
  cannot be driven to a correct terminal state (modulo first-install, E6).
- **I6 — Atomic durable writes.** Persisted state is written atomically (e.g. temp file + rename); a torn
  write must never be read as a valid state. A missing/unreadable record degrades to `IDLE`.
- **I7 — Bounded resources.** Artefact size is capped before it is written; staging is cleaned up when the
  operation ends.

**Latitude note (recovery completeness).** On a crash *before the artefact is usable* (before/at `L`),
two recovery outcomes both satisfy I3 and I5:

1. **Abandon** to `FAILED` — the system stays on the old, healthy version. Requires nothing beyond I2's
   minimum.
2. **Resume** — re-fetch the artefact and complete the install. Requires the artefact source (URL +
   checksum) to have been persisted (a stronger form of I2).

Both are correct; neither lies. An implementation chooses based on how strongly it must honour "MUST
auto-recover". The current implementation abandons (it persists only the progress marker); the
reconcile design resumes (it persists the goal, URL included).

5. State transition requirements
---------------------------------

Allowed transitions and the guard each requires (an implementation may *store* these transitions or
*derive* them — the guard is what matters):

| To               | Required guard                                                                                  |
|------------------|-------------------------------------------------------------------------------------------------|
| `DOWNLOADING(v)` | a valid `InstallUpdate(v, url, sha)` accepted while not already in flight (I1)                  |
| `INSTALLING(v)`  | artefact downloaded + verified; **I2 record durable before any host mutation**                  |
| `COMPLETED(v)`   | `running(H) = v` **and** `confirmed(v)` (I3, requirement C)                                     |
| `FAILED`         | system on a healthy known-good version, or explicit unrecoverable error (I3)                    |
| (re-enter)       | after a crash, recovery resumes from the durable record and drives toward a terminal state (I5) |

The `running(H) = v` clause on `COMPLETED` is the operative part of I3. An asserted confirmation supplies
it inherently; an observed confirmation must add it explicitly.

6. Interruption points — exhaustive
------------------------------------

Crash points `P*` across the host operations of section 3. For each: the host state, `running(H)`, the
one thing that must **never** happen, and the outcomes permitted (all of which satisfy I3/I5). Where two
outcomes are listed, an implementation may choose (the latitude note in section 4); where one is listed,
it is required.

| Crash point                                                           | Host `H`                                       | `running(H)` | Must never                   | Permitted outcome(s)                                         |
|-----------------------------------------------------------------------|------------------------------------------------|--------------|------------------------------|--------------------------------------------------------------|
| **P1** during `D`                                                     | untouched                                      | old          | `COMPLETED(v)`               | `IDLE` (request dropped), or resume → `COMPLETED(v)`         |
| **P2** after I2 record, before `L`                                    | image absent, `:current`=old                   | old          | `COMPLETED(v)`               | `FAILED` (abandon), or resume → `COMPLETED(v)`               |
| **P3** during `L`                                                     | image absent (apply is atomic), `:current`=old | old          | `COMPLETED(v)`               | `FAILED`, or resume → `COMPLETED(v)`                         |
| **P4** after `T1`, before `T2`                                        | image present, `:previous`=old, `:current`=old | old          | `COMPLETED(v)`               | `FAILED`, or select+restart+confirm → `COMPLETED(v)`         |
| **P5** after `T2`, before `Rr`                                        | `:current`=v, running still old                | old          | `COMPLETED(v)` (running ≠ v) | restart → confirm → `COMPLETED(v)`, else rollback → `FAILED` |
| **P6** during/after `Rr`, before `Cf`                                 | `:current`=v, running=v                        | v            | —                            | confirm → `COMPLETED(v)`, else rollback → `FAILED`           |
| **P7** during `Cf`                                                    | `:current`=v, running=v, confirmation pending  | v            | —                            | confirm → `COMPLETED(v)`, else rollback → `FAILED`           |
| **RB1** rollback chosen, before selecting previous                    | `:current`=v(bad), running=v(bad)              | v            | `COMPLETED(v)`               | confirm fails → rollback → `FAILED`                          |
| **RB2** after selecting previous (`:current`=old), before its restart | `:current`=old, running still v(bad)           | v(bad)       | `COMPLETED(v)`               | restart → running=old → confirm → `FAILED`                   |
| **P8** after terminal write                                           | settled                                        | matches      | —                            | no-op                                                        |

Two things to read off this table:

- **The "must never" column is invariant I3 made concrete.** At P2–P4 and RB2 the running version is not
  `v`, so reporting `COMPLETED(v)` is a lie. An *observed* confirmation that checks health only — without
  the identity check required by requirement C — falls into exactly this trap (the old version is healthy
  and would be reported as the new one). An *asserted* `Commit(v)` cannot, because only the running
  version can send it. This is why requirement C is non-negotiable, and it is the current
  implementation's one outstanding gap.
- **The latitude rows (P1–P4) are where the two implementations legitimately differ** — abandon vs.
  resume — and both are correct.

7. Worked examples
------------------

**E1 — Happy path.** `IDLE → DOWNLOADING(v2) → INSTALLING(v2)` → `L,T1,T2,Rr` → `confirmed(v2)` →
`COMPLETED(v2)`. `:current=v2`, `:previous=v1`.

**E2 — New version unhealthy.** … `INSTALLING(v2)` → apply → `Cf` fails → select v1, restart, confirm →
`FAILED(v2, "update unhealthy: …")`. System runs v1, healthy.

**E3 — Power-cut during apply (P3) — the implementations diverge here.** Reboot. Image `v2` never
finished applying; `:current` still v1; `running=v1`. Both behaviours are correct:
- *Abandon* (current impl): no artefact source persisted → `FAILED(v2, "not applied")`; node keeps v1.
- *Resume* (reconcile impl): goal carries the URL → re-fetch, apply, restart, confirm → `COMPLETED(v2)`.
Neither reports `COMPLETED(v2)` while still on v1.

**E4 — Power-cut after selecting v2, before restart (P5).** Reboot. `:current=v2`, running still v1.
Recovery restarts → running=v2 → confirm → `COMPLETED(v2)`. Identical for both implementations.

**E5 — Power-cut mid-rollback after re-selecting v1 (RB2).** Reboot. `:current=v1`, was running v2(bad).
Recovery restarts → running=v1 → confirm ok but `running ≠ v2` → `FAILED(v2, …)`. Correct on both axes;
the `running ≠ v` guard is what prevents a false `COMPLETED(v2)`.

**E6 — First install ever, fails confirmation.** No `:previous`. `T1` is a no-op. `Cf` fails → rollback
finds no previous → `FAILED(v1, "no previous version to roll back to")`. The node may have no healthy BOS;
this is the single case where I3's "healthy known-good version" cannot hold, so the rule degrades to its
explicit-unrecoverable-error clause. Worth flagging in the deployment doc as an accepted limitation.

8. Confirmation, restated for FreeBSD
-------------------------------------

The Rocky implementation can obtain confirmation by observation because Podman exposes a health signal.
FreeBSD (rc.d + binary tarball) does not, so an observed implementation has no portable confirm step at
all. An *asserted* confirmation (`Commit`) is therefore not just a cleaner Rocky design but the only
uniform way to satisfy requirement C across both platforms. This is the primary reason to prefer the
explicit `Commit` signal; see [`commit-protocol.md`](./commit-protocol.md).

9. How each implementation satisfies the model
----------------------------------------------

The model above is common to both. This table is the only place the implementation strategies appear; it
is where coupling is deliberately quarantined.

**The Supervisor now implements the right-hand column** (reconcile-to-desired-state + asserted `Commit`,
per [`commit-protocol.md`](./commit-protocol.md)): the persisted goal `{version, url, sha}` is the only
durable state, recovery re-runs the single reconcile, and confirmation is BOS calling `Commit(v)`. The
left-hand column documents the prior design for contrast; its outstanding I3 identity gap (item 3 below)
is discharged structurally by the new one.

| Concern                              | Current: progress-log + observed health                                       | New: reconcile-to-desired-state + asserted `Commit`                                 |
|--------------------------------------|-------------------------------------------------------------------------------|-------------------------------------------------------------------------------------|
| State representation                 | a stored progress enum; the in-flight state is persisted as a recovery marker | derived from the persisted goal + observed host; the goal is the only durable state |
| I2 record written before `L`         | `INSTALLING(v)` marker                                                        | the goal `{version, url, sha}`                                                      |
| Recovery (I5)                        | a dedicated recovery routine that resumes the marker                          | re-run the single idempotent reconcile against the goal                             |
| Confirmation (req. C)                | observe Podman health **and** must add the `running = v` identity check       | BOS asserts `Commit(v)`; identity carried by the caller (structural)                |
| Crash before artefact usable (P1–P4) | abandon → `FAILED` (no URL persisted)                                         | resume: re-fetch → `COMPLETED` (URL in goal)                                        |
| `running(H)` / health source         | host inspection — Rocky-only                                                  | reported by `Commit` — portable to FreeBSD                                          |
| I3 at P2–P4 / RB2                    | enforced by a hand-written identity check (currently the gap)                 | structural — only the running version can `Commit`                                  |

10. Obligations checklist
-------------------------

Independent of implementation, any build must:

1. Take the single-flight exclusion at startup, before serving (I1).
2. Durably record the in-flight update before mutating the host (I2); decide and document whether
   recovery abandons or resumes (section 4 latitude).
3. Gate `COMPLETED(v)` on both function and identity (`confirmed(v)` ∧ `running(H)=v`) (I3, req. C) — in
   the normal path and in recovery alike.
4. Keep `:previous` a valid rollback target whenever `:current` has moved (I4).
5. Make recovery idempotent and convergent; treat a missing/torn record as `IDLE` (I5, I6).
6. Bound the download and clean up staging (I7).
7. Bound how long shutdown waits for an in-flight update to reach a persisted terminal state, with the
   service unit's stop timeout ≥ that bound.

Of these, item 3's identity check was the prior design's outstanding gap; the implemented `Commit` design
discharges it structurally rather than by an added check.
