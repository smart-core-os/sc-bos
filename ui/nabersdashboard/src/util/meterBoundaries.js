/**
 * Fetch the accumulator value for a set of meters at a set of instants, filling
 * gaps in the meter history where it can.
 *
 * This is the I/O half of gap filling; {@link module:util/meterEstimation} is
 * the maths. It exists so the base building and tenancy stores share one
 * implementation rather than the two divergent copies they had before.
 *
 * The fetch is deliberately staged, so a healthy building costs no more than it
 * did before this feature existed:
 *
 * 0. One live `GetMeterReading` per meter. This is the liveness signal, and it is
 *    load-bearing rather than a convenience. `pkg/auto/history` records a meter
 *    reading only when it changes, so a silent history is ambiguous: the meter
 *    may be offline, or it may simply be idle. A meter that answers a live read
 *    is present, and the value it returns is its accumulator *now* — which both
 *    resolves the trailing boundary exactly and turns what would have been a
 *    forward projection into a bounded carry-forward. Without it, an idle meter
 *    gets consumption invented for it at its own historic rate.
 * 1. One "last record at or before the instant" probe per (meter, instant).
 *    Sorted `desc` from the instant rather than `asc` from it, so it finds the
 *    nearest reading rather than the first one within an arbitrary two-day
 *    window, and it carries a timestamp.
 * 2. A "first record at or after the instant" probe, issued *only* where pass 1
 *    came back empty or stale. A meter reporting every fifteen minutes never
 *    reaches this, so the extra cost falls on exactly the meters with gaps.
 * 3. Purely local resolution. Each meter's samples are pooled across every
 *    instant, so a projection rate comes free from readings already fetched for
 *    other boundaries rather than costing more queries.
 *
 * @module util/meterBoundaries
 */

import {startOfMonth, addMonths, format} from 'date-fns';
import {getMeterReading, getMeterReadingsBefore, getMeterReadingAfter} from '@/api/sc/traits/meter.js';
import {mapLimit, MAX_CONCURRENT_READS} from '@/util/concurrency.js';
import {describeRpcError} from '@/util/rpcError.js';
import {
  normaliseSamples, resolveBoundary, observedTickKwh, idleToleranceFor,
  plausibleCeiling, plausibleSamples, withoutDropouts
} from '@/util/meterEstimation.js';

const HOUR_MS = 60 * 60 * 1000;

/**
 * @param {string} name
 * @param {Date} at
 * @return {string}
 */
export const boundaryKey = (name, at) => `${name}@${at.getTime()}`;

/**
 * The instants to resolve for a period-to-date figure: its start, every month
 * boundary inside it, and its end.
 *
 * Interior instants exist for the gap accounting, not the energy. A cumulative
 * accumulator's total over a span needs only the two ends, and more samples
 * cannot improve it — but with only two, the opening boundary brackets across
 * every record in between that was never fetched, so a hole of a couple of months
 * is reported as spanning almost the whole period. Monthly granularity bounds any
 * one gap to about a month and is what the rolling table already samples at, so
 * the two views agree.
 *
 * A rating period is capped at a year, so this is at most 13 instants.
 *
 * @param {Date} start
 * @param {Date} end
 * @return {Date[]} ascending, `start` first and `end` last
 */
export function periodInstants(start, end) {
  const out = [start];
  for (let m = startOfMonth(addMonths(start, 1)); m < end; m = addMonths(m, 1)) {
    out.push(m);
  }
  out.push(end);
  return out;
}

/**
 * A boundary lookup over the meters and instants it was built for.
 *
 * @typedef {{
 *   get: function(string, Date): import('./meterEstimation.js').ResolvedBoundary,
 *   samplesFor: function(string): Array<{usage: number, at: Date}>,
 *   errorFor: function(string): string|null,
 *   reachable: function(string): boolean,
 *   isIdle: function(string): boolean,
 *   tickFor: function(string): number|null,
 *   rejectedFor: function(string): number
 * }} BoundaryTable
 */

/**
 * Read every (meter, instant) pair, filling gaps per `opts`.
 *
 * Never throws and never rejects: {@link mapLimit}'s workers must not see a
 * rejection, and one meter's transient deadline must not blow away every other
 * end use. A meter that could not be read at all resolves to an `unknown`
 * boundary carrying the RPC reason, which is what lets the UI name the board
 * rather than just the end use.
 *
 * @param {string[]} meterNames may contain duplicates; they are fetched once
 * @param {Date[]} instants
 * @param {Object} opts as built by {@link import('./meterEstimation.js').estimationOptions}
 * @param {boolean} opts.enabled
 * @param {number} opts.gapThresholdHours
 * @param {number} opts.searchWindowDays
 * @param {number} opts.extrapolationUpliftPct
 * @return {Promise<BoundaryTable>}
 */
