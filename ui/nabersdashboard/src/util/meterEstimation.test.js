import {describe, it, expect} from 'vitest';
import {
  estimationOptions,
  normaliseSamples,
  meanRate,
  resolveBoundary,
  overlapHours,
  widestGap,
  observedTickKwh,
  idleToleranceFor,
  plausibleCeiling,
  plausibleSamples,
  regressionAllowanceKwh,
  withoutDropouts,
  spanDelta,
  boundaryDelta,
  sumDeltas,
  estimatedSharePct,
  formatGap
} from './meterEstimation.js';

const HOUR = 60 * 60 * 1000;

/** An arbitrary fixed epoch, so nothing here depends on the wall clock. */
const T0 = new Date('2026-01-01T00:00:00Z').getTime();

/**
 * @param {number} hoursFromT0
 * @return {Date}
 */
const at = (hoursFromT0) => new Date(T0 + hoursFromT0 * HOUR);

/**
 * @param {number} hoursFromT0
 * @param {number} usage
 * @return {{usage: number, at: Date}}
 */
const sample = (hoursFromT0, usage) => ({at: at(hoursFromT0), usage});

const opts = estimationOptions();

describe('estimationOptions', () => {
  it('defaults to enabled with a 3 h threshold and 10% uplift', () => {
    expect(opts).toEqual({
      enabled:                true,
      gapThresholdHours:      3,
      searchWindowDays:       45,
      extrapolationUpliftPct: 10,
      materialSharePct:       0.5,
      idleToleranceKwh:       1,
      tickSampleCount:        5,
      tickCapSharePct:        2,
      regressionSharePct:     1,
      regressionToleranceKwh: 1,
      maxDropoutSamples:      2
    });
  });

  it('keeps a configured zero uplift rather than restoring the default', () => {
    // `||` would silently put the 10% back; a configured 0 means "interpolate
    // but never inflate", which is a real choice.
    expect(estimationOptions({extrapolationUpliftPct: 0}).extrapolationUpliftPct).toBe(0);
  });

  it('can be turned off', () => {
    expect(estimationOptions({enabled: false}).enabled).toBe(false);
  });
});

describe('normaliseSamples', () => {
  it('sorts by time and drops duplicate instants', () => {
    const out = normaliseSamples([sample(5, 50), sample(1, 10), sample(5, 50), sample(3, 30)]);
    expect(out.map(s => s.usage)).toEqual([10, 30, 50]);
  });

  it('discards nulls and unusable readings', () => {
    const out = normaliseSamples([
      null,
      {usage: null, at: at(1)},
      {usage: NaN, at: at(2)},
      {usage: 5, at: new Date('nonsense')},
      sample(4, 40)
    ]);
    expect(out).toEqual([sample(4, 40)]);
  });
});

describe('meanRate', () => {
  it('is kWh per millisecond across the sample span', () => {
    // 100 kWh over 10 hours.
    expect(meanRate([sample(0, 0), sample(10, 100)])).toBeCloseTo(100 / (10 * HOUR), 12);
  });

  it('ignores a reset rather than reporting a negative rate', () => {
    // 0 → 100, reset to 0, then 0 → 40. 140 kWh of real consumption, not -60.
    const rate = meanRate([sample(0, 0), sample(10, 100), sample(11, 0), sample(20, 40)]);
    expect(rate).toBeCloseTo(140 / (20 * HOUR), 12);
  });

  it('is null when there is nothing to measure a rate over', () => {
    expect(meanRate([])).toBeNull();
    expect(meanRate([sample(0, 10)])).toBeNull();
    expect(meanRate(null)).toBeNull();
  });
});

