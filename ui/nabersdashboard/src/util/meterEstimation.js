/**
 * Gap filling for cumulative meter readings.
 *
 * Every energy figure in this dashboard is the difference between two point
 * readings of a monotonically increasing kWh accumulator. When a meter stops
 * reporting, the reading at a month boundary is simply absent and the whole
 * chain — month, end use, rating — becomes unknown. One dead distribution board
 * out of seventeen is enough to blank the headline figure.
 *
 * The NABERS method permits estimating missing data provided the estimation is
 * disclosed, so this module recovers the missing boundary values and reports
 * exactly how much of each figure was estimated.
 *
 * **A gap is not the same as missing data.** `pkg/auto/history` writes a record
 * only when the reading changes — `meter.go` gates every write behind a deduper,
 * and nothing writes on a timer regardless of change. For a BACnet meter, whose
 * driver never sets `end_time` and applies its own equivalence check on top, an
 * idle meter therefore produces no records at all, indefinitely.
 *
 * So for a cumulative accumulator recorded on change, absence of records is
 * positive evidence of *no consumption*, not evidence of absence. Treating every
 * gap as missing data and projecting a rate across it fabricates energy: an idle
 * meter with a month-long silence was measured at +22.7 kWh when the truth was
 * zero. The rules that follow exist to keep that from happening.
 *
 * - A gap **bounded** by a real reading on each side resolves to the *earlier*
 *   reading, carried forward. This follows directly from on-change recording
 *   rather than being a convenience. `collectChanges` polls on a fixed schedule
 *   — every five minutes on the sites this serves — and writes only when the
 *   value differs, so a stretch with no records is a stretch in which every poll read
 *   the last written value. The accumulator therefore *held* that value until the
 *   poll that produced the later record, and the boundary's value is known, not
 *   estimated. The residual error is bounded by one poll interval, being the
 *   window in which the step could have landed.
 *
 *   Linear interpolation used to live here and was wrong in a way worth naming:
 *   it spread the later record's step backwards across the whole bracket, so a
 *   board that consumed nothing for three weeks and then 400 kWh in an hour was
 *   reported as consuming steadily throughout — and both adjacent months were
 *   badged as estimated when neither figure was ever uncertain. The same
 *   carry-forward reading is what `mepc-3cs`'s month-end report has always
 *   applied to the same records, so the two artefacts now agree.
 * - A gap **open at the later end** is only an outage if the meter is actually
 *   unreachable, which the caller establishes with a live trait read. A
 *   reachable meter that has simply gone quiet is idle: carry the last value
 *   forward, report `actual`, disclose nothing. Only when `opts.unreachable` is
 *   set is the rate projected forward, inflated by `extrapolationUpliftPct` so a
 *   substituted value cannot flatter the rating. This is the one branch that
 *   still substitutes a value nobody measured, and so the only one `enabled`
 *   governs.
 * - A gap **open at the earlier end** is refused outright. There is no bounded
 *   interval to lean on and no liveness signal for the past, so "idle" and
 *   "offline" are indistinguishable and the two answers differ by hundreds of
 *   kWh. The caller names the meter instead, which is actionable.
 * - A register that goes **backwards** is graded by how far it fell, as a share of
 *   what it fell from. A reset loses everything it had counted, so consumption
 *   across the affected span is unrecoverable and stays unknown. A small
 *   correction — a re-registration, a stale write, a read landing mid-update —
 *   loses only its own magnitude, so the span reports zero and discloses the step.
 *   See {@link DEFAULT_REGRESSION_SHARE_PCT}, which also records why that
 *   understates and why it is nonetheless the better of the two errors here.
 *
 * **Where carry-forward can still mislead.** It assumes the recorder was running.
 * If the poll was failing, the automation was down, or the records existed and
 * were later trimmed by a retention policy, then silence carries no information
 * and the earlier month is understated while the later one is overstated by the
 * same amount. Two things bound that: consecutive months share a boundary, so the
 * error cancels in any total spanning both — a 12-month rating figure telescopes
 * to `last − first` and cannot move — and the stretch is still reported through
 * `quietFrom`/`quietTo` as unrecorded history, so it stays visible as a
 * commissioning fault even though it is no longer disclosed as an estimate.
 *
 * Nothing here does I/O. {@link module:util/meterBoundaries} fetches the
 * samples and establishes reachability; this module is the maths, so it can be
 * tested exhaustively.
 *
 * @module util/meterEstimation
 */

/** A gap no wider than this is normal reporting jitter, not missing data. */
export const DEFAULT_GAP_THRESHOLD_HOURS = 3;

/** How far either side of an instant to hunt for a bracketing reading. */
export const DEFAULT_SEARCH_WINDOW_DAYS = 45;

/** How much to inflate an extrapolated span, so it cannot flatter the rating. */
export const DEFAULT_EXTRAPOLATION_UPLIFT_PCT = 10;

/**
 * Below this share of an interval's energy, estimation is not worth disclosing.
 *
 * An idle meter that ticks once after a month of silence leaves a bracket a
 * single resolution unit wide. Interpolating across it is accurate to that one
 * tick, so labelling the month "estimated" would be crying wolf — and on a site
 * with a car park and exterior lighting meter it would never stop. The exact
 * figure is still reported; only the badge is suppressed.
 */
export const DEFAULT_MATERIAL_SHARE_PCT = 0.5;

/**
 * Accumulator movement across a bracket below which there is nothing to estimate.
 *
 * This is a stronger statement than a tolerance. The accumulator is monotonic, so
 * if it reads the same at both ends of a bracket its value at *every* instant in
 * between is that same value — known exactly, with no interpolation and no
 * uncertainty, however many weeks separate the two records. Allowing a small
 * movement rather than insisting on zero extends that to a meter that ticked once
 * or twice: the value inside is then known to within this many kWh.
 *
 * Elapsed time is irrelevant to it, which is the point. Reported from site: two
 * records twenty days apart both reading 1030.000 — a device restart writing a
 * fresh record of an unchanged value — were shown as a twenty-day gap. Meters on
 * lightly used circuits change very infrequently and must not read as faulty.
 */
export const DEFAULT_IDLE_TOLERANCE_KWH = 1;

/**
 * How many consecutive readings each probe returns, for measuring resolution.
 *
 * Costs no extra queries — the boundary reader already issues one query per
 * probed instant, and this only widens its page.
 */