export async function readBoundaries(meterNames, instants, opts) {
  const names = [...new Set(meterNames ?? [])];
  const errors = new Map();
  /** Meters that did not answer a live read, so a trailing silence is an outage. */
  const unreachable = new Set();

  /**
   * @param {string} name
   * @param {*} e
   * @return {null}
   */
  const noteError = (name, e) => {
    // First reason wins: later probes for the same dead meter add nothing.
    if (!errors.has(name)) errors.set(name, describeRpcError(e));
    return null;
  };

  // ── Pass 0: live read, to tell an idle meter from an offline one ────────────
  // A failure here is not a failure of the whole meter: its history may still be
  // perfectly readable, and a bounded interval needs no liveness signal at all.
  // It only decides how a *trailing* silence is treated.
  //
  // Stamped at the latest instant the caller asked about rather than at the wall
  // clock, whenever the two are within the gap threshold of each other — which
  // for a period-to-date read they always are, since the caller's "now" is a few
  // milliseconds earlier than this call. That makes the closing boundary an exact
  // hit rather than one resolved from a bracket a millisecond wide.
  const readAt = new Date();
  const latest = instants.reduce((a, b) => (b.getTime() > a.getTime() ? b : a), instants[0] ?? readAt);
  const liveAt = (readAt.getTime() - latest.getTime()) <= opts.gapThresholdHours * HOUR_MS
    ? latest
    : readAt;
  const live = await mapLimit(names, MAX_CONCURRENT_READS, async (name) => {
    try {
      const reading = await getMeterReading(name);
      return reading?.usage == null ? null : {usage: reading.usage, at: liveAt};
    } catch (e) {
      // Deliberately not `noteError`: an unreachable meter is a distinct, and
      // more informative, condition than a failed history query, and the reason
      // reported to the UI should describe whichever actually blocked the figure.
      unreachable.add(name);
      void e;
      return null;
    }
  });
  const liveByMeter = new Map(names.map((name, i) => [name, live[i]]));
  // No usable live value and no thrown error still means we cannot vouch for the
  // meter being present, so treat it as unreachable for gap purposes.
  names.forEach(name => {
    if (!liveByMeter.get(name)) unreachable.add(name);
  });

  const work = names.flatMap(name => instants.map(at => ({name, at})));

  // ── Pass 1: a short run of readings ending at or before each instant ───────
  // A run rather than a single record, for the same one query per instant: the
  // page is just wider. The nearest reading is what the boundary needs, and the
  // consecutive readings behind it are what measures the meter's resolution —
  // see `observedTickKwh`. Meters tick at different quanta, some every 1 kWh and
  // some every 16, so that resolution has to be measured per device rather than
  // configured once for all of them.
  const runs = await mapLimit(work, MAX_CONCURRENT_READS, async ({name, at}) => {
    try {
      return await getMeterReadingsBefore(name, at, opts.searchWindowDays, opts.tickSampleCount);
    } catch (e) {
      noteError(name, e);
      return [];
    }
  });
  // Runs are ascending, so the reading nearest the instant is the last one.
  const before = runs.map(run => (run.length ? run[run.length - 1] : null));

  // ── Pass 2: only where pass 1 left a gap ───────────────────────────────────
  // A stale pass-1 sample is not yet a gap — it may be bracketed by a reading
  // just after the instant, which turns a forward projection into a carry-forward
  // and proves the accumulator had not moved. That is worth one query to find out.
  const staleMs = opts.gapThresholdHours * HOUR_MS;
  const followUps = work.filter((w, i) => {
    const b = before[i];
    return b === null || (w.at.getTime() - b.at.getTime()) > staleMs;
  });

  const after = await mapLimit(followUps, MAX_CONCURRENT_READS, async ({name, at}) => {
    try {
      return await getMeterReadingAfter(name, at, opts.searchWindowDays);
    } catch (e) {
      return noteError(name, e);
    }
  });

  // ── Pass 3: pool each meter's samples and resolve locally ──────────────────
  const samplesByMeter = new Map(names.map(n => [n, []]));
  // Runs are kept grouped as well as pooled: the resolution measurement needs to
  // know which readings were consecutive, and the flattened pool has lost that.
  const runsByMeter = new Map(names.map(n => [n, []]));
  work.forEach(({name}, i) => {
    const run = runs[i];
    if (!run?.length) return;
    samplesByMeter.get(name).push(...run);
    if (run.length > 1) runsByMeter.get(name).push(run);
  });
  followUps.forEach(({name}, i) => {
    if (after[i]) samplesByMeter.get(name).push(after[i]);
  });
  // The live value joins the pool as a sample in its own right, so a boundary at
  // "now" is an exact hit and any earlier boundary is bracketed by it rather than
  // projected past it.
  names.forEach(name => {
    const l = liveByMeter.get(name);
    if (l) samplesByMeter.get(name).push(l);
  });
  // Corrupt readings are discarded before anything reads the pool — the boundary
  // maths, the resolution measurement and the rate all take it on trust. A driver
  // fault can leave records a cumulative accumulator could not have produced, and
  // an unfiltered spike wrecks the mean rate in particular.
  const rejectedByMeter = new Map();
  for (const [name, raw] of samplesByMeter) {
    const sorted = normaliseSamples(raw);
    const ceiling = plausibleCeiling(sorted, liveByMeter.get(name));
    const {samples, rejected} = plausibleSamples(sorted, ceiling);
    // Then the dropouts: a bare 0 in the middle of a healthy series is neither
    // negative nor above the current value, so the ceiling filter cannot see it, and
    // it is the corruption that does the most damage precisely because it looks like
    // a reset rather than like corruption. Both counts are reported as one, since
    // from the outside they are the same fact: readings a cumulative register could
    // not have produced, discarded before any figure was derived.
    const dropouts = withoutDropouts(samples, opts);
    samplesByMeter.set(name, dropouts.samples);
    rejectedByMeter.set(name, rejected + dropouts.rejected);
  }
  // Runs feed the resolution measurement, so they need the same filters or a spike
  // inside one would be measured as a step — and a dropout would contribute the
  // register's whole value as the step after it.
  for (const [name, meterRuns] of runsByMeter) {
    const ceiling = plausibleCeiling(samplesByMeter.get(name), liveByMeter.get(name));
    runsByMeter.set(name, meterRuns
      .map(run => withoutDropouts(plausibleSamples(run, ceiling).samples, opts).samples)
      .filter(run => run.length > 1));
  }

  const resolved = new Map();
  const idle = new Set();
  const ticks = new Map();
  for (const name of names) {
    const samples = samplesByMeter.get(name);
    // The meter's own resolution sets its idle threshold, capped against how far
    // it moved over the history we hold. Measured before any boundary is resolved,
    // so the cap cannot depend on the threshold it is capping.
    const observedTick = observedTickKwh(runsByMeter.get(name));
    const spanKwh = samples.length > 1
      ? samples[samples.length - 1].usage - samples[0].usage
      : 0;
    const idleToleranceKwh = idleToleranceFor({observedTick, spanKwh, opts});
    ticks.set(name, observedTick);

    const meterOpts = {...opts, unreachable: unreachable.has(name), idleToleranceKwh};
    instants.forEach((at) => {
      const b = resolveBoundary(samples, at, meterOpts);
      if (b.usage === null) {
        // An RPC failure is a more useful reason than "no readings near boundary":
        // "deadline exceeded" tells an engineer to look at the network, "no history
        // recorded" tells them to look at the meter.
        if (errors.has(name)) {
          b.reason = errors.get(name);
        } else if (samples.length) {
          // Name the instant and what *is* held. Bare, this read "no reading at or
          // before this point" against a meter with a current reading and one from
          // today, which looks like the check is broken. The instant is often months
          // back, and the useful fact — the one that says whether looking further
          // back could ever have helped — is where the meter's history begins.
          b.reason = `${b.reason} (${format(at, 'd MMM yy')}); ` +
            `earliest held ${format(samples[0].at, 'd MMM yy')}`;
        }
        // Independently of the above, not instead of it. This was an `else if`, so a
        // single discarded reading suppressed the earliest-held detail entirely — and
        // those two travel together far more often than not, because discarding a
        // meter's only early record is *why* it then has no reading at the boundary.
        // The one case that needed both facts was the one case that got neither.
        const rejected = rejectedByMeter.get(name);
        if (rejected > 0) {
          b.reason = `${b.reason} (${rejected} implausible readings discarded)`;
        }
      }
      resolved.set(boundaryKey(name, at), b);
    });
    // Reachable, but its history has nothing recent to say. Not an error and not
    // estimated — the accumulator genuinely has not moved — but worth surfacing:
    // a meter that ought to be consuming and reads idle is a commissioning
    // fault, and it is the one case a silently-caching driver could hide.
    if (!unreachable.has(name) && samples.length) {
      const last = samples[samples.length - 1];
      const penultimate = samples.length > 1 ? samples[samples.length - 2] : null;
      // `<= idleToleranceKwh`, not `<= 0`: a meter that ticked once in a fortnight
      // is idle in every sense that matters, and insisting on an exactly unmoved
      // accumulator would miss it.
      if (penultimate && last.at.getTime() - penultimate.at.getTime() > staleMs &&
          last.usage - penultimate.usage <= idleToleranceKwh) {
        idle.add(name);
      }
    }
  }

  const unknown = {
    usage: null, quality: 'unknown', gapFrom: null, gapTo: null, gapKwh: 0,
    quietFrom: null, quietTo: null, reason: 'meter not requested'
  };
  return {
    get:        (name, at) => resolved.get(boundaryKey(name, at)) ?? unknown,
    samplesFor: (name) => samplesByMeter.get(name) ?? [],
    errorFor:   (name) => errors.get(name) ?? null,
    reachable:  (name) => !unreachable.has(name),
    isIdle:     (name) => idle.has(name),
    tickFor:    (name) => ticks.get(name) ?? null,
    rejectedFor: (name) => rejectedByMeter.get(name) ?? 0
  };
}