describe('resolveBoundary', () => {
  it('takes an exact hit as actual', () => {
    const b = resolveBoundary([sample(0, 10), sample(24, 34)], at(0), opts);
    expect(b).toMatchObject({usage: 10, quality: 'actual', gapFrom: null, gapTo: null});
  });

  it('carries the earlier reading forward inside the threshold', () => {
    // Readings 2 h apart straddling the boundary: normal reporting, not a gap.
    // History records only changes, so the accumulator still read 10 here and
    // stepped to 30 at the poll that wrote the later record. Interpolation used to
    // report 20, splitting a step that had not happened yet.
    const b = resolveBoundary([sample(-1, 10), sample(1, 30)], at(0), opts);
    expect(b.quality).toBe('actual');
    expect(b.usage).toBe(10);
    expect(b.quietFrom).toBeNull();
  });

  it('is actual when the accumulator did not move, however wide the bracket', () => {
    // Reported from site, with these exact readings: a record on 2026-06-24 at
    // 1030.000 and the next on 2026-07-14, twenty days later, also at 1030.000 —
    // a device restart writing a fresh record of an unchanged value. That was
    // shown as a twenty-day gap.
    //
    // It is not one, and the reason is stronger than "not worth warning about":
    // the accumulator is monotonic, so if it reads the same at both ends its value
    // at every instant between them is that same value. Known exactly. There is
    // nothing to interpolate and nothing to disclose.
    const before = {at: new Date('2026-06-24T14:05:00Z'), usage: 1030};
    const after = {at: new Date('2026-07-14T09:40:00Z'), usage: 1030};
    const b = resolveBoundary([before, after], new Date('2026-07-01T00:00:00Z'), opts);
    expect(b.quality).toBe('actual');
    expect(b.usage).toBe(1030);
    expect(b.gapFrom).toBeNull();
    expect(b.gapKwh).toBe(0);
  });

  it('is actual when the accumulator ticked only once across weeks', () => {
    // Meters on lightly used circuits change very infrequently. One tick inside
    // the tolerance leaves the value known to within that tolerance.
    const b = resolveBoundary([sample(-480, 1030), sample(480, 1030.5)], at(0), opts);
    expect(b.quality).toBe('actual');
    expect(b.gapFrom).toBeNull();
  });

  it('reports a materially-crossed bracket as unrecorded, not as estimated', () => {
    // The same twenty-day span, but 400 kWh crossed it. Records really are missing,
    // so the stretch is worth chasing — but the value here is still the earlier
    // reading, and the 400 kWh belongs to the interval ending at the later record.
    const b = resolveBoundary([sample(-240, 1030), sample(240, 1430)], at(0), opts);
    expect(b.quality).toBe('actual');
    expect(b.usage).toBe(1030);
    // Nothing was substituted, so nothing is disclosed...
    expect(b.gapFrom).toBeNull();
    expect(b.gapKwh).toBe(0);
    // ...but the missing history is still located, for the quality view.
    expect(b.quietFrom).toEqual(at(-240));
    expect(b.quietTo).toEqual(at(240));
  });

  it('lets a site tighten the idle tolerance', () => {
    // The tolerance no longer decides the value — that is carried forward either
    // way — only whether a half-kWh movement counts as history worth reporting.
    const strict = estimationOptions({idleToleranceKwh: 0});
    const b = resolveBoundary([sample(-480, 1030), sample(480, 1030.5)], at(0), strict);
    expect(b.usage).toBe(1030);
    expect(b.quietFrom).toEqual(at(-480));
  });

  it('is actual when a real reading is close, however wide the bracket', () => {
    // The site-reported bug. The sample pool is sparse by design — about one
    // reading per probed instant — so consecutive pooled readings sit a month
    // apart even for a meter recording every fifteen minutes. Judging by bracket
    // width flagged such a meter as having a 31-day gap at every month boundary.
    // A reading a quarter of an hour away pins the boundary; the bracket's far
    // edge being a month later says nothing about this instant.
    const b = resolveBoundary([sample(-0.25, 1000), sample(744, 2488)], at(0), opts);
    expect(b.quality).toBe('actual');
    expect(b.gapFrom).toBeNull();
    expect(b.gapKwh).toBe(0);
    // The value still comes from the bracket, so it is not merely snapped back.
    expect(b.usage).toBeGreaterThanOrEqual(1000);
    expect(b.usage).toBeLessThan(1001);
  });

  it('is actual when the close reading is the later one', () => {
    const b = resolveBoundary([sample(-500, 100), sample(1, 2000)], at(0), opts);
    expect(b.quality).toBe('actual');
    expect(b.gapFrom).toBeNull();
  });

  it('carries a bracketed gap forward and reports its extent', () => {
    // 48 h apart, well past the 3 h threshold, with the boundary a quarter of the
    // way in. Interpolation put it a quarter up the step, at 160; carry-forward
    // leaves it at 100, because that is what every poll across those 48 h read.
    const b = resolveBoundary([sample(-12, 100), sample(36, 340)], at(0), opts);
    expect(b.quality).toBe('actual');
    expect(b.usage).toBe(100);
    expect(b.gapKwh).toBe(0);
    expect(b.quietFrom).toEqual(at(-12));
    expect(b.quietTo).toEqual(at(36));
  });

  it('treats a reset inside the bracket as unknown, not zero', () => {
    const b = resolveBoundary([sample(-12, 500), sample(36, 20)], at(0), opts);
    expect(b.usage).toBeNull();
    expect(b.quality).toBe('unknown');
    expect(b.reason).toMatch(/reset/);
  });

  it('carries forward across a bracket the register merely stepped back over', () => {
    // 87,335.5 down to 87,185.5, the 3CS ASHP 2 case: a 150 kWh step, 0.17% of the
    // register. Nulling the boundary would withhold the months on BOTH sides of a
    // correction whose entire cost is 150 kWh, so the earlier reading is carried
    // forward exactly as any other bracket's is. `spanDelta` discloses the step
    // against the span that contains it.
    const b = resolveBoundary([sample(-12, 87335.5), sample(36, 87185.5)], at(0), opts);
    expect(b.usage).toBe(87335.5);
    expect(b.quality).toBe('actual');
    expect(b.reason).toBeNull();
    // Still reported as unrecorded history, because it is worth chasing.
    expect(b.quietFrom).toEqual(at(-12));
    expect(b.quietTo).toEqual(at(36));
  });

  it('will not carry forward across a step back when estimation is off', () => {
    const off = estimationOptions({enabled: false});
    const b = resolveBoundary([sample(-12, 87335.5), sample(36, 87185.5)], at(0), off);
    expect(b.usage).toBeNull();
    expect(b.reason).toMatch(/reset/);
  });

  it('is unknown when there is no history near the boundary at all', () => {
    const b = resolveBoundary([], at(0), opts);
    expect(b).toMatchObject({usage: null, quality: 'unknown'});
  });

  it('accepts a one-sided reading inside the threshold as actual', () => {
    // The live-reading case: last record 1 h old, meter perfectly healthy.
    const b = resolveBoundary([sample(-10, 100), sample(-1, 190)], at(0), opts);
    expect(b).toMatchObject({usage: 190, quality: 'actual'});
  });

  describe('a trailing silence: idle meter versus outage', () => {
    // `pkg/auto/history` records a meter reading only when it changes, so a
    // silent history does not mean a silent meter. These are the two readings of
    // the same data, told apart by whether the meter answered a live read.
    const samples = [sample(-20, 0), sample(-10, 100)];

    it('reports a reachable meter\'s silence as zero consumption', () => {
      // The regression this guards: projecting the meter's own 10 kWh/h rate
      // across the silence invented 110 kWh that was never consumed.
      const b = resolveBoundary(samples, at(0), opts);
      expect(b.quality).toBe('actual');
      expect(b.usage).toBe(100);
      expect(b.gapFrom).toBeNull();
      expect(b.gapKwh).toBe(0);
    });

    it('projects an unreachable meter forward, and upward', () => {
      // 10 kWh/h established over 10 h, then unreachable. Boundary is 10 h past
      // the last reading, so a plain projection is +100 and the uplift makes 110.
      const b = resolveBoundary(samples, at(0), {...opts, unreachable: true});
      expect(b.quality).toBe('extrapolated');
      expect(b.usage).toBeCloseTo(100 + 110, 6);
      // Upward, because this boundary ends the interval being reported, so a
      // substituted value cannot flatter the rating.
      expect(b.usage).toBeGreaterThan(200);
      expect(b.gapFrom).toEqual(at(-10));
      expect(b.gapTo).toEqual(at(0));
      expect(b.gapKwh).toBeCloseTo(110, 6);
    });

    it('is unknown when an unreachable meter has too little history for a rate', () => {
      const b = resolveBoundary([sample(-10, 100)], at(0), {...opts, unreachable: true});
      expect(b).toMatchObject({usage: null, quality: 'unknown'});
      expect(b.reason).toMatch(/too little history/);
    });

    it('still takes a recent reading as actual either way', () => {
      // Inside the threshold there is no silence to interpret.
      const recent = [sample(-10, 100), sample(-1, 190)];
      expect(resolveBoundary(recent, at(0), opts).usage).toBe(190);
      expect(resolveBoundary(recent, at(0), {...opts, unreachable: true}).usage).toBe(190);
    });
  });

  describe('a leading gap is refused, never guessed', () => {
    it('is unknown when nothing was recorded at or before the instant', () => {
      // Backward extrapolation used to live here and invented 880 kWh for a
      // 40-day leading gap. Idle-since-before-the-period and records-start-late
      // are indistinguishable, and the two answers differ by hundreds of kWh.
      const b = resolveBoundary([sample(10, 1000), sample(20, 1100)], at(0), opts);
      expect(b).toMatchObject({usage: null, quality: 'unknown', gapKwh: 0});
      expect(b.reason).toMatch(/at or before/);
    });

    it('is refused for an unreachable meter too', () => {
      const b = resolveBoundary([sample(10, 1000), sample(20, 1100)], at(0),
        {...opts, unreachable: true});
      expect(b).toMatchObject({usage: null, quality: 'unknown'});
    });
  });

  describe('with estimation disabled', () => {
    const off = estimationOptions({enabled: false});

    it('still smooths inside the threshold', () => {
      expect(resolveBoundary([sample(-1, 10), sample(1, 30)], at(0), off).quality).toBe('actual');
    });

    it('still carries a bracketed gap forward, which is not a substitution', () => {
      // The flag withholds figures resting on a value nobody measured. A
      // carried-forward boundary is not one of those: on-change recording says the
      // accumulator held 100 across the bracket, so turning estimation off must not
      // turn that measurement into an unknown. It governs the projection branch.
      const b = resolveBoundary([sample(-12, 100), sample(36, 340)], at(0), off);
      expect(b).toMatchObject({usage: 100, quality: 'actual'});
    });

    it('refuses to project an unreachable meter forward', () => {
      const b = resolveBoundary([sample(-20, 0), sample(-10, 100)], at(0),
        {...off, unreachable: true});
      expect(b).toMatchObject({usage: null, quality: 'unknown'});
    });

    it('still reports a reachable meter\'s silence as zero, which is not an estimate', () => {
      // Turning estimation off must not turn a measurement into an unknown.
      const b = resolveBoundary([sample(-20, 0), sample(-10, 100)], at(0), off);
      expect(b).toMatchObject({usage: 100, quality: 'actual'});
    });
  });
});