export const DEFAULT_TICK_SAMPLE_COUNT = 5;

/**
 * A backwards step no larger than this share of the earlier reading is a
 * correction, not a reset.
 *
 * The accumulator is supposed to be monotonic, so any decrease means something
 * happened to the register rather than to the building. Two very different things
 * look the same in two point readings:
 *
 * - A **reset**, where the register returned to zero or near it and started again.
 *   Everything it counted before is gone, so consumption across the affected span
 *   really is unrecoverable — and it could be the meter's whole lifetime, hundreds
 *   of thousands of kWh. Guessing here would move the rating by more than the
 *   rating's own resolution.
 * - A **correction**, where the register moved back a little: a re-registration
 *   after a device swap, a driver writing a stale value, a protocol read landing
 *   mid-update. What was lost is bounded by the size of the step, which is small.
 *
 * Scaling by the earlier reading is what separates them, and it separates them by
 * orders of magnitude rather than by a fine judgement: a reset is ~100% of the
 * register, and the corrections seen at 3CS are a fraction of one percent. Elapsed
 * time is deliberately not part of the test — a register that fell by 150 kWh fell
 * by 150 kWh whether it did so over an hour or a month.
 *
 * Below the allowance the span's consumption is reported as zero and **disclosed as
 * estimated**, carrying the size of the step as the energy at stake. Above it the
 * figure is still withheld and the board still named.
 */
export const DEFAULT_REGRESSION_SHARE_PCT = 1;

/**
 * Absolute floor under {@link DEFAULT_REGRESSION_SHARE_PCT}, in kWh.
 *
 * A share of a small register is a very small number — 1% of a pump that has
 * totalled 3 kWh is 0.03 — and a sub-kWh wobble on such a meter is the same
 * ordinary correction it would be anywhere else. Mirrors
 * {@link DEFAULT_IDLE_TOLERANCE_KWH}, for the same reason: proportional tests need
 * a floor or they turn brittle exactly where the quantities are smallest.
 */
export const DEFAULT_REGRESSION_TOLERANCE_KWH = 1;

/**
 * How many consecutive bad reads still count as one dropout.
 *
 * See {@link withoutDropouts}. Two, because the fault observed at 3CS is a single
 * record and a second is cheap insurance, while a genuine reset needs far more
 * samples than this to climb back to where it was and so cannot be mistaken for one.
 * Raising it much past a handful starts to erode that separation.
 */
export const DEFAULT_MAX_DROPOUT_SAMPLES = 2;

/**
 * Ceiling on a measured resolution, as a share of the meter's own movement.
 *
 * Insurance against a measurement landing too high. If a meter that consumed
 * little over the period somehow reports a coarse quantum, an uncapped threshold
 * could dismiss a real outage as idle; this keeps the threshold proportionate to
 * what the meter actually did.
 */
export const DEFAULT_TICK_CAP_SHARE_PCT = 2;

const HOUR_MS = 60 * 60 * 1000;

/**
 * @typedef {{usage: number, at: Date}} MeterSample
 */

/**
 * How a boundary value was arrived at.
 *
 * `actual` covers an exact hit, a bracket narrow enough to be ordinary reporting
 * jitter, and a value carried forward across a stretch with no records — in every
 * case the accumulator's value here follows from what was recorded rather than
 * being substituted for it. There is deliberately no `interpolated`: bracketed
 * boundaries are carried forward, so nothing produces one. See the module doc.
 *
 * @typedef {'actual'|'extrapolated'|'unknown'} BoundaryQuality
 */

/**
 * A resolved accumulator value at one instant.
 *
 * Two different stretches are tracked, and keeping them apart is the point:
 *
 * `gapFrom`/`gapTo` bound a stretch whose value had to be **substituted** — today
 * only a forward projection past an unreachable meter's last reading. `gapKwh` is
 * how far the accumulator moved across it, and is what the estimated-energy
 * arithmetic is built from; using elapsed time instead would charge an idle month
 * with a share of its neighbours' energy. These feed the NABERS disclosure, so
 * they must be set only where a figure genuinely rests on a substitution.
 *
 * `quietFrom`/`quietTo` bound a stretch with **no recorded history**, whether or
 * not anything was substituted across it. A carried-forward boundary sets these
 * and not the gap pair: its value is known, but the missing records are still a
 * commissioning fault worth surfacing. These feed the data-quality view only.
 *
 * All four are null when there is nothing to report.
 *
 * @typedef {{
 *   usage: number|null,
 *   quality: BoundaryQuality,
 *   gapFrom: Date|null,
 *   gapTo: Date|null,
 *   gapKwh: number,
 *   quietFrom: Date|null,
 *   quietTo: Date|null,
 *   reason: string|null
 * }} ResolvedBoundary
 */

/**
 * Which mechanism produced a span's estimated energy.
 *
 * This has to travel with the figure, because the two mechanisms err in opposite
 * directions and a disclosure that names the wrong one is worse than a vague
 * one:
 *
 * - `projected` — a rate carried past the last reading of an unreachable meter,
 *   deliberately inflated by `extrapolationUpliftPct`. It **overstates**, which is
 *   the direction the NABERS method requires of a substituted value.
 * - `regressed` — a register that stepped backwards by less than its allowance,
 *   so the span reports zero and discloses the size of the step. It
 *   **understates** by whatever the meter really consumed, which is the direction
 *   the method forbids. See {@link DEFAULT_REGRESSION_SHARE_PCT} for why it is
 *   nonetheless the better of the two errors on an indicative dashboard.
 * - `mixed` — both, somewhere in the pool being summed.
 *
 * Null when nothing was estimated. Every published disclosure — the banner, the
 * gauge caveat, the monthly report and the CSV preamble — used to assert
 * `projected` unconditionally, so a month withheld only by a register correction
 * told its reader the figure had been conservatively inflated when it had been
 * floored at zero.
 *
 * @typedef {'projected'|'regressed'|'mixed'|null} EstimationKind
 */