describe('overlapHours', () => {
  it('clips intervals to the window', () => {
    expect(overlapHours([{from: at(-10), to: at(10)}], at(0), at(5))).toBe(5);
  });

  it('counts overlapping intervals once', () => {
    const ivs = [{from: at(0), to: at(6)}, {from: at(4), to: at(10)}];
    expect(overlapHours(ivs, at(0), at(24))).toBe(10);
  });

  it('sums disjoint intervals', () => {
    const ivs = [{from: at(0), to: at(2)}, {from: at(8), to: at(11)}];
    expect(overlapHours(ivs, at(0), at(24))).toBe(5);
  });

  it('is zero when nothing overlaps', () => {
    expect(overlapHours([{from: at(30), to: at(40)}], at(0), at(24))).toBe(0);
    expect(overlapHours([], at(0), at(24))).toBe(0);
  });
});

describe('discarding readings an accumulator could not have produced', () => {
  // The sequence reported from site, from a driver fault since fixed: the record
  // history still holds 469780064 and -2147465600 between a 14848 and an 18025.
  // The negative sits just above int32's floor of -2147483648, so it is an integer
  // overflow escaping as signed, not a reading.
  const dirty = [
    sample(0, 14848),
    sample(1, 469780064),
    sample(2, -2147465600),
    sample(3, 18025)
  ];

  it('takes the live reading as the ceiling', () => {
    expect(plausibleCeiling(dirty, {at: at(4), usage: 18100})).toBe(18100);
  });

  it('falls back to the latest reading held when there is no live one', () => {
    expect(plausibleCeiling(dirty, null)).toBe(18025);
    expect(plausibleCeiling([], null)).toBeNull();
  });

  it('drops the negative and the impossibly high, keeping the rest', () => {
    const {samples, rejected} = plausibleSamples(dirty, 18025);
    expect(samples.map(s => s.usage)).toEqual([14848, 18025]);
    expect(rejected).toBe(2);
  });

  it('keeps a reading equal to the ceiling', () => {
    // The live sample is itself in the pool, so the test must be strictly greater.
    expect(plausibleSamples([sample(0, 18025)], 18025).rejected).toBe(0);
  });

  it('still drops negatives with no ceiling to compare against', () => {
    const {samples, rejected} = plausibleSamples(dirty, null);
    expect(samples.map(s => s.usage)).toEqual([14848, 469780064, 18025]);
    expect(rejected).toBe(1);
  });

  it('rescues the rate, which a spike wrecks', () => {
    // `meanRate` sums positive steps, so the 469780064 spike contributes a step of
    // nearly half a billion kWh. Any projection built on that rate is nonsense.
    const dirtyRate = meanRate(dirty);
    const {samples} = plausibleSamples(dirty, 18025);
    const cleanRate = meanRate(samples);
    expect(dirtyRate).toBeGreaterThan(cleanRate * 1000);
    // 3177 kWh across 3 hours, from the two readings that are real.
    expect(cleanRate).toBeCloseTo(3177 / (3 * HOUR), 12);
  });

  it('keeps a corrupt bracket from being read as a reset', () => {
    // Without filtering, the boundary between the spike and the negative brackets
    // an accumulator that "fell" by billions, which reads as a reset and nulls the
    // figure. Cleaned, the two real readings bracket it normally and the earlier
    // one carries forward.
    const before = resolveBoundary(dirty, at(1.5), opts);
    expect(before.usage).toBeNull();
    const {samples} = plausibleSamples(dirty, 18025);
    const after = resolveBoundary(samples, at(1.5), opts);
    expect(after.usage).toBe(14848);
  });
});