/**
 * Consumption between two resolved boundaries, with its provenance.
 *
 * `estimatedHours`/`estimatedKwh` describe what the reported total rests on, and
 * `estimatedKind` says by what mechanism; `unrecordedHours` is how much of the
 * span had no recorded history at all, which is a data-quality figure and can be
 * large even when the total is exact. See {@link spanDelta}.
 *
 * @typedef {{
 *   kwh: number|null,
 *   estimated: boolean,
 *   estimatedHours: number,
 *   estimatedKwh: number,
 *   estimatedKind: EstimationKind,
 *   unrecordedHours: number,
 *   longestGap: {from: Date, to: Date, hours: number}|null,
 *   reason: string|null
 * }} BoundaryDelta
 */

/**
 * Combine the mechanisms seen across a pool into one.
 *
 * @param {Array<EstimationKind>} kinds
 * @return {EstimationKind}
 */
export function mergeEstimationKinds(kinds) {
  const seen = new Set((kinds ?? []).filter(Boolean));
  if (seen.size === 0) return null;
  if (seen.size > 1) return 'mixed';
  const [only] = seen;
  return only;
}

/**
 * Estimation settings, with this module's defaults filled in.
 *
 * @param {Object} [cfg]
 * @param {boolean} [cfg.enabled]
 * @param {number} [cfg.gapThresholdHours]
 * @param {number} [cfg.searchWindowDays]
 * @param {number} [cfg.extrapolationUpliftPct]
 * @param {number} [cfg.materialSharePct]
 * @param {number} [cfg.idleToleranceKwh]
 * @param {number} [cfg.tickSampleCount]
 * @param {number} [cfg.tickCapSharePct]
 * @param {number} [cfg.regressionSharePct]
 * @param {number} [cfg.regressionToleranceKwh]
 * @param {number} [cfg.maxDropoutSamples]
 * @return {{enabled: boolean, gapThresholdHours: number, searchWindowDays: number,
 *   extrapolationUpliftPct: number, materialSharePct: number, idleToleranceKwh: number,
 *   tickSampleCount: number, tickCapSharePct: number, regressionSharePct: number,
 *   regressionToleranceKwh: number, maxDropoutSamples: number}}
 */
export function estimationOptions(cfg) {
  const c = cfg ?? {};
  // `??` not `||` throughout: a configured 0 uplift means "interpolate but never
  // inflate", which is a real choice, and `||` would silently restore the 10%.
  return {
    enabled:                c.enabled ?? true,
    gapThresholdHours:      c.gapThresholdHours ?? DEFAULT_GAP_THRESHOLD_HOURS,
    searchWindowDays:       c.searchWindowDays ?? DEFAULT_SEARCH_WINDOW_DAYS,
    extrapolationUpliftPct: c.extrapolationUpliftPct ?? DEFAULT_EXTRAPOLATION_UPLIFT_PCT,
    materialSharePct:       c.materialSharePct ?? DEFAULT_MATERIAL_SHARE_PCT,
    idleToleranceKwh:       c.idleToleranceKwh ?? DEFAULT_IDLE_TOLERANCE_KWH,
    tickSampleCount:        c.tickSampleCount ?? DEFAULT_TICK_SAMPLE_COUNT,
    tickCapSharePct:        c.tickCapSharePct ?? DEFAULT_TICK_CAP_SHARE_PCT,
    regressionSharePct:     c.regressionSharePct ?? DEFAULT_REGRESSION_SHARE_PCT,
    regressionToleranceKwh: c.regressionToleranceKwh ?? DEFAULT_REGRESSION_TOLERANCE_KWH,
    maxDropoutSamples:      c.maxDropoutSamples ?? DEFAULT_MAX_DROPOUT_SAMPLES
  };
}

/**
 * The largest backwards step in the register that is still a correction rather
 * than a reset, given the reading it fell from.
 *
 * See {@link DEFAULT_REGRESSION_SHARE_PCT} for why the test is proportional to the
 * earlier reading and not to elapsed time.
 *
 * @param {number} fromKwh the earlier, higher reading
 * @param {Object} [opts] as built by {@link estimationOptions}
 * @return {number}
 */
export function regressionAllowanceKwh(fromKwh, opts) {
  const floor = opts?.regressionToleranceKwh ?? DEFAULT_REGRESSION_TOLERANCE_KWH;
  const pct   = opts?.regressionSharePct ?? DEFAULT_REGRESSION_SHARE_PCT;
  const share = (fromKwh > 0 ? fromKwh : 0) * (pct / 100);
  return Math.max(floor, share);
}

/**
 * The highest value a historical reading could legitimately hold.
 *
 * The accumulator only ever climbs, so the reading taken *now* is a ceiling on
 * every reading that came before it. That single fact is enough to identify a
 * whole class of corrupt records — a driver or protocol fault spitting out
 * 469780064 between a 14848 and an 18025 — without needing to know anything about
 * the meter or when the fault happened.
 *
 * The live trait read is preferred, being the current value by definition. Failing
 * that, the most recent reading held is the best available stand-in; note that if
 * that one is itself the corrupt reading then the ceiling is useless and nothing
 * gets filtered, which degrades to the old behaviour rather than making anything
 * worse.
 *
 * @param {MeterSample[]} samples ascending by time
 * @param {MeterSample|null} [liveSample] the current reading, if one was obtained
 * @return {number|null} null when there is nothing to compare against
 */
export function plausibleCeiling(samples, liveSample) {
  if (liveSample && liveSample.usage >= 0) return liveSample.usage;
  for (let i = (samples?.length ?? 0) - 1; i >= 0; i--) {
    if (samples[i].usage >= 0) return samples[i].usage;
  }
  return null;
}

/**
 * Drop readings a cumulative accumulator could not have produced.
 *
 * Two rules, both from the accumulator's own nature rather than from any judgement
 * about the meter:
 *
 * - **Negative** is impossible. The observed values cluster near int32's floor —
 *   -2147465600 against a -2147483648 minimum — so they are an integer overflow
 *   escaping as a signed value, not a reading.
 * - **Above the current value** is impossible, since the accumulator only climbs.
 *
 * Left in place these corrupt one thing badly beyond the obvious. {@link meanRate}
 * sums positive steps, so a single 469780064 spike contributes a step of nearly
 * half a billion kWh and inflates the meter's rate — and therefore any projection
 * made from it — beyond all sense.
 *
 * A genuine accumulator reset also puts historical readings above the current one,
 * and those get discarded too. The outcome is the same either way: with nothing
 * usable before the period start the meter is reported as unreadable and named,
 * exactly as a detected reset already caused. The caller notes the discard count
 * in the reason so the two remain distinguishable.
 *
 * @param {MeterSample[]} samples
 * @param {number|null} ceilingKwh as from {@link plausibleCeiling}
 * @return {{samples: MeterSample[], rejected: number}}
 */
export function plausibleSamples(samples, ceilingKwh) {
  const kept = [];
  let rejected = 0;
  for (const s of samples ?? []) {
    if (s.usage < 0 || (ceilingKwh !== null && s.usage > ceilingKwh)) {
      rejected++;
      continue;
    }
    kept.push(s);
  }
  return {samples: kept, rejected};
}

/**
 * Drop a **dropout**: a reading that dips below the register and is immediately
 * recovered.
 *
 * {@link plausibleSamples} catches a reading that is negative or above the current
 * value. It cannot catch the commonest corruption of all, because the value is
 * neither: a driver or protocol fault returning a bare **0** in the middle of an
 * otherwise healthy series. That is just as impossible as the others — a cumulative
 * register that has reached 16,768 kWh cannot read 0 and then read 18,560 — and left
 * in place it is far more damaging, because it does not look like corruption. It
 * looks like a reset, so the month around it is withheld and the board is reported as
 * faulty when the meter is fine and one record is not.
 *
 * **What separates a dropout from a genuine reset is the recovery, not the size.** A
 * dropout is followed at once by a reading back at or above where the register was: to
 * be real, the meter would have had to consume its entire lifetime total again in the
 * interval, which is a statement about physics rather than a threshold. A reset does
 * not recover — the register climbs from near zero over weeks, so the reading after it
 * is small. That is why the test is "does the very next reading return to the previous
 * level", bounded to `maxDropoutSamples` consecutive bad reads: a reset needs many
 * more samples than that to climb back, so it survives and is still reported.
 *
 * This is a *filter*, not an estimate. With the impossible reading gone the boundary
 * either side is a real recorded value and the month is **exact** — nothing is
 * substituted, nothing needs disclosing, and no NABERS trade is involved. The count
 * is still reported through `rejectedReadings` and named in the boundary reason,
 * because silently discarding data is not something a rating should do without saying
 * so.
 *
 * @param {MeterSample[]} samples ascending by time, as from {@link normaliseSamples}
 * @param {Object} [opts] as built by {@link estimationOptions}
 * @return {{samples: MeterSample[], rejected: number}}
 */
export function withoutDropouts(samples, opts) {
  const maxRun = opts?.maxDropoutSamples ?? DEFAULT_MAX_DROPOUT_SAMPLES;
  if (!samples?.length || maxRun < 1) return {samples: samples ?? [], rejected: 0};

  const kept = [];
  let rejected = 0;
  let i = 0;
  while (i < samples.length) {
    const prev = kept.length ? kept[kept.length - 1] : null;
    // The first sample has nothing to dip below, so it is taken on trust. A leading
    // dropout is indistinguishable from a meter whose history simply starts low.
    if (!prev || samples[i].usage >= prev.usage) {
      kept.push(samples[i]);
      i++;
      continue;
    }
    // A dip. Walk forward over consecutive readings that are all below `prev`, no
    // further than one dropout's worth, and see whether the register comes back.
    let j = i;
    while (j < samples.length && samples[j].usage < prev.usage && (j - i) < maxRun) j++;
    if (j < samples.length && j > i && samples[j].usage >= prev.usage) {
      rejected += j - i;
      i = j;
      continue;
    }
    // It never came back within reach, so this is the new baseline of a reset rather
    // than a bad read. Keep it, and let the reset be reported as one.
    kept.push(samples[i]);
    i++;
  }
  return {samples: kept, rejected};
}

/**
 * The meter's effective resolution: the smallest positive step it has been seen
 * to take.
 *
 * Under on-change recording every stored record is one tick of the accumulator —
 * or a small multiple, where a poll interval spanned several — so the smallest
 * difference between adjacent records is the quantum below which the meter cannot
 * report movement at all. That makes it the right threshold for "did this meter
 * move", and it differs per device: some tick every 1 kWh, some every 16, and a
 * single configured number cannot serve both.
 *
 * Deltas are taken **only within a run** of consecutive records. Two samples from
 * different probes may straddle a stretch of missing history, and that difference
 * is the missing energy, not a tick.
 *
 * The minimum is taken across *every* run rather than the latest one alone. A run
 * captured while the meter was busy shows several ticks per record and would
 * overstate the quantum; runs come from instants spread across the whole period,
 * so the smallest of them lands on the true one.
 *
 * Non-positive steps contribute nothing, so a duplicate record written after a
 * device restart and an accumulator reset are both ignored.
 *
 * @param {Array<MeterSample[]>} runs
 * @return {number|null} null when no positive step was observed
 */
export function observedTickKwh(runs) {
  let smallest = null;
  for (const run of runs ?? []) {
    for (let i = 1; i < (run?.length ?? 0); i++) {
      const step = run[i].usage - run[i - 1].usage;
      if (step > 0 && (smallest === null || step < smallest)) smallest = step;
    }
  }
  return smallest;
}

/**
 * The idle threshold for one meter: its measured resolution, capped.
 *
 * `spanKwh` is how far the meter moved across all the history we hold for it, and
 * bounds the threshold to a share of that. The cap can never pull the threshold
 * below `idleToleranceKwh`, which matters for the case this whole mechanism
 * exists to protect: a meter that never moved has a `spanKwh` of 0, and a cap
 * allowed to reach 0 would call its flat readings a gap again.
 *
 * @param {Object} args
 * @param {number|null} args.observedTick as measured by {@link observedTickKwh}
 * @param {number} args.spanKwh
 * @param {Object} args.opts as built by {@link estimationOptions}
 * @return {number}
 */
export function idleToleranceFor({observedTick, spanKwh, opts}) {
  const fallback = opts?.idleToleranceKwh ?? DEFAULT_IDLE_TOLERANCE_KWH;
  if (observedTick === null || !(observedTick > 0)) return fallback;
  const capPct = opts?.tickCapSharePct ?? DEFAULT_TICK_CAP_SHARE_PCT;
  const cap = Math.max((spanKwh > 0 ? spanKwh : 0) * (capPct / 100), fallback);
  return Math.min(observedTick, cap);
}