describe('observedTickKwh', () => {
  it('is the smallest positive step between adjacent readings', () => {
    // A 16 kWh meter: consecutive records differ by 16, so 16 is its resolution.
    expect(observedTickKwh([[sample(0, 100), sample(1, 116), sample(2, 132)]])).toBe(16);
  });

  it('takes the smallest across every run, not just the latest', () => {
    // A run captured while the meter was busy shows several ticks per record and
    // would overstate the quantum; another run taken when it was quiet reveals it.
    const busy = [sample(100, 5000), sample(101, 5064), sample(102, 5128)];
    const quiet = [sample(0, 100), sample(1, 116), sample(2, 132)];
    expect(observedTickKwh([busy, quiet])).toBe(16);
  });

  it('never measures across two runs', () => {
    // These two runs are a month apart with a gap between them. That difference is
    // the missing energy, not a tick, and treating it as one would set an enormous
    // threshold and hide every real outage on the meter.
    const early = [sample(0, 100), sample(1, 116)];
    const late = [sample(720, 900), sample(721, 916)];
    expect(observedTickKwh([early, late])).toBe(16);
  });

  it('ignores duplicate records and resets', () => {
    // A device restart writing an unchanged value gives a zero step; a reset gives
    // a negative one. Neither says anything about resolution.
    expect(observedTickKwh([[sample(0, 1030), sample(1, 1030), sample(2, 1030)]])).toBeNull();
    expect(observedTickKwh([[sample(0, 500), sample(1, 20), sample(2, 36)]])).toBe(16);
  });

  it('is null when there is nothing to measure', () => {
    expect(observedTickKwh([])).toBeNull();
    expect(observedTickKwh([[sample(0, 100)]])).toBeNull();
    expect(observedTickKwh(null)).toBeNull();
  });
});

describe('idleToleranceFor', () => {
  it('uses the measured resolution', () => {
    expect(idleToleranceFor({observedTick: 16, spanKwh: 100000, opts})).toBe(16);
  });

  it('falls back to the configured value when nothing was measured', () => {
    expect(idleToleranceFor({observedTick: null, spanKwh: 100000, opts})).toBe(1);
  });

  it('caps a coarse resolution against what the meter actually consumed', () => {
    // A meter that moved only 200 kWh all period should not tolerate a 16 kWh
    // step: at 2% the ceiling is 4, so a real outage cannot be dismissed as idle.
    expect(idleToleranceFor({observedTick: 16, spanKwh: 200, opts})).toBe(4);
  });

  it('never lets the cap fall below the configured floor', () => {
    // The case this whole mechanism protects. A meter that never moved has a span
    // of zero, and a cap allowed to reach zero would call its flat readings a gap
    // again — which is the bug the idle tolerance exists to prevent.
    expect(idleToleranceFor({observedTick: 16, spanKwh: 0, opts})).toBe(1);
    expect(idleToleranceFor({observedTick: 0.5, spanKwh: 0, opts})).toBe(0.5);
  });
});