/**
 * Sort by time and drop duplicate instants.
 *
 * The boundary reader probes overlapping windows, so the same record commonly
 * comes back for several instants. Duplicates would contribute a zero-length
 * span to {@link meanRate} and are pure noise everywhere else.
 *
 * @param {Array<MeterSample|null>} samples
 * @return {MeterSample[]}
 */
export function normaliseSamples(samples) {
  const byTime = new Map();
  for (const s of samples) {
    if (!s || s.usage == null || Number.isNaN(s.usage) || !(s.at instanceof Date)) continue;
    const t = s.at.getTime();
    if (Number.isNaN(t)) continue;
    byTime.set(t, {usage: s.usage, at: s.at});
  }
  return [...byTime.entries()].sort((a, b) => a[0] - b[0]).map(([, s]) => s);
}

/**
 * The meter's mean consumption rate, in kWh per millisecond.
 *
 * Summed over positive pairwise steps rather than taken end-to-end, so an
 * accumulator reset somewhere in the middle does not produce a negative or
 * absurdly small rate. Returns null when there is nothing to measure a rate
 * over — one sample, or no elapsed time — because a fabricated rate is worse
 * than an admitted unknown.
 *
 * @param {MeterSample[]} samples sorted ascending, as from {@link normaliseSamples}
 * @return {number|null}
 */
export function meanRate(samples) {
  if (!samples || samples.length < 2) return null;
  const spanMs = samples[samples.length - 1].at.getTime() - samples[0].at.getTime();
  if (spanMs <= 0) return null;
  let consumed = 0;
  for (let i = 1; i < samples.length; i++) {
    consumed += Math.max(0, samples[i].usage - samples[i - 1].usage);
  }
  return consumed / spanMs;
}

/**
 * An options object rather than six positional arguments: there are now two
 * distinct stretches a boundary can carry, and `boundary(v, q, null, null, 0, r)`
 * gave no clue which null was which.
 *
 * @param {number|null} usage
 * @param {BoundaryQuality} quality
 * @param {Object} [extra]
 * @param {Date|null} [extra.gapFrom]
 * @param {Date|null} [extra.gapTo]
 * @param {number} [extra.gapKwh]
 * @param {Date|null} [extra.quietFrom]
 * @param {Date|null} [extra.quietTo]
 * @param {string|null} [extra.reason]
 * @return {ResolvedBoundary}
 */
function boundary(usage, quality, extra = {}) {
  return {
    usage,
    quality,
    gapFrom:   extra.gapFrom ?? null,
    gapTo:     extra.gapTo ?? null,
    gapKwh:    extra.gapKwh ?? 0,
    quietFrom: extra.quietFrom ?? null,
    quietTo:   extra.quietTo ?? null,
    reason:    extra.reason ?? null
  };
}

/**
 * @param {string} reason
 * @return {ResolvedBoundary}
 */
function unknownBoundary(reason) {
  return boundary(null, 'unknown', {reason});
}

/**
 * A kWh figure short enough to sit inside a reason string.
 *
 * @param {number} v
 * @return {string}
 */
const fmtKwh = (v) => (Math.abs(v) >= 100 ? Math.round(v) : Number(v.toFixed(1))).toLocaleString();

/**
 * Why a backwards step was refused, with the three numbers that decide it.
 *
 * The bare reason was not actionable. "accumulator reset near boundary" against a
 * meter whose entire register is 3 kWh looks identical to the same words against one
 * that fell 24,000, and the two want opposite responses: the first is a threshold to
 * widen, the second a meter to replace. Whether tolerating a step is negligible or
 * material cannot be judged without its size, and reading it off the raw history is
 * hours of work per board.
 *
 * @param {string} what
 * @param {number} fromKwh the reading it fell from
 * @param {number} droppedKwh how far it fell
 * @param {Object} [opts] as built by {@link estimationOptions}
 * @return {string}
 */
function regressionReason(what, fromKwh, droppedKwh, opts) {
  return `${what} (fell ${fmtKwh(droppedKwh)} kWh from ${fmtKwh(fromKwh)}; ` +
    `allowance ${fmtKwh(regressionAllowanceKwh(fromKwh, opts))} kWh)`;
}

/**
 * The accumulator value at `at`, carried forward or extrapolated from `samples`.
 *
 * @param {MeterSample[]} samples sorted ascending, as from {@link normaliseSamples}
 * @param {Date} at
 * @param {Object} opts as built by {@link estimationOptions}
 * @param {boolean} opts.enabled
 * @param {number} opts.gapThresholdHours
 * @param {number} opts.searchWindowDays also caps how far a projection may reach
 * @param {number} opts.extrapolationUpliftPct
 * @param {boolean} [opts.unreachable] the meter did not answer a live read, so a
 *   trailing silence is an outage rather than an idle meter
 * @return {ResolvedBoundary}
 */
export function resolveBoundary(samples, at, opts) {
  const t = at.getTime();
  const gapMs = opts.gapThresholdHours * HOUR_MS;

  let before = null;
  let after = null;
  for (const s of samples) {
    const st = s.at.getTime();
    if (st <= t) before = s;
    if (st >= t) {
      after = s;
      break;
    }
  }

  // Exact hit. Nothing to estimate, and no rounding to introduce.
  if (before && after && before.at.getTime() === after.at.getTime()) {
    return boundary(before.usage, 'actual');
  }

  if (before && after) {
    // The accumulator went backwards across this bracket, so something happened to
    // the register inside it. A step large enough to be a reset takes the meter's
    // history with it: where it happened and how much ran beforehand cannot be
    // recovered from two points, so that stays unknown rather than a clamped zero.
    //
    // A step small enough to be a correction does not. The boundary is resolved the
    // way every other bracket is — the earlier reading, carried forward — which is
    // still the best available value for this instant and keeps the month *before*
    // the correction exact. The lost energy is bounded by the size of the step and
    // is disclosed by `spanDelta` against the span that actually contains it, so
    // one register wobble no longer withholds the two months either side of it.
    if (after.usage < before.usage) {
      const droppedKwh = before.usage - after.usage;
      if (!opts.enabled || droppedKwh > regressionAllowanceKwh(before.usage, opts)) {
        return unknownBoundary(
          regressionReason('accumulator reset near boundary', before.usage, droppedKwh, opts));
      }
      return boundary(before.usage, 'actual', {quietFrom: before.at, quietTo: after.at});
    }
    const movedKwh = after.usage - before.usage;

    // Carry the earlier reading forward. History records only changes, so no
    // record between these two means every poll in between read `before.usage`,
    // and that is this instant's value — measured, not estimated. See the module
    // doc for why this replaced linear interpolation, and for what it assumes.
    //
    // Two tests decide only whether the bracket is worth reporting as *unrecorded*
    // history; neither affects the value any more.
    //
    // The first is movement. The accumulator is monotonic, so a bracket it barely
    // crossed pins its value here to within `idleToleranceKwh` — exactly, if it did
    // not move at all — however far apart the two records are. Two records twenty
    // days apart both reading 1030.000 describe a meter that consumed nothing, not
    // twenty days of missing data, and must not be reported as a gap.
    //
    // The second is proximity, judged by how close the *nearest* real reading is
    // rather than by how wide the bracket is. The sample pool is sparse on purpose
    // — roughly one reading per probed instant — so consecutive pooled readings sit
    // about a month apart however healthy the meter is. Testing bracket width
    // instead flagged every interior boundary of a perfectly continuous meter as a
    // month-long gap: 21,000 records at fifteen-minute spacing reported "31 days,
    // 30 Jun–31 Jul", and the denser the meter reported the worse it looked.
    const nearestMs = Math.min(t - before.at.getTime(), after.at.getTime() - t);
    const quiet = nearestMs > gapMs &&
      movedKwh > (opts.idleToleranceKwh ?? DEFAULT_IDLE_TOLERANCE_KWH);

    // Deliberately not gated on `opts.enabled`. That flag withholds figures that
    // rest on a substituted value, and this one does not — turning it off must not
    // turn a measurement into an unknown.
    return boundary(before.usage, 'actual',
      quiet ? {quietFrom: before.at, quietTo: after.at} : {});
  }

  if (!before && !after) return unknownBoundary('no readings near boundary');

  if (!before) {
    // Nothing at or before this instant. The opening accumulator is genuinely
    // unknown: a meter idle since before the period and a meter whose records
    // start late look identical here, and the two answers differ by hundreds of
    // kWh. Refusing names the board, which someone can act on; guessing quietly
    // moves the rating. Backward extrapolation used to live here and was wrong.
    return unknownBoundary('no reading at or before this point');
  }

  // Trailing silence. Whether that is an outage or an idle meter is not knowable
  // from the history alone — `pkg/auto/history` does not record unchanged values
  // — so it turns on the caller's live read.
  const elapsed = t - before.at.getTime();
  if (elapsed <= gapMs) return boundary(before.usage, 'actual');

  if (!opts.unreachable) {
    // The meter answered a live read, so it is present and simply has not
    // changed. Its accumulator held its last value: consumption across the
    // silence is zero, which is a measurement, not an estimate.
    return boundary(before.usage, 'actual');
  }

  if (!opts.enabled) {
    return unknownBoundary('meter unreachable, estimation disabled');
  }

  // Extrapolation has an explicit reach. Projecting a rate across months of
  // silence is fabrication rather than estimation, so past the search window the
  // boundary stays honestly unknown and the caller names the board.
  //
  // This used to fall out of the probe windows by accident — no sample within
  // `searchWindowDays` meant no rate to project with. Sampling the period's
  // interior removed that side effect, so the rule is stated here instead, where
  // it can be read and tested.
  if (elapsed > opts.searchWindowDays * 24 * HOUR_MS) {
    return unknownBoundary('unreachable for longer than the estimation window');
  }

  const rate = meanRate(samples);
  if (rate === null) {
    return unknownBoundary('too little history to estimate');
  }
  // Projecting forward past the last reading of an unreachable meter. A *higher*
  // value enlarges the interval that ends at this boundary, which is the interval
  // being reported, so upward is the conservative direction.
  const uplifted = rate * elapsed * (1 + opts.extrapolationUpliftPct / 100);
  // Even projected, a movement this small is not worth disclosing: the value is
  // pinned to within the same tolerance as an unmoved accumulator.
  if (uplifted <= (opts.idleToleranceKwh ?? DEFAULT_IDLE_TOLERANCE_KWH)) {
    return boundary(before.usage + uplifted, 'actual');
  }
  // Both stretches, because a projection is a substitution *and* a stretch with no
  // recorded history: it is disclosed as estimated and reported as unrecorded.
  return boundary(before.usage + uplifted, 'extrapolated', {
    gapFrom: before.at, gapTo: at, gapKwh: uplifted,
    quietFrom: before.at, quietTo: at
  });
}

/**
 * Total length of `intervals` clipped to `[from, to]`, in hours, counting any
 * overlap once.
 *
 * @param {Array<{from: Date, to: Date}>} intervals
 * @param {Date} from
 * @param {Date} to
 * @return {number}
 */
export function overlapHours(intervals, from, to) {
  const lo = from.getTime();
  const hi = to.getTime();
  const clipped = intervals
    .map(iv => [Math.max(lo, iv.from.getTime()), Math.min(hi, iv.to.getTime())])
    .filter(([a, b]) => b > a)
    .sort((a, b) => a[0] - b[0]);

  let total = 0;
  let cursor = -Infinity;
  for (const [a, b] of clipped) {
    const start = Math.max(a, cursor);
    if (b > start) {
      total += b - start;
      cursor = b;
    }
  }
  return total / HOUR_MS;
}

/**
 * The widest single stretch in `gaps`, clipped to `[from, to]`.
 *
 * The *total* unrecorded time answers "how much of this period has no history",
 * which is truthful but not actionable: a meter reporting every couple of weeks
 * adds up to most of the period in small increments and looks alarming. The
 * longest single stretch is the one an engineer can go and investigate, and
 * carrying its dates is what lets the UI say *when* rather than just how long.
 *
 * @param {Array<{from: Date, to: Date}>} gaps
 * @param {Date} from
 * @param {Date} to
 * @return {{from: Date, to: Date, hours: number}|null}
 */
export function widestGap(gaps, from, to) {
  const lo = from.getTime();
  const hi = to.getTime();
  let best = null;
  for (const g of gaps) {
    const a = Math.max(lo, g.from.getTime());
    const b = Math.min(hi, g.to.getTime());
    if (b <= a) continue;
    const hours = (b - a) / HOUR_MS;
    if (!best || hours > best.hours) best = {from: new Date(a), to: new Date(b), hours};
  }
  return best;
}