describe('withoutDropouts', () => {
  const usages = (r) => r.samples.map(s => s.usage);

  it('discards a bare zero the next reading recovers from', () => {
    // EM/007 at 3CS: 916, then 0, then 930. For the 0 to be real the immersion heater
    // would have had to consume 930 kWh between two records; it consumed 14 that
    // month. The recovery is what makes it impossible, not the size of the fall.
    const r = withoutDropouts([sample(0, 916), sample(1, 0), sample(2, 930)], opts);
    expect(usages(r)).toEqual([916, 930]);
    expect(r.rejected).toBe(1);
  });

  it('discards a dip that is not all the way to zero', () => {
    const r = withoutDropouts([sample(0, 5000), sample(1, 4200), sample(2, 5010)], opts);
    expect(usages(r)).toEqual([5000, 5010]);
    expect(r.rejected).toBe(1);
  });

  it('discards a run of two bad reads', () => {
    const r = withoutDropouts(
      [sample(0, 916), sample(1, 0), sample(2, 0), sample(3, 930)], opts);
    expect(usages(r)).toEqual([916, 930]);
    expect(r.rejected).toBe(2);
  });

  it('keeps a genuine reset, which does not recover', () => {
    // The distinction the whole filter rests on. After a reset the register climbs
    // from near zero over weeks, so the reading after it is small — nowhere near the
    // old peak — and the reset must still be reported rather than quietly repaired.
    const r = withoutDropouts(
      [sample(0, 24217), sample(1, 0), sample(2, 5), sample(3, 12), sample(4, 30)], opts);
    expect(usages(r)).toEqual([24217, 0, 5, 12, 30]);
    expect(r.rejected).toBe(0);
  });

  it('will not eat a slow recovery just because it eventually passes the old peak', () => {
    // A reset whose climb does overtake the old value in the end. It must not be read
    // backwards as one long dropout, which is what bounding the run to
    // `maxDropoutSamples` protects against.
    const r = withoutDropouts(
      [sample(0, 1000), sample(1, 10), sample(2, 400), sample(3, 800), sample(4, 1200)], opts);
    expect(usages(r)).toEqual([1000, 10, 400, 800, 1200]);
    expect(r.rejected).toBe(0);
  });

  it('leaves a clean monotonic series alone', () => {
    const r = withoutDropouts([sample(0, 10), sample(1, 20), sample(2, 20), sample(3, 31)], opts);
    expect(usages(r)).toEqual([10, 20, 20, 31]);
    expect(r.rejected).toBe(0);
  });

  it('takes the first sample on trust, having nothing to compare it against', () => {
    // A leading dropout is indistinguishable from a meter whose history starts low.
    const r = withoutDropouts([sample(0, 0), sample(1, 916)], opts);
    expect(usages(r)).toEqual([0, 916]);
  });

  it('can be turned off', () => {
    const off = estimationOptions({maxDropoutSamples: 0});
    const r = withoutDropouts([sample(0, 916), sample(1, 0), sample(2, 930)], off);
    expect(usages(r)).toEqual([916, 0, 930]);
    expect(r.rejected).toBe(0);
  });

  it('is empty-safe', () => {
    expect(withoutDropouts([], opts).samples).toEqual([]);
    expect(withoutDropouts(null, opts).samples).toEqual([]);
  });
});

describe('a repaired dropout leaves the boundary exact', () => {
  it('brackets the boundary from the real readings either side', () => {
    // End to end: the record that would have read as a reset is gone, so the boundary
    // is a measurement again and the month needs no disclosure at all.
    const {samples} = withoutDropouts(
      normaliseSamples([sample(-48, 916), sample(-1, 0), sample(48, 930)]), opts);
    const b = resolveBoundary(samples, at(0), opts);
    expect(b.usage).toBe(916);
    expect(b.quality).toBe('actual');
    expect(b.reason).toBeNull();
  });
});

describe('regressionAllowanceKwh', () => {
  it('scales with the register it fell from', () => {
    expect(regressionAllowanceKwh(87335.5, opts)).toBeCloseTo(873.355, 6);
  });

  it('separates a correction from a reset by orders of magnitude', () => {
    // The discrimination this rests on. A reset is essentially the whole register;
    // the corrections seen at 3CS are a fraction of one percent. Nothing here is a
    // fine judgement between neighbouring values.
    const allowance = regressionAllowanceKwh(87335.5, opts);
    expect(150).toBeLessThan(allowance);        // ASHP 2, December 2025
    expect(87335.5).toBeGreaterThan(allowance); // a reset to zero
  });

  it('floors the share, so a small register is not brittle', () => {
    // 1% of a pump that has totalled 3 kWh is 0.03, and a sub-kWh wobble there is
    // the same ordinary correction it would be on a meter a thousand times larger.
    expect(regressionAllowanceKwh(3, opts)).toBe(1);
    expect(regressionAllowanceKwh(0, opts)).toBe(1);
  });

  it('needs a floor large enough for the register it is protecting', () => {
    // EM/015 Chw Pressurisation Unit: the whole register is 3.0 kWh and has not moved
    // since September. A proportional test cannot help it — 1% of 3 is 0.03 — so the
    // floor is the only thing standing between a 3 kWh wobble and two withheld months
    // of ~68,900 and ~64,300 kWh. At the 1 kWh default it fell through; 3CS therefore
    // configures 25, which still cannot mask a reset, since a reset returns the
    // register to about zero rather than to 25 kWh below where it was.
    const dflt = regressionAllowanceKwh(3, opts);
    expect(3).toBeGreaterThan(dflt);
    const floored = estimationOptions({regressionToleranceKwh: 25});
    expect(3).toBeLessThan(regressionAllowanceKwh(3, floored));
    // Above 2,500 kWh the share is the larger of the two, so the floor stops binding.
    expect(regressionAllowanceKwh(2500, floored)).toBe(25);
    expect(regressionAllowanceKwh(87335.5, floored)).toBeCloseTo(873.355, 6);
  });
});