/**
 * The substituted stretch a resolved boundary rests on, if any, with the energy
 * that actually crossed it. Feeds the disclosure.
 *
 * @param {ResolvedBoundary} b
 * @return {Array<{from: Date, to: Date, kwh: number}>}
 */
function gapOf(b) {
  return (b?.gapFrom && b?.gapTo) ? [{from: b.gapFrom, to: b.gapTo, kwh: b.gapKwh ?? 0}] : [];
}

/**
 * The stretch of missing history a resolved boundary sits in, if any. Feeds the
 * data-quality view, and carries no energy: a carried-forward boundary is exact,
 * so what is reported here is the absence of records, not an uncertainty.
 *
 * Falls back to the substituted extent, because every substituted stretch is by
 * definition also an unrecorded one, and a caller that sets only the gap pair
 * should still have it counted here.
 *
 * @param {ResolvedBoundary} b
 * @return {Array<{from: Date, to: Date}>}
 */
function quietOf(b) {
  if (b?.quietFrom && b?.quietTo) return [{from: b.quietFrom, to: b.quietTo}];
  return (b?.gapFrom && b?.gapTo) ? [{from: b.gapFrom, to: b.gapTo}] : [];
}

/**
 * Energy attributable to estimated data inside `[from, to]`.
 *
 * Each gap contributes its own movement, apportioned by how much of it falls in
 * the interval — *not* a share of the interval's total energy. That distinction
 * is the whole point: a month half spent in an idle meter's silence and half
 * spent consuming 400 kWh has nothing estimated about the 400, because the
 * silence is known to have carried no energy. Charging it half of 400 was the
 * original bug.
 *
 * @param {Array<{from: Date, to: Date, kwh: number}>} gaps
 * @param {Date} from
 * @param {Date} to
 * @return {number}
 */
function estimatedEnergy(gaps, from, to) {
  // Deduplicated by extent first. An interval lying wholly inside one gap finds
  // that same gap from both of its boundaries, and its energy is attributable
  // once, not twice.
  const distinct = new Map();
  for (const g of gaps) distinct.set(`${g.from.getTime()}-${g.to.getTime()}`, g);

  let total = 0;
  for (const g of distinct.values()) {
    const width = (g.to.getTime() - g.from.getTime()) / HOUR_MS;
    if (width <= 0) continue;
    total += g.kwh * (overlapHours([g], from, to) / width);
  }
  return total;
}

/**
 * Consumption across a span, from every boundary resolved inside it.
 *
 * Two different questions get two different answers here, and conflating them is
 * a mistake worth naming:
 *
 * - **Is the total exact?** That depends only on the span's two ends. A
 *   cumulative accumulator's consumption over a span is `end − start`, so if both
 *   of those are real readings the total is exact no matter how much history went
 *   unrecorded in between. `estimatedKwh` and `estimated` therefore come from the
 *   ends alone: they are the disclosure attached to the reported figure, and
 *   claiming a figure was estimated when it is exact is as wrong as the reverse.
 * - **How much of the span has no recorded history?** That is a data-quality
 *   question, it is independent of the total's exactness, and it is only as
 *   accurate as the sampling. `unrecordedHours` answers it from *every* boundary,
 *   since an interior one is the only thing that can see a hole in the middle.
 *
 * Interior boundaries are still worth passing, though no longer for the first
 * answer. Bracketed boundaries are carried forward rather than substituted, so
 * they contribute nothing to `estimatedKwh` however wide their bracket — the case
 * that once reported 290 days and 95% estimated where the truth was 66 days and 9%
 * cannot arise from a bracket at all now. What interior boundaries still do is
 * bound each *unrecorded* stretch to roughly the probe spacing, and they remain
 * the only thing that can see a hole in the middle of the span.
 *
 * Stretches are unioned and clipped to the span, deduplicated by extent so a wide
 * one seen from several boundaries is charged once.
 *
 * @param {ResolvedBoundary[]} boundaries in span order; first is `from`, last is `to`
 * @param {Date} from
 * @param {Date} to
 * @param {Object} [opts] as built by {@link estimationOptions}
 * @param {number} [opts.materialSharePct]
 * @return {BoundaryDelta}
 */
export function spanDelta(boundaries, from, to, opts) {
  const none = (reason) => ({
    kwh: null, estimated: false, estimatedHours: 0, estimatedKwh: 0, estimatedKind: null,
    unrecordedHours: 0, longestGap: null, reason
  });
  if (!boundaries?.length) return none('no boundaries resolved');

  const a = boundaries[0];
  const b = boundaries[boundaries.length - 1];

  if (a?.usage == null) return none(a?.reason ?? 'no reading at start of interval');
  if (b?.usage == null) return none(b?.reason ?? 'no reading at end of interval');

  // Data quality, from every boundary, whether or not anything was substituted.
  // Hoisted above the regression branch so a span whose register fell still reports
  // the unrecorded history either side of it.
  const allQuiet = boundaries.flatMap(quietOf);
  const unrecordedHours = overlapHours(allQuiet, from, to);
  const longestGap = widestGap(allQuiet, from, to);

  // The register ended the span below where it started. A reset takes the whole
  // pre-reset total with it and cannot be recovered from two points, so that is
  // still withheld — see `DEFAULT_REGRESSION_SHARE_PCT` for where the line sits.
  //
  // A correction is bounded by its own size, and refusing the whole span over it
  // costs far more than it protects: at 3CS a 150 kWh backwards step on one air
  // source heat pump withheld all 72,000 kWh of a month metered across 58 boards,
  // and the month then read as an outage on a dashboard whose other artefacts showed
  // figures for every meter. So the span reports the only defensible total, zero,
  // and discloses the step as the energy at stake.
  //
  // This deliberately UNDERSTATES by however much the meter really consumed, which
  // is the direction the NABERS method forbids of a substituted value. That is a
  // considered trade for an indicative dashboard on a building still commissioning
  // its metering: the alternative on offer was substituting this meter's own mean
  // rate, which for a standby unit invents thousands of kWh to satisfy the rule and
  // is the larger error. It is bounded, badged amber and named per board, so it
  // cannot be quoted without the caveat travelling with it — but a submission figure
  // must come from the FM provider's verified readings for the affected month.
  if (b.usage < a.usage) {
    const droppedKwh = a.usage - b.usage;
    if (!opts?.enabled || droppedKwh > regressionAllowanceKwh(a.usage, opts)) {
      return none(regressionReason('accumulator reset mid-interval', a.usage, droppedKwh, opts));
    }
    return {
      kwh:          0,
      estimated:    true,
      // The whole span, because the register could have fallen anywhere inside it.
      estimatedHours: overlapHours([{from, to}], from, to),
      estimatedKwh: droppedKwh,
      // Named, so the disclosure can say this understated rather than claiming the
      // inflated forward projection it is not.
      estimatedKind: 'regressed',
      unrecordedHours,
      longestGap,
      reason:       null
    };
  }

  const kwh = b.usage - a.usage;

  // What the reported total rests on: only its two ends can make it inexact.
  const endGaps = [...gapOf(a), ...gapOf(b)];
  const estimatedHours = overlapHours(endGaps, from, to);
  // Capped at the span's own energy: overlapping gaps, or a gap wider than the
  // span, must not disclose more than there is.
  const estimatedKwh = Math.min(kwh, Math.max(0, estimatedEnergy(endGaps, from, to)));

  // `unrecordedHours` and `longestGap` are resolved above, before the regression
  // branch. They are the column that keeps a carried-forward stretch visible: the
  // figure it produced is exact, but a board that recorded nothing for eight days is
  // still worth chasing, and dropping it here would hide that.
  const materialPct = opts?.materialSharePct ?? DEFAULT_MATERIAL_SHARE_PCT;
  const material = estimatedKwh > 0 && estimatedKwh > (materialPct / 100) * kwh;

  return {
    kwh,
    estimated:    material,
    estimatedHours,
    estimatedKwh,
    // Only extrapolation sets a gap pair, so any energy attributed here came from
    // a forward projection past an unreachable meter.
    estimatedKind: estimatedKwh > 0 ? 'projected' : null,
    unrecordedHours,
    longestGap,
    reason:       null
  };
}

/**
 * Consumption between two resolved boundaries: {@link spanDelta} for the common
 * case of a span sampled only at its ends, such as one month of the rolling
 * table.
 *
 * @param {ResolvedBoundary} a value at `from`
 * @param {ResolvedBoundary} b value at `to`
 * @param {Date} from
 * @param {Date} to
 * @param {Object} [opts] as built by {@link estimationOptions}
 * @return {BoundaryDelta}
 */
export function boundaryDelta(a, b, from, to, opts) {
  return spanDelta([a, b], from, to, opts);
}

/**
 * Sum a pool of deltas, where one unknown makes the whole sum unknown.
 *
 * Strict, matching the rule the stores already apply: dropping a dead meter and
 * summing the rest is a silent undercount, and a flatteringly low rating is the
 * one failure mode worth ruling out by construction. Estimation reduces how
 * often this fires; it does not license relaxing it.
 *
 * An empty pool is null, not 0 — twelve months of a real zero would publish a
 * settled six-star rating for a building with no meters.
 *
 * @param {BoundaryDelta[]} deltas
 * @return {BoundaryDelta}
 */
export function sumDeltas(deltas) {
  const none = (reason) => ({
    kwh: null, estimated: false, estimatedHours: 0, estimatedKwh: 0, estimatedKind: null,
    unrecordedHours: 0, longestGap: null, reason
  });
  if (!deltas?.length) return none('no meters');

  let kwh = 0;
  let estimatedKwh = 0;
  let estimatedHours = 0;
  let unrecordedHours = 0;
  let longestGap = null;
  let material = false;
  const kinds = [];
  for (const d of deltas) {
    if (d?.kwh == null || Number.isNaN(d.kwh)) return none(d?.reason ?? 'unreadable meter');
    kwh += d.kwh;
    estimatedKwh += d.estimatedKwh;
    // The longest single outage in the pool, not the sum: two boards down over
    // the same week is one week of degraded data, not two.
    estimatedHours = Math.max(estimatedHours, d.estimatedHours);
    unrecordedHours = Math.max(unrecordedHours, d.unrecordedHours ?? 0);
    // Same argument, and carried rather than dropped: this used to be absent from
    // the summed shape while the `BoundaryDelta` typedef promised it, so a caller
    // reading it off a pool got `undefined` instead of the documented null.
    if (d.longestGap && (!longestGap || d.longestGap.hours > longestGap.hours)) {
      longestGap = d.longestGap;
    }
    kinds.push(d.estimatedKind ?? null);
    if (d.estimated) material = true;
  }
  // `material`, propagated from the per-meter deltas, not recomputed from hours.
  // An earlier version keyed this on `estimatedHours > 0`, reasoning that a meter
  // which consumed nothing across a gap had still been estimated. That is exactly
  // backwards: `pkg/auto/history` does not record unchanged readings, so silence
  // with no accumulator movement is the *normal* case for an idle meter and there
  // is nothing to disclose. Keying on hours made the dashboard cry wolf on every
  // car park and exterior lighting meter.
  return {
    kwh, estimated: material, estimatedHours, estimatedKwh,
    estimatedKind: mergeEstimationKinds(kinds),
    unrecordedHours, longestGap, reason: null
  };
}

/**
 * The share of reported energy that was estimated, as a percentage.
 *
 * This is the single number the dashboard, the report footer and the CSV all
 * quote. Null when there is no energy to take a share of, so callers show
 * nothing rather than a meaningless 0%.
 *
 * @param {Array<{kwh: number|null, estimatedKwh: number}>} items
 * @return {number|null}
 */
export function estimatedSharePct(items) {
  let total = 0;
  let estimated = 0;
  for (const it of items ?? []) {
    if (it?.kwh == null) continue;
    total += it.kwh;
    estimated += it.estimatedKwh ?? 0;
  }
  if (total <= 0) return null;
  return (estimated / total) * 100;
}

/**
 * Round a duration in hours to a short human string: "6 h", "3.5 days".
 *
 * @param {number} hours
 * @return {string}
 */
export function formatGap(hours) {
  if (!Number.isFinite(hours) || hours <= 0) return '';
  if (hours < 48) return `${Math.round(hours)} h`;
  const days = hours / 24;
  return `${days < 10 ? days.toFixed(1) : Math.round(days)} days`;
}