describe('a refused backwards step says how far it fell', () => {
  // The bare reason was not actionable: "accumulator reset near boundary" against a
  // 3 kWh register reads identically to the same words against one that fell 24,000,
  // and the two want opposite responses — widen a threshold, or replace a meter.
  it('carries the drop, the reading it fell from, and the allowance', () => {
    const b = resolveBoundary([sample(-12, 87335.5), sample(36, 85335.5)], at(0), opts);
    expect(b.usage).toBeNull();
    expect(b.reason).toBe(
      'accumulator reset near boundary (fell 2,000 kWh from 87,336; allowance 873 kWh)');
  });

  it('does the same for a span, and keeps small figures readable', () => {
    const at0 = (usage) => ({usage, quality: 'actual', gapFrom: null, gapTo: null, reason: null});
    const d = boundaryDelta(at0(3), at0(0), at(0), at(744), opts);
    expect(d.kwh).toBeNull();
    expect(d.reason).toBe(
      'accumulator reset mid-interval (fell 3 kWh from 3; allowance 1 kWh)');
  });
});

describe('a coarse meter, one tick across weeks', () => {
  // The case that motivated measuring resolution rather than configuring it.
  const before = sample(-240, 1030);
  const after = sample(240, 1046);   // exactly one 16 kWh tick, twenty days later
  const runs = [[sample(-244, 998), sample(-242, 1014), sample(-240, 1030)]];

  it('reads as unrecorded history against the fixed 1 kWh default', () => {
    const b = resolveBoundary([before, after], at(0), opts);
    expect(b.usage).toBe(1030);
    expect(b.quietFrom).toEqual(at(-240));
  });

  it('is not a gap at all once its own resolution is measured', () => {
    const tick = observedTickKwh(runs);
    expect(tick).toBe(16);
    const perMeter = {...opts, idleToleranceKwh: idleToleranceFor({
      observedTick: tick, spanKwh: 100000, opts
    })};
    const b = resolveBoundary([before, after], at(0), perMeter);
    expect(b.quality).toBe('actual');
    expect(b.gapFrom).toBeNull();
    // One tick across twenty days is not missing history either, so the quality
    // view stays quiet too rather than reporting a twenty-day hole.
    expect(b.quietFrom).toBeNull();
  });
});

describe('widestGap', () => {
  it('picks the widest stretch and clips it to the window', () => {
    const g = widestGap([
      {from: at(0), to: at(6)},
      {from: at(10), to: at(40)},
      {from: at(50), to: at(52)}
    ], at(0), at(24));
    // The 30 h gap clipped to the window is 14 h, still the widest.
    expect(g.hours).toBe(14);
    expect(g.from).toEqual(at(10));
    expect(g.to).toEqual(at(24));
  });

  it('is null when nothing overlaps', () => {
    expect(widestGap([{from: at(30), to: at(40)}], at(0), at(24))).toBeNull();
    expect(widestGap([], at(0), at(24))).toBeNull();
  });
});

describe('spanDelta separates an exact total from missing history', () => {
  const actualAt = (usage) => ({usage, quality: 'actual', gapFrom: null, gapTo: null, gapKwh: 0, reason: null});
  const gapAt = (usage, fromH, toH, kwh) =>
    ({usage, quality: 'extrapolated', gapFrom: at(fromH), gapTo: at(toH), gapKwh: kwh, reason: null});

  it('reports an exact total even with months unrecorded in between', () => {
    // Both ends are real readings, so `end - start` is exactly right however
    // little was recorded between them — that is a property of a cumulative
    // accumulator. Only the interior boundary knows history is missing.
    const d = spanDelta(
      [actualAt(0), gapAt(500, 0, 24, 1000), actualAt(1000)], at(0), at(24), opts);
    expect(d.kwh).toBe(1000);
    // Nothing is disclosed as estimated, because the reported total is not.
    expect(d.estimatedKwh).toBe(0);
    expect(d.estimated).toBe(false);
    // But the missing history is still reported, and located.
    expect(d.unrecordedHours).toBe(24);
    expect(d.longestGap.hours).toBe(24);
  });

  it('discloses estimation when an end boundary is the projected one', () => {
    const d = spanDelta([gapAt(250, -12, 12, 400), actualAt(1000)], at(0), at(24), opts);
    expect(d.kwh).toBe(750);
    expect(d.estimatedKwh).toBeCloseTo(200, 6);
    expect(d.estimated).toBe(true);
  });

  it('sharpens an end gap when interior boundaries are supplied', () => {
    // The defect this guards: with only two boundaries the opening one bracketed
    // across every record never fetched, so a narrow hole read as spanning the
    // whole span. Supplying the interior boundary does not change the total, but
    // the gap it reports is the real one.
    const wide = spanDelta([gapAt(100, 0, 240, 2000), actualAt(2100)], at(0), at(240), opts);
    const tight = spanDelta(
      [gapAt(100, 0, 24, 200), actualAt(1000), actualAt(2100)], at(0), at(240), opts);
    expect(wide.longestGap.hours).toBe(240);
    expect(tight.longestGap.hours).toBe(24);
    expect(tight.estimatedKwh).toBeLessThan(wide.estimatedKwh);
  });

  it('is null when the span has no boundaries at all', () => {
    expect(spanDelta([], at(0), at(24), opts).kwh).toBeNull();
  });
});

describe('boundaryDelta', () => {
  const actual = (usage) => ({usage, quality: 'actual', gapFrom: null, gapTo: null, reason: null});

  it('is a plain difference when neither end was estimated', () => {
    const d = boundaryDelta(actual(100), actual(340), at(0), at(24));
    expect(d).toMatchObject({kwh: 240, estimated: false, estimatedHours: 0, estimatedKwh: 0});
  });

  it('is null when either end is unknown', () => {
    expect(boundaryDelta(actual(100), {usage: null, reason: 'nope'}, at(0), at(24)).kwh).toBeNull();
    expect(boundaryDelta({usage: null, reason: 'nope'}, actual(100), at(0), at(24)).kwh).toBeNull();
  });

  it('is null across an accumulator reset, never a clamped zero', () => {
    const d = boundaryDelta(actual(500), actual(20), at(0), at(24));
    expect(d.kwh).toBeNull();
    expect(d.reason).toMatch(/reset/);
  });

  it('reports zero and discloses it when the register merely stepped back', () => {
    // The 3CS December case: EM/048 Ashp 02 ended the month 150 kWh below where it
    // started, 0.17% of its own register. Withholding the month cost all 72,000 kWh
    // metered across the other 57 boards to protect a 150 kWh uncertainty, so the
    // span reports the only defensible total and carries the step as the disclosure.
    const d = boundaryDelta(actual(87335.5), actual(87185.5), at(0), at(744), opts);
    expect(d.kwh).toBe(0);
    expect(d.reason).toBeNull();
    expect(d.estimated).toBe(true);
    expect(d.estimatedKwh).toBeCloseTo(150, 6);
    // The whole span, because the register could have fallen anywhere inside it.
    expect(d.estimatedHours).toBe(744);
  });

  it('still refuses a step back too large to be a correction', () => {
    // 1% of 87,335.5 is 873.4, so this 2,000 kWh step is a reset as far as the
    // dashboard can tell, and the pre-reset total is genuinely unrecoverable.
    const d = boundaryDelta(actual(87335.5), actual(85335.5), at(0), at(744), opts);
    expect(d.kwh).toBeNull();
    expect(d.reason).toMatch(/reset/);
  });

  it('honours a configured regression tolerance', () => {
    const tight = estimationOptions({regressionSharePct: 0, regressionToleranceKwh: 0});
    expect(boundaryDelta(actual(500), actual(499), at(0), at(24), tight).kwh).toBeNull();
    const loose = estimationOptions({regressionSharePct: 50});
    expect(boundaryDelta(actual(500), actual(300), at(0), at(24), loose).kwh).toBe(0);
  });

  it('withholds a step back rather than reporting zero when estimation is off', () => {
    // Reporting 0 substitutes a value nobody measured, so `enabled: false` — which
    // exists to withhold exactly that — has to cover it too.
    const off = estimationOptions({enabled: false});
    const d = boundaryDelta(actual(87335.5), actual(87185.5), at(0), at(744), off);
    expect(d.kwh).toBeNull();
    expect(d.reason).toMatch(/reset/);
  });

  it('charges only the energy that crossed the gap, not a share of the window', () => {
    // A 24 h window whose closing boundary sat in a 12 h gap, half of which falls
    // inside the window. The gap itself carried 20 kWh, so 10 kWh is estimated —
    // regardless of the 240 kWh the window consumed in total.
    const a = actual(0);
    const b = {usage: 240, quality: 'extrapolated', gapFrom: at(18), gapTo: at(30), gapKwh: 20, reason: null};
    const d = boundaryDelta(a, b, at(0), at(24), opts);
    expect(d.kwh).toBe(240);
    expect(d.estimatedHours).toBe(6);
    expect(d.estimatedKwh).toBeCloseTo(10, 10);
    expect(d.estimated).toBe(true);
  });

  it('charges an idle meter\'s silence nothing, however long it is', () => {
    // The defect this replaces: estimatedKwh was the interval's energy times the
    // *time* fraction spent in a gap, so a month half spent in an idle meter's
    // silence and half consuming 400 kWh reported 200 kWh estimated. History
    // records only changes, so the silence provably carried no energy.
    const a = actual(0);
    const b = {usage: 400, quality: 'extrapolated', gapFrom: at(0), gapTo: at(12), gapKwh: 0, reason: null};
    const d = boundaryDelta(a, b, at(0), at(24), opts);
    expect(d.kwh).toBe(400);
    expect(d.estimatedHours).toBe(12);   // the silence is still reported...
    expect(d.estimatedKwh).toBe(0);      // ...but it cost nothing
    expect(d.estimated).toBe(false);     // ...so there is nothing to badge
  });

  it('does not badge a projection below the materiality share', () => {
    // An idle meter that ticks once after a month leaves a one-unit bracket.
    // Bridging it is accurate to that tick, so the month is not "estimated".
    const a = actual(0);
    const b = {usage: 41100, quality: 'extrapolated', gapFrom: at(0), gapTo: at(24), gapKwh: 1, reason: null};
    const d = boundaryDelta(a, b, at(0), at(24), opts);
    expect(d.estimatedKwh).toBeCloseTo(1, 10);
    expect(d.estimated).toBe(false);
    // The figure is still reported in full — only the badge is suppressed.
    expect(d.estimatedHours).toBe(24);
  });

  it('counts a gap that spans the whole interval exactly once', () => {
    // A month lying wholly inside one outage. Both boundaries resolve from the
    // same bracket, so its energy must be charged once: 500 kWh over 200 h, of
    // which this 24 h window holds 60 — which is also all of the window's energy.
    const g = (u) =>
      ({usage: u, quality: 'extrapolated', gapFrom: at(-100), gapTo: at(100), gapKwh: 500, reason: null});
    const d = boundaryDelta(g(250), g(310), at(0), at(24), opts);
    expect(d.kwh).toBe(60);
    expect(d.estimatedKwh).toBeCloseTo(60, 10);
    expect(d.estimated).toBe(true);
  });

  it('never discloses more estimated energy than the interval holds', () => {
    const g = (u, kwh) =>
      ({usage: u, quality: 'extrapolated', gapFrom: at(-100), gapTo: at(100), gapKwh: kwh, reason: null});
    // Deliberately inconsistent inputs, to prove the cap holds regardless.
    const d = boundaryDelta(g(0, 100000), g(10, 100000), at(0), at(24), opts);
    expect(d.estimatedKwh).toBe(10);
  });
});

describe('carrying a bracketed gap forward preserves the total', () => {
  it('moves the energy to the later month without changing their sum', () => {
    // The invariant that makes this change safe to make. Adjacent months share a
    // boundary, so wherever that boundary lands the pair's total is `end - start`
    // and cannot move. A 12-month rating figure telescopes the same way, which is
    // why it is untouched here and only the attribution between two neighbouring
    // months differs from what interpolation produced.
    const samples = normaliseSamples([sample(0, 1000), sample(72, 1720)]);
    const start = at(0);
    const mid = at(36);
    const end = at(72);

    const bStart = resolveBoundary(samples, start, opts);
    const bMid = resolveBoundary(samples, mid, opts);
    const bEnd = resolveBoundary(samples, end, opts);

    expect(bMid.quality).toBe('actual');
    expect(bMid.usage).toBe(1000);

    const first = boundaryDelta(bStart, bMid, start, mid, opts);
    const second = boundaryDelta(bMid, bEnd, mid, end, opts);

    expect(first.kwh + second.kwh).toBeCloseTo(720, 8);
    // All of it lands in the half ending at the later record, because that is
    // where the accumulator was seen to step.
    expect(first.kwh).toBe(0);
    expect(second.kwh).toBeCloseTo(720, 8);
    // Neither half discloses an estimate, because neither rests on one...
    expect(first.estimatedKwh).toBe(0);
    expect(second.estimatedKwh).toBe(0);
    expect(first.estimated).toBe(false);
    expect(second.estimated).toBe(false);
    // ...and both still report the missing history behind them.
    expect(first.unrecordedHours).toBe(36);
    expect(second.unrecordedHours).toBe(36);
  });
});

describe('sumDeltas', () => {
  const d = (kwh, estimatedKwh = 0, estimatedHours = 0) =>
    ({kwh, estimated: estimatedKwh > 0, estimatedHours, estimatedKwh, reason: null});

  it('sums energy and estimated energy', () => {
    const out = sumDeltas([d(100), d(50, 10, 6), d(25, 5, 2)]);
    expect(out.kwh).toBe(175);
    expect(out.estimatedKwh).toBe(15);
    expect(out.estimated).toBe(true);
  });

  it('takes the longest outage, not the sum of them', () => {
    // Two boards down over the same week is one week of degraded data.
    expect(sumDeltas([d(10, 1, 168), d(10, 1, 168)]).estimatedHours).toBe(168);
  });

  it('flags estimation by outage hours, not by estimated energy', () => {
    // A meter that consumed nothing across a gap still had its reading
    // estimated; a zero estimate must not relabel the month as measured.
    const out = sumDeltas([{kwh: 0, estimated: true, estimatedHours: 48, estimatedKwh: 0, reason: null}]);
    expect(out.estimated).toBe(true);
  });

  it('nulls the whole sum when one meter is unreadable', () => {
    const out = sumDeltas([d(100), {kwh: null, estimatedKwh: 0, estimatedHours: 0, reason: 'dead'}]);
    expect(out.kwh).toBeNull();
    expect(out.reason).toBe('dead');
  });

  it('is null for an empty pool, not zero', () => {
    expect(sumDeltas([]).kwh).toBeNull();
    expect(sumDeltas(null).kwh).toBeNull();
  });

  it('reports a month one meter stepped back over, and badges it estimated', () => {
    // December 2025 at 3CS, end to end. 57 boards summed to about 72,000 kWh and
    // ASHP 2's register ended 150 kWh below where it started, which nulled the pool
    // and blanked the whole month. Now the month reports, amber, disclosing 150 kWh
    // — the badge is what keeps the understatement from travelling unmarked.
    const others = d(72004.5);
    const ashp2 = {kwh: 0, estimated: true, estimatedHours: 744, estimatedKwh: 150, reason: null};
    const out = sumDeltas([others, ashp2]);
    expect(out.kwh).toBe(72004.5);
    expect(out.estimated).toBe(true);
    expect(out.estimatedKwh).toBe(150);
    expect(out.reason).toBeNull();
  });
});

describe('estimatedSharePct', () => {
  it('is estimated energy over total energy', () => {
    expect(estimatedSharePct([{kwh: 900, estimatedKwh: 0}, {kwh: 100, estimatedKwh: 50}]))
      .toBeCloseTo(5, 10);
  });

  it('skips unknown items rather than treating them as zero', () => {
    expect(estimatedSharePct([{kwh: null, estimatedKwh: 0}, {kwh: 100, estimatedKwh: 25}]))
      .toBeCloseTo(25, 10);
  });

  it('is null when there is no energy to take a share of', () => {
    expect(estimatedSharePct([])).toBeNull();
    expect(estimatedSharePct([{kwh: 0, estimatedKwh: 0}])).toBeNull();
  });
});

describe('formatGap', () => {
  it('uses hours below two days and days above', () => {
    expect(formatGap(6)).toBe('6 h');
    expect(formatGap(47)).toBe('47 h');
    expect(formatGap(60)).toBe('2.5 days');
    expect(formatGap(24 * 30)).toBe('30 days');
  });

  it('is empty for nothing to report', () => {
    expect(formatGap(0)).toBe('');
    expect(formatGap(NaN)).toBe('');
  });
});
