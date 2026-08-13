import {describe, it, expect, beforeEach, vi} from 'vitest';
import {ref} from 'vue';
import {setActivePinia, createPinia} from 'pinia';
import {intensityForStars} from '@/util/nabersRating.js';

// The store reaches for gRPC; the transport is stubbed and driven by `_meters`,
// a per-test fixture. A meter absent from the fixture behaves like one with no
// history and no current reading.
const _meters = new Map();

/** Highest number of concurrent stub calls seen, for the limiter tests. */
let _peakInFlight = 0;
let _inFlight = 0;

/**
 * @param {function(): *} body
 * @return {Promise<*>}
 */
async function tracked(body) {
  _inFlight++;
  _peakInFlight = Math.max(_peakInFlight, _inFlight);
  try {
    await Promise.resolve();
    return body();
  } finally {
    _inFlight--;
  }
}

/**
 * A date as yyyy-mm-dd in *local* time.
 *
 * Not `toISOString`, which is UTC: the store's month boundaries are local
 * midnights, so during BST a UTC rendering shifts them back to the previous day
 * and no fixture would ever match.
 *
 * @param {Date} d
 * @return {string}
 */
function localDay(d) {
  const p = n => String(n).padStart(2, '0');
  return `${d.getFullYear()}-${p(d.getMonth() + 1)}-${p(d.getDate())}`;
}

/**
 * The inverse of {@link localDay}: local midnight on that date.
 *
 * @param {string} day yyyy-mm-dd
 * @return {Date}
 */
function fromLocalDay(day) {
  const [y, m, d] = day.split('-').map(Number);
  return new Date(y, m - 1, d);
}

const DAY_MS = 24 * 60 * 60 * 1000;

/**
 * A fixture key `n` days before today, in local time.
 *
 * Gaps have to be positioned relative to now, not to a fixed date: the boundary
 * reader only looks `searchWindowDays` either side of an instant, so a fixture
 * pinned to a calendar date drifts out of reach as the suite ages.
 *
 * @param {number} n
 * @return {string} yyyy-mm-dd
 */
function daysAgo(n) {
  return localDay(new Date(Date.now() - n * DAY_MS));
}

/**
 * A fixture's recorded history, as a timestamped series.
 *
 * Only the `at` map: history and the live trait are separate sources, and
 * conflating them makes the case that matters untestable. `pkg/auto/history`
 * records a reading only when it changes, so an idle meter has a *live* value and
 * no recent history — exactly the shape `current` alone describes. An earlier
 * version of this helper appended `current` here as a record a minute old, which
 * made every idle meter look like it had just reported.
 *
 * @param {{at?: Object<string, number>, current?: number}} [m]
 * @return {Array<{usage: number, at: Date}>} sorted ascending
 */
function meterSamples(m) {
  // `series` is for fixtures too dense to express as one entry per day — a real
  // meter reports every fifteen minutes, and that turned out to be the case the
  // day-granularity map could not describe and so never tested.
  if (m?.series) return m.series;
  return Object.entries(m?.at ?? {})
    .map(([day, usage]) => ({at: fromLocalDay(day), usage}))
    .sort((a, b) => a.at.getTime() - b.at.getTime());
}

/**
 * A meter recording every `everyHours` for the last `days`, optionally with a
 * stretch missing.
 *
 * @param {Object} spec
 * @param {number} spec.days
 * @param {number} spec.everyHours
 * @param {number} [spec.holeFrom] days ago the hole opens
 * @param {number} [spec.holeTo] days ago it closes
 * @return {{series: Array<{usage: number, at: Date}>, current: number}}
 */
function denseMeter({days, everyHours, holeFrom = null, holeTo = null}) {
  const now = Date.now();
  const series = [];
  let usage = 10000;
  for (let t = now - days * DAY_MS; t <= now; t += everyHours * 60 * 60 * 1000) {
    usage += everyHours * 2;
    const inHole = holeFrom !== null &&
      t >= now - holeFrom * DAY_MS && t <= now - holeTo * DAY_MS;
    if (!inHole) series.push({at: new Date(t), usage});
  }
  return {series, current: usage};
}

vi.mock('@/api/sc/traits/meter.js', () => ({
  getMeterReading: vi.fn(name => tracked(() => {
    const m = _meters.get(name);
    if (m?.throws) throw m.throws;
    return m?.current == null ? {} : {usage: m.current};
  })),
  getFirstMeterReadingInPeriod: vi.fn((name, startTime) => tracked(() => {
    const m = _meters.get(name);
    if (m?.throws) throw m.throws;
    const usage = m?.at?.[localDay(startTime)];
    return usage == null ? null : {meterReading: {usage}};
  })),
  getMeterReadingBefore: vi.fn((name, at, windowDays) => tracked(() => {
    const m = _meters.get(name);
    if (m?.throws) throw m.throws;
    const lo = at.getTime() - windowDays * DAY_MS;
    const hits = meterSamples(m)
      .filter(s => s.at.getTime() <= at.getTime() && s.at.getTime() >= lo);
    return hits.length ? hits[hits.length - 1] : null;
  })),
  // A run of the last `limit` readings, ascending, as the real API returns. The
  // run behind the boundary is what measures the meter's resolution, so a mock
  // handing back only the nearest record would make that untestable.
  getMeterReadingsBefore: vi.fn((name, at, windowDays, limit) => tracked(() => {
    const m = _meters.get(name);
    if (m?.throws) throw m.throws;
    const lo = at.getTime() - windowDays * DAY_MS;
    const hits = meterSamples(m)
      .filter(s => s.at.getTime() <= at.getTime() && s.at.getTime() >= lo);
    return hits.slice(Math.max(0, hits.length - Math.max(1, limit)));
  })),
  getMeterReadingAfter: vi.fn((name, at, windowDays) => tracked(() => {
    const m = _meters.get(name);
    if (m?.throws) throw m.throws;
    const hi = at.getTime() + windowDays * DAY_MS;
    return meterSamples(m)
      .find(s => s.at.getTime() >= at.getTime() && s.at.getTime() <= hi) ?? null;
  }))
}));

// Device metadata, keyed by name, in the shape `Metadata.toObject()` returns: a
// `moreMap` of entry pairs rather than an object. A name absent from the fixture
// behaves like a device with no metadata, which is the fallback path.
const _metadata = new Map();

vi.mock('@/api/sc/traits/metadata.js', () => ({
  getMetadata: vi.fn(name => tracked(() => {
    const m = _metadata.get(name);
    if (!m) throw new Error(`no metadata for ${name}`);
    return m;
  }))
}));

/**
 * Register a device's metadata as the wire presents it.
 *
 * @param {Object<string, {title?: string, ref?: string, floor?: string, zone?: string}>} spec
 */
function givenMetadata(spec) {
  for (const [name, m] of Object.entries(spec)) {
    _metadata.set(name, {
      appearance: {title: m.title ?? '', description: ''},
      location:   {floor: m.floor ?? '', zone: m.zone ?? '', title: '', description: ''},
      moreMap:    m.ref ? [['ref', m.ref]] : []
    });
  }
}

const _uiConfig = ref({});
vi.mock('./uiConfig.js', () => ({
  useUiConfigStore: () => ({
    get config() {
      return _uiConfig.value;
    }
  })
}));

const {
  useNabersBaseBuildingStore, BB_CATEGORIES, humanizeCategory, mapLimit, splitMeterName,
  meterIdentity
} = await import('./nabersBaseBuildingMetrics.js');

/**
 * @param {Object} nabersBaseBuilding
 * @return {Object} the store, configured
 */
function storeWith(nabersBaseBuilding) {
  _uiConfig.value = {nabersBaseBuilding};
  return useNabersBaseBuildingStore();
}

/**
 * Register meter fixtures.
 *
 * @param {Object<string, {current?: number, at?: Object<string, number>, throws?: *}>} spec
 */
function givenMeters(spec) {
  for (const [name, m] of Object.entries(spec)) _meters.set(name, m);
}

/**
 * The 13 month-boundary dates `refreshMonthly` reads, as local yyyy-mm-dd.
 *
 * @return {string[]}
 */
function boundaryDates() {
  const now = new Date();
  return Array.from({length: 13}, (_, i) =>
    localDay(new Date(now.getFullYear(), now.getMonth() - (12 - i), 1)));
}

/**
 * A meter that reads cleanly at every month boundary, rising by `perMonth`.
 *
 * @param {number} perMonth
 * @param {number} [base]
 * @return {{current: number, at: Object<string, number>}}
 */
function risingMeter(perMonth, base = 1000) {
  const at = {};
  boundaryDates().forEach((d, i) => {
    at[d] = base + i * perMonth;
  });
  return {current: base + 13 * perMonth, at};
}

/**
 * A meter whose register genuinely resets at `atIndex` and climbs from zero after.
 *
 * Two things make this a *reset* rather than a dropout, and both matter or the
 * fixture tests something else entirely:
 *
 * - **It does not recover quickly.** `withoutDropouts` discards a fall that the very
 *   next reading undoes, because the meter would have had to consume its whole
 *   lifetime total again to do that. A reset climbs back over months, so the readings
 *   after it stay below the old peak for longer than one dropout's worth.
 * - **It does eventually exceed the old peak**, so `current` sits above it. Otherwise
 *   the pre-reset readings are all above the live value and the ceiling filter
 *   discards them first, which is a different failure with a different reason.
 *
 * @param {number} perMonth after the reset; steep, so the old peak is passed in time
 * @param {number} base
 * @param {number} atIndex boundary the reset lands on
 * @return {{current: number, at: Object<string, number>}}
 */
function resetMeter(perMonth, base, atIndex) {
  const at = {};
  const dates = boundaryDates();
  dates.forEach((d, i) => {
    at[d] = i < atIndex ? base + i * 10 : (i - atIndex) * perMonth;
  });
  return {current: (dates.length - atIndex) * perMonth, at};
}

/**
 * A meter that steps back a little at `atIndex` and stays down, climbing slowly.
 *
 * The narrow case the regression tolerance is actually for, once dropouts are
 * filtered: a register that is corrected downwards and carries on from there, rather
 * than one bad reading. Slowly, so the step survives the dropout filter.
 *
 * @param {number} perMonth before the step
 * @param {number} base
 * @param {number} atIndex
 * @param {number} stepKwh how far back it goes
 * @return {{current: number, at: Object<string, number>}}
 */
function steppedBackMeter(perMonth, base, atIndex, stepKwh) {
  const at = {};
  const dates = boundaryDates();
  const floor = base + (atIndex - 1) * perMonth - stepKwh;
  dates.forEach((d, i) => {
    at[d] = i < atIndex ? base + i * perMonth : floor + (i - atIndex) * 50;
  });
  return {current: floor + (dates.length - atIndex) * 50, at};
}

/**
 * The end uses a building can meter separately is a property of the building,
 * so the category set comes from config. These tests pin the two shapes that
 * exist in production — a six-category site and an eleven-category one — and
 * the failure mode that matters: a category silently dropped from the set is a
 * category silently dropped from the rating.
 */
describe('base building category derivation', () => {
  beforeEach(() => {
    setActivePinia(createPinia());
    _uiConfig.value = {};
    _meters.clear();
    _metadata.clear();
    _peakInFlight = 0;
    _inFlight = 0;
  });

  it('falls back to BB_CATEGORIES when config supplies neither meters nor targets', () => {
    expect(storeWith({enabled: true}).categories).toEqual(BB_CATEGORIES);
  });

  it('reproduces BB_CATEGORIES exactly for a six-category config', () => {
    // Verbatim key set and order from a deployed site's ui-config.json.
    const store = storeWith({
      enabled: true,
      meterNames: {
        hvac:               [],
        lifts:              [],
        commonAreaLighting: [],
        exteriorLighting:   [],
        carPark:            [],
        smallPower:         [],
        pvGeneration:       [],
        pvExport:           []
      },
      dfpTargets: {hvac: 42.6, lifts: 5.6, commonAreaLighting: 7.0, smallPower: 1.9, total: 55.7, totalGross: 59.1}
    });
    expect(store.categories).toEqual(BB_CATEGORIES);
  });

  it('derives eleven end uses in config order for a full-DfP-method site', () => {
    const store = storeWith({
      enabled: true,
      meterNames: {
        lighting:         ['z/meters-lighting'],
        lifts:            ['z/meters-lifts'],
        smallPower:       ['z/meters-small-power'],
        server:           ['z/meters-server'],
        other:            [],
        dhw:              ['z/meters-dhw'],
        centralAhu:       ['z/meters-central-ahu'],
        terminalUnitFans: ['z/meters-fcus'],
        pumps:            ['z/meters-pumps'],
        coolingHeating:   ['z/meters-cah'],
        dehum:            ['z/meters-dehumidifiers'],
        pvGeneration:     ['z/meters-generation'],
        pvExport:         []
      }
    });
    expect(store.categories).toEqual([
      'lighting', 'lifts', 'smallPower', 'server', 'other', 'dhw',
      'centralAhu', 'terminalUnitFans', 'pumps', 'coolingHeating', 'dehum'
    ]);
    // `other` has no meter, so it draws its target bar but is not awaited.
    expect(store.configuredCategories).toHaveLength(10);
  });

  it('excludes generation and roll-up keys from the end uses', () => {
    const store = storeWith({
      enabled:    true,
      meterNames: {hvac: ['a'], pvGeneration: ['b'], pvExport: ['c']},
      dfpTargets: {hvac: 1, total: 2, totalGross: 3}
    });
    expect(store.categories).toEqual(['hvac']);
  });

  it('excludes _-prefixed provenance keys, which this config family uses as comments', () => {
    const store = storeWith({
      enabled:    true,
      meterNames: {_meterNamesNote: 'not a category', hvac: ['a']},
      dfpTargets: {_dfpTargetsNote: 'nor this', lifts: 2}
    });
    expect(store.categories).toEqual(['hvac', 'lifts']);
  });

  it('keeps a target-only category so its reference bar still draws', () => {
    const store = storeWith({
      enabled:    true,
      meterNames: {hvac: ['a']},
      dfpTargets: {hvac: 1, other: 7.95}
    });
    expect(store.categories).toContain('other');
    expect(store.configuredCategories).toEqual(['hvac']);
  });

  it('feeds every derived category into the rating, not just the built-in six', () => {
    // The regression this guards: `refreshMonthly` and `refresh` once flattened
    // meters off the hardcoded list, so a meter under an unrecognised key was
    // dropped from the total with no null and no warning — a flattering rating.
    const store = storeWith({
      enabled:    true,
      meterNames: {centralAhu: ['z/ahu'], pumps: ['z/pumps'], hvac: ['z/hvac']}
    });
    expect(store.categories.flatMap(c => store.bbCfg.meterNames[c] ?? []))
      .toEqual(['z/ahu', 'z/pumps', 'z/hvac']);
  });
});

describe('category labels', () => {
  beforeEach(() => {
    setActivePinia(createPinia());
    _uiConfig.value = {};
    _meters.clear();
    _metadata.clear();
    _peakInFlight = 0;
    _inFlight = 0;
  });

  it('labels both the six-category and eleven-category key sets', () => {
    const store = storeWith({enabled: true});
    expect(store.categoryLabels.commonAreaLighting).toBe('Common Lighting');
    expect(store.categoryLabels.terminalUnitFans).toBe('Terminal Fans');
    expect(store.categoryLabels.coolingHeating).toBe('Cooling + Heating');
    // Previously unlabelled, so it rendered as a raw key in the quality table.
    expect(store.categoryLabels.pvExport).toBe('PV Export');
  });

  it('lets config override a label', () => {
    const store = storeWith({enabled: true, categoryLabels: {terminalUnitFans: 'FCUs'}});
    expect(store.categoryLabels.terminalUnitFans).toBe('FCUs');
    expect(store.categoryLabels.lifts).toBe('Lifts');
  });

  it('humanizes an unknown key rather than showing camelCase', () => {
    expect(humanizeCategory('carParkVentilation')).toBe('Car Park Ventilation');
    expect(humanizeCategory('dhw')).toBe('Dhw');
  });
});

/**
 * When an end use is a single zone aggregate, "any meter unreadable" and "all
 * meters unreadable" are the same condition. Over seventeen distribution boards
 * they are not, and the difference decides whether a dead board shows up as an
 * honest gap or as a quietly lower, and therefore better looking, rating.
 */
describe('strict sums across many meters in one category', () => {
  const FCUS = Array.from({length: 17}, (_, i) => `bldg-1/floors/0${(i % 9) + 1}/devices/db-ll-x${i}`);
  const ANCHOR = '2026-01-01';

  beforeEach(() => {
    setActivePinia(createPinia());
    _uiConfig.value = {};
    _meters.clear();
    _peakInFlight = 0;
    _inFlight = 0;
    FCUS.forEach(n => givenMeters({[n]: {at: {[ANCHOR]: 100}, current: 110}}));
  });

  /** @return {Object} the store, with seventeen FCU boards and an unmetered end use */
  function fcuStore() {
    return storeWith({
      enabled:           true,
      ratingPeriodStart: ANCHOR,
      nia:               16903,
      meterNames:        {terminalUnitFans: FCUS, other: [], pvGeneration: [], pvExport: []}
    });
  }

  it('sums the category when every meter reads', async () => {
    const store = fcuStore();
    await store.refresh();
    expect(store.categoryPeriodKwh.terminalUnitFans).toBe(17 * 10);
    expect(store.unreadableCategories).toEqual([]);
    expect(store.meterFailures).toEqual([]);
  });

  it('nulls the whole category when a single meter of seventeen is unreadable', async () => {
    _meters.delete(FCUS[4]);
    const store = fcuStore();
    await store.refresh();
    // Not 160. A partial total understates consumption and flatters the rating.
    expect(store.categoryPeriodKwh.terminalUnitFans).toBeNull();
    expect(store.unreadableCategories).toContain('terminalUnitFans');
  });

  it('names the failing board, not just the end use', async () => {
    _meters.delete(FCUS[4]);
    const store = fcuStore();
    await store.refresh();
    expect(store.meterFailures).toHaveLength(1);
    expect(store.meterFailures[0]).toMatchObject({name: FCUS[4], category: 'terminalUnitFans'});
    expect(store.unreadableMeters).toEqual([FCUS[4]]);
    expect(store.unreadableMeterLabels[0]).toContain('db-ll-x4');
  });

  it('does not blank the section when a meter errors rather than merely missing', async () => {
    givenMeters({[FCUS[2]]: {throws: {code: 4, message: ''}}}); // DEADLINE_EXCEEDED
    const store = fcuStore();
    await store.refresh();
    // This used to reject the top-level Promise.all and set `error`, which
    // renders an alert in place of the entire dashboard.
    expect(store.error).toBeNull();
    expect(store.meterFailures).toHaveLength(1);
    expect(store.meterFailures[0].reason).toBeTruthy();
  });

  it('keeps an unmetered category at 0 and out of the unreadable list', async () => {
    const store = fcuStore();
    await store.refresh();
    // `other` is deliberately unmetered: its DfP target still draws a reference
    // bar, but it is "nothing here", not "unknown".
    expect(store.categoryPeriodKwh.other).toBe(0);
    expect(store.unreadableCategories).not.toContain('other');
  });
});

describe('per-meter monthly deltas', () => {
  const A = 'site/devices/a';
  const B = 'site/devices/b';

  beforeEach(() => {
    setActivePinia(createPinia());
    _uiConfig.value = {};
    _meters.clear();
  });

  /**
   * The store with two lighting meters.
   *
   * Carries a postcode and rated hours so a rating can actually compute: without
   * them `standingRating` is null whatever the meter data does, which would make
   * "the gap withheld the rating" indistinguishable from "the config did".
   *
   * @param {Object} [extra] merged over the config
   * @return {Object}
   */
  function twoMeterStore(extra = {}) {
    return storeWith({
      enabled:    true,
      nia:        1000,
      postcode:   'WD25 9NH',
      ratedHours: 60,
      meterNames: {lighting: [A, B], pvGeneration: [], pvExport: []},
      ...extra
    });
  }

  it('sums per-meter monthly consumption', async () => {
    givenMeters({[A]: risingMeter(10), [B]: risingMeter(5)});
    const store = twoMeterStore();
    await store.refreshMonthly();
    expect(store.monthlyData).toHaveLength(12);
    expect(store.monthlyData.every(m => m.grossKwh === 15)).toBe(true);
    expect(store.monthsOfData).toBe(12);
  });

  it('carries a bracketed gap forward, attributing its energy to the later month', async () => {
    const b = boundaryDates();
    const gappy = risingMeter(10);
    delete gappy.at[b[3]];
    givenMeters({[A]: gappy, [B]: risingMeter(5)});
    const store = twoMeterStore();
    await store.refreshMonthly();

    // Both months keep a figure, and both are measured rather than estimated:
    // history records only changes, so A's accumulator held its boundary-2 value
    // until the record at boundary 4 — that is a measurement, not a substitution.
    expect(store.monthlyData[2].quality).toBe('actual');
    expect(store.monthlyData[3].quality).toBe('actual');
    expect(store.monthlyData[2].estimatedKwh).toBe(0);

    // The pair's total is exactly right, being `end - start` across two contiguous
    // months. All of A's 20 kWh lands in the month ending at its next record rather
    // than being spread across both by elapsed time, which is what interpolation
    // did when it reported roughly 15 and 15.
    expect(store.monthlyData[2].grossKwh + store.monthlyData[3].grossKwh).toBe(30);
    expect(store.monthlyData[2].grossKwh).toBe(5);    // B's 5 alone
    expect(store.monthlyData[3].grossKwh).toBe(25);   // B's 5, plus all of A's 20

    // Untouched months stay exact, and stay labelled actual.
    expect(store.monthlyData[1].grossKwh).toBe(15);
    expect(store.monthlyData[4].grossKwh).toBe(15);
    expect(store.monthlyData[1].quality).toBe('actual');

    // The gap still does not withhold the rating, and now discloses nothing
    // either, because no figure on screen rests on a substituted value.
    expect(store.monthsOfData).toBe(12);
    expect(store.hasFullRatingPeriod).toBe(true);
    expect(store.standingRating).not.toBeNull();
    expect(store.monthlyEstimatedSharePct).toBe(0);
    expect(store.hasEstimatedData).toBe(false);
  });

  it('still reports those months when estimation is switched off', async () => {
    // `estimation.enabled` withholds figures resting on a value nobody measured. A
    // carried-forward boundary is not one of those — it follows from on-change
    // recording — so turning the flag off must not turn a measurement into a
    // missing month. What it still governs is projecting an unreachable meter.
    const b = boundaryDates();
    const gappy = risingMeter(10);
    delete gappy.at[b[3]];
    givenMeters({[A]: gappy, [B]: risingMeter(5)});
    const store = twoMeterStore({estimation: {enabled: false}});
    await store.refreshMonthly();
    expect(store.monthlyData[2].grossKwh).toBe(5);
    expect(store.monthlyData[3].grossKwh).toBe(25);
    expect(store.monthlyData[2].quality).toBe('actual');
    expect(store.monthlyData[1].grossKwh).toBe(15);
    expect(store.monthsOfData).toBe(12);
    expect(store.hasFullRatingPeriod).toBe(true);
    expect(store.hasEstimatedData).toBe(false);
  });

  it('leaves the 12-month total untouched wherever the gaps fall', async () => {
    // Why changing the boundary rule cannot move a rating, and the guard that keeps
    // it that way. Each month is `end - start` and consecutive months share a
    // boundary, so a 12-month sum telescopes to the last boundary minus the first.
    // Where the interior boundaries land — exactly hit or carried forward across
    // three separate holes — cancels out of the total entirely.
    const b = boundaryDates();

    givenMeters({[A]: risingMeter(10), [B]: risingMeter(5)});
    let store = twoMeterStore();
    await store.refreshMonthly();
    const exactTotal = store.monthlyData.reduce((acc, m) => acc + m.grossKwh, 0);
    const exactStars = store.standingRating.stars;
    expect(exactTotal).toBe(180);

    // Three missing boundaries, two of them adjacent so a whole month is bracketed
    // by a single pair of records.
    const gappy = risingMeter(10);
    delete gappy.at[b[3]];
    delete gappy.at[b[7]];
    delete gappy.at[b[8]];

    setActivePinia(createPinia());
    _meters.clear();
    givenMeters({[A]: gappy, [B]: risingMeter(5)});
    store = twoMeterStore();
    await store.refreshMonthly();

    expect(store.monthlyData.reduce((acc, m) => acc + m.grossKwh, 0)).toBe(exactTotal);
    // And the figure the rating is actually computed from, which is the net sum.
    expect(store.monthlyData.reduce((acc, m) => acc + m.netKwh, 0)).toBe(180);
    expect(store.standingRating.stars).toBe(exactStars);
    // The individual months did move, which is the whole point — this is not
    // passing because the fixture happened to resolve identically.
    expect(store.monthlyData[7].grossKwh).not.toBe(15);
  });

  it('treats a genuinely reset register as unknown, not a zero-consumption month', async () => {
    givenMeters({[A]: resetMeter(400, 1000, 6), [B]: risingMeter(5)});
    const store = twoMeterStore();
    await store.refreshMonthly();
    expect(store.monthlyData[5].grossKwh).toBeNull();
    expect(store.monthlyData[5].hasData).toBe(false);
  });

  it('repairs a single dropout instead of reading it as a reset', async () => {
    // The 3CS Sep/Oct/Nov 2025 case, and by far the commonest corruption on site: a
    // driver returning a bare 0 in the middle of a healthy series. It is neither
    // negative nor above the current value, so the ceiling filter cannot see it, and
    // it does more damage than a spike precisely because it does not look like
    // corruption — it looks like a reset, so the month is withheld and a working
    // board is reported as faulty.
    //
    // What makes it impossible is the recovery, not the size: for the 0 to be real,
    // the meter would have to consume its whole lifetime total again before the next
    // reading. So the record is DISCARDED and the month comes out exact — nothing
    // substituted, nothing to disclose.
    const b = boundaryDates();
    const dropout = risingMeter(10);
    dropout.at[b[6]] = 0;
    givenMeters({[A]: dropout, [B]: risingMeter(5)});
    const store = twoMeterStore();
    await store.refreshMonthly();

    expect(store.monthlyData[5].hasData).toBe(true);
    expect(store.monthlyData[5].quality).toBe('actual');
    expect(store.monthlyData[5].estimatedKwh).toBe(0);
    // The pair either side of the discarded record still totals exactly what it
    // should: two months of both meters is 2 * (10 + 5).
    expect(store.monthlyData[5].grossKwh + store.monthlyData[6].grossKwh).toBe(30);
    expect(store.monthsOfData).toBe(12);
    expect(store.hasFullRatingPeriod).toBe(true);
    expect(store.standingRating).not.toBeNull();
  });

  it('names the board and the reason on a month it cannot report', async () => {
    // Without this the table showed a bare "✗ Missing" against a month the site's
    // own month-end report had figures for every meter in, which reads as the
    // dashboard being broken rather than as one named board having done something a
    // cumulative meter cannot do.
    givenMeters({[A]: resetMeter(400, 1000, 6), [B]: risingMeter(5)});
    const store = twoMeterStore();
    await store.refreshMonthly();
    expect(store.monthlyData[5].reason).toMatch(/reset/);
    expect(store.monthlyData[5].unreadableMeters).toEqual([A]);
    // And a month that reported carries neither.
    expect(store.monthlyData[1].reason).toBeNull();
    expect(store.monthlyData[1].unreadableMeters).toEqual([]);
  });

  it('gives every failed board its own reason, not the first one shared', async () => {
    // September 2025 at 3CS listed 17 boards against a single reason, because
    // `sumDeltas` short-circuits. The 17 had not all done the same thing, and the one
    // that mattered was indistinguishable from the sixteen that did not.
    const b = boundaryDates();
    const reset = resetMeter(400, 1000, 6);   // a real reset, which does not recover
    const late = risingMeter(5);
    delete late.at[b[0]];            // and a different failure, at the earliest boundary
    delete late.at[b[1]];
    delete late.at[b[2]];
    delete late.at[b[3]];
    delete late.at[b[4]];
    delete late.at[b[5]];
    delete late.at[b[6]];
    givenMeters({[A]: reset, [B]: late});
    const store = twoMeterStore();
    await store.refreshMonthly();

    const failed = store.monthlyData.find(m => (m.failures ?? []).length > 1);
    expect(failed).toBeDefined();
    expect(failed.failures.map(f => f.name).sort()).toEqual([A, B].sort());
    // Two boards, two distinct explanations - which is the whole point.
    const reasons = new Set(failed.failures.map(f => f.reason));
    expect(reasons.size).toBe(2);
    // And the drop is quantified, so "widen the threshold" and "replace the meter"
    // can be told apart without reading the raw history.
    const resetFailure = failed.failures.find(f => f.name === A);
    if (/reset/.test(resetFailure.reason)) {
      expect(resetFailure.reason).toMatch(/fell [\d,.]+ kWh from [\d,.]+; allowance [\d,.]+ kWh/);
    }
  });

  it('reports a month one meter merely stepped back over, and badges it', async () => {
    // A register corrected downwards that then carries on from the lower value, which
    // is the narrow case the regression tolerance is for once dropouts are filtered
    // out. 150 kWh against an 80,000 kWh register is 0.19%, so it cannot be allowed
    // to withhold the month; a reset still is, per the test above.
    givenMeters({[A]: steppedBackMeter(4000, 60000, 6, 150), [B]: risingMeter(4000, 5000)});
    const store = twoMeterStore();
    await store.refreshMonthly();

    const dec = store.monthlyData[5];
    expect(dec.hasData).toBe(true);
    // B's 4000 in full; A contributes the only defensible figure, zero.
    expect(dec.grossKwh).toBe(4000);
    expect(dec.netKwh).toBe(4000);
    expect(dec.totalIntensity).toBe(4);
    expect(dec.reason).toBeNull();

    // Disclosed, not silently absorbed. This is the part that keeps an
    // understatement from being quoted as a measurement.
    expect(dec.quality).toBe('estimated');
    expect(dec.estimatedKwh).toBeCloseTo(150, 6);
    expect(dec.estimatedMeters).toEqual([A]);
    expect(store.hasEstimatedData).toBe(true);

    // Neighbouring months are untouched, and the rating can settle again.
    expect(store.monthlyData[4].grossKwh).toBe(8000);
    expect(store.monthlyData[4].quality).toBe('actual');
    expect(store.monthlyData[6].quality).toBe('actual');
    expect(store.monthsOfData).toBe(12);
    expect(store.hasFullRatingPeriod).toBe(true);
    expect(store.standingRating).not.toBeNull();
  });
});

/**
 * The situation this feature exists for: on site, boards drop out for days at a
 * time and the strict null propagation blanked the headline figure entirely.
 */
describe('gap filling over the rating period', () => {
  const A = 'site/devices/a';
  const B = 'site/devices/b';
  const ANCHOR = '2026-01-01';

  beforeEach(() => {
    setActivePinia(createPinia());
    _uiConfig.value = {};
    _meters.clear();
  });

  /**
   * @param {Object} [extra]
   * @return {Object}
   */
  function store2(extra = {}) {
    return storeWith({
      enabled:           true,
      ratingPeriodStart: ANCHOR,
      nia:               1000,
      postcode:          'WD25 9NH',
      ratedHours:        60,
      meterNames:        {lighting: [A, B], pvGeneration: [], pvExport: []},
      ...extra
    });
  }

  it('reports no estimation when every meter reads cleanly', async () => {
    givenMeters({
      [A]: {at: {[ANCHOR]: 100}, current: 200},
      [B]: {at: {[ANCHOR]: 50}, current: 90}
    });
    const store = store2();
    await store.refresh();
    expect(store.categoryPeriodKwh.lighting).toBe(140);
    expect(store.hasEstimatedData).toBe(false);
    expect(store.periodEstimatedSharePct).toBe(0);
    expect(store.meterEstimates).toEqual([]);
  });

  it('reports a reachable but idle meter as zero, not as invented consumption', async () => {
    // The regression that prompted this. `pkg/auto/history` records a meter
    // reading only when it changes, and for a BACnet meter — whose driver never
    // sets end_time and dedupes on its own account too — an idle meter writes
    // nothing at all, indefinitely. B last moved 31 days ago and its live reading
    // still matches, so it has genuinely consumed nothing.
    //
    // Projecting B's historic rate across that silence reported +22.7 kWh of
    // energy that never existed, inflating consumption and depressing the rating.
    givenMeters({
      [A]: {at: {[ANCHOR]: 100}, current: 200},
      [B]: {at: {[ANCHOR]: 4980, [daysAgo(31)]: 5000}, current: 5000}
    });
    const store = store2();
    await store.refresh();

    // 100 from A, and from B exactly the 20 kWh it really accumulated before
    // going quiet — the 31-day silence adds nothing. Not "less than 22.7": the
    // silence contributes precisely zero.
    expect(store.categoryPeriodKwh.lighting).toBe(120);
    expect(store.unreadableCategories).toEqual([]);
    // Nothing was estimated, so nothing is disclosed. An idle car park meter must
    // not make the dashboard claim estimation every day of the year.
    expect(store.hasEstimatedData).toBe(false);
    expect(store.meterEstimates).toEqual([]);
    // It is surfaced as idle instead, which is neither a failure nor an estimate.
    expect(store.meterIdle).toEqual([B]);
  });

  it('gives an exact period figure when a reachable meter lost history writes', async () => {
    // Distinct from idle: the live reading is higher than the last record, so
    // writes were lost rather than deduped. Both ends of the period are real
    // readings, so the total is exact and there is nothing to disclose — only the
    // *distribution* across the silence is unknown, which a period-to-date figure
    // does not depend on. The monthly table is where that surfaces.
    givenMeters({
      [A]: {at: {[ANCHOR]: 100}, current: 200},
      [B]: {at: {[ANCHOR]: 4980, [daysAgo(31)]: 5000}, current: 5400}
    });
    const store = store2();
    await store.refresh();

    // 100 from A, 420 from B — the real accumulator movement, not a projection.
    expect(store.categoryPeriodKwh.lighting).toBe(520);
    expect(store.hasEstimatedData).toBe(false);
    expect(store.meterEstimates).toEqual([]);
    expect(store.meterIdle).toEqual([]);
  });

  it('attributes a lost-writes silence to the month holding the later record', async () => {
    // The same lost-writes meter, split across months. Interpolation apportioned
    // its movement across every month the silence touched and badged them all
    // estimated; carry-forward holds the accumulator at its last recorded value, so
    // the energy lands where the meter was next seen to move and no month claims an
    // estimate.
    //
    // This is the case where the assumption behind carry-forward is load-bearing.
    // If those writes were lost while the meter was consuming steadily, the earlier
    // months are understated and the later one overstated by the same amount. Two
    // things bound that: the error cancels in any total spanning both, and the
    // silence is still reported as unrecorded history, asserted below.
    givenMeters({
      [A]: risingMeter(10),
      [B]: {at: {[ANCHOR]: 4980, [daysAgo(70)]: 5000}, current: 5400}
    });
    const store = store2();
    await store.refreshMonthly();
    await store.refresh();

    expect(store.monthlyData.some(m => m.quality === 'estimated')).toBe(false);
    expect(store.monthlyEstimatedSharePct).toBe(0);
    expect(store.hasEstimatedData).toBe(false);
    // Not disclosed, but not hidden either: the stretch with no records is what the
    // quality table's gap column reports, so an engineer can still go and find out
    // why 70 days of writes are missing.
    expect(store.meterEstimation[B].unrecordedHours).toBeGreaterThan(0);
    expect(store.meterEstimation[B].longestGap).not.toBeNull();
  });

  it('names the instant it means and the earliest reading it holds', async () => {
    // Reported confusion: the fault read "no reading at or before this point"
    // against a meter with a current reading and another from today, which looks
    // like the check is broken. The instant is months back — the meter's history
    // simply begins after it — so the message now says which instant it means and
    // where the history actually starts.
    //
    // Deliberately no longer worded "period start". The rolling monthly table
    // resolves twelve boundaries that are nothing of the kind, and it hits this same
    // message on its earliest one, where calling August's boundary a period start
    // would be plainly wrong.
    const onlyA = {lighting: [A], pvGeneration: [], pvExport: []};
    givenMeters({[A]: {series: [
      {at: new Date('2026-06-01T09:00:00Z'), usage: 5000},
      {at: new Date('2026-07-01T09:00:00Z'), usage: 6000}
    ], current: 6500}});
    const store = store2({meterNames: onlyA, ratingPeriodStart: '2026-01-01'});
    await store.refresh();

    const reason = store.meterFailures[0].reason;
    expect(reason).toContain('no reading at or before this point');
    expect(reason).toContain('1 Jan 26');
    expect(reason).toContain('1 Jun 26');
  });

  it('keeps the earliest-held detail when readings were also discarded', async () => {
    // These two facts were mutually exclusive, and they travel together far more
    // often than not: discarding a meter's only early record is *why* it then has no
    // reading at the boundary. So the one case that needed both got neither — which
    // is exactly what August 2025 at 3CS reported, leaving no way to tell whether
    // looking further back could ever have helped.
    const onlyA = {lighting: [A], pvGeneration: [], pvExport: []};
    givenMeters({[A]: {series: [
      // Above the current reading, so the ceiling filter discards it. It is also the
      // only record before the period start.
      {at: new Date('2026-02-01T09:00:00Z'), usage: 999999},
      {at: new Date('2026-06-01T09:00:00Z'), usage: 5000},
      {at: new Date('2026-07-01T09:00:00Z'), usage: 6000}
    ], current: 6500}});
    const store = store2({meterNames: onlyA, ratingPeriodStart: '2026-01-01'});
    await store.refresh();

    const reason = store.meterFailures[0].reason;
    expect(reason).toContain('earliest held 1 Jun 26');
    expect(reason).toContain('implausible readings discarded');
  });

  it('refuses a meter with no reading at or before the period start', async () => {
    // A leading gap has no bounded interval to lean on and no liveness signal for
    // the past, so idle-since-before-the-period and records-start-late are
    // indistinguishable. Backward extrapolation used to invent 880 kWh here.
    givenMeters({
      [A]: {at: {[ANCHOR]: 100}, current: 200},
      [B]: {at: {[daysAgo(20)]: 5000}, current: 5400}
    });
    const store = store2();
    await store.refresh();
    expect(store.categoryPeriodKwh.lighting).toBeNull();
    expect(store.unreadableMeters).toEqual([B]);
    expect(store.meterFailures[0].reason).toMatch(/at or before/);
  });

  it('estimates a meter that stopped reporting, and discloses which one', async () => {
    // B went silent 20 days ago. Its own mean rate is the only basis for the
    // tail, so that stretch is extrapolated rather than left unknown.
    givenMeters({
      [A]: {at: {[ANCHOR]: 100}, current: 200},
      [B]: {at: {[ANCHOR]: 50, [daysAgo(20)]: 250}}
    });
    const store = store2();
    await store.refresh();

    // The category now has a figure at all, where before it was null and the
    // whole rating was withheld.
    expect(store.categoryPeriodKwh.lighting).not.toBeNull();
    expect(store.unreadableCategories).toEqual([]);
    expect(store.hasEstimatedData).toBe(true);
    expect(store.periodEstimatedSharePct).toBeGreaterThan(0);

    // And it names the board, not just the end use.
    expect(store.meterEstimates).toHaveLength(1);
    expect(store.meterEstimates[0]).toMatchObject({name: B, category: 'lighting'});
    expect(store.estimatedMeterLabels[0]).toContain('b');
    expect(store.meterEstimation[B].estimatedHours).toBeGreaterThan(0);
  });

  it('never understates a dead meter\'s consumption', async () => {
    // B accumulated steadily until it went silent 20 days ago. The uplift must
    // make the reported figure strictly larger than a plain projection, so a
    // substituted value can never flatter the rating.
    const onlyB = {lighting: [B], pvGeneration: [], pvExport: []};
    givenMeters({[B]: {at: {[ANCHOR]: 0, [daysAgo(20)]: 310}}});

    const uplifted = store2({meterNames: onlyB});
    await uplifted.refresh();
    const withUplift = uplifted.categoryPeriodKwh.lighting;

    setActivePinia(createPinia());
    const plain = store2({meterNames: onlyB, estimation: {extrapolationUpliftPct: 0}});
    await plain.refresh();

    expect(withUplift).toBeGreaterThan(plain.categoryPeriodKwh.lighting);
    // Both still beat the un-estimated 310, which is what a truncated read gives.
    expect(plain.categoryPeriodKwh.lighting).toBeGreaterThan(310);
  });

  it('leaves a meter silent for longer than the search window unreadable', async () => {
    // Estimation has a reach. Beyond `searchWindowDays` there is no nearby
    // reading to anchor a projection, and inventing months of consumption from
    // a half-year-old rate would be fabrication, not estimation.
    givenMeters({
      [A]: {at: {[ANCHOR]: 100}, current: 200},
      [B]: {at: {[ANCHOR]: 50, [daysAgo(120)]: 250}}
    });
    const store = store2();
    await store.refresh();
    expect(store.categoryPeriodKwh.lighting).toBeNull();
    expect(store.unreadableMeters).toEqual([B]);
  });

  it('still reports a meter with no history at all as unreadable', async () => {
    // Estimation recovers gaps; it does not invent a meter. Nothing to
    // interpolate between means the end use stays honestly unknown.
    givenMeters({[A]: {at: {[ANCHOR]: 100}, current: 200}});
    const store = store2();
    await store.refresh();
    expect(store.categoryPeriodKwh.lighting).toBeNull();
    expect(store.unreadableCategories).toContain('lighting');
    expect(store.meterFailures).toHaveLength(1);
    expect(store.meterFailures[0].name).toBe(B);
  });

  it('discards corrupt readings and still produces a figure', async () => {
    // The sequence reported from site, from a driver fault since fixed: history
    // still holds 469780064 and -2147465600 between a 14848 and an 18025. Left in,
    // the negative reads as an accumulator reset and nulls the whole category, and
    // the spike wrecks the meter's mean rate along with it.
    const onlyA = {lighting: [A], pvGeneration: [], pvExport: []};
    givenMeters({[A]: {series: [
      {at: new Date('2026-05-02T00:00:00Z'), usage: 14000},
      {at: new Date('2026-06-10T10:00:00Z'), usage: 14848},
      {at: new Date('2026-06-10T10:15:00Z'), usage: 469780064},
      {at: new Date('2026-06-10T10:30:00Z'), usage: -2147465600},
      {at: new Date('2026-07-20T09:00:00Z'), usage: 18025}
    ], current: 18100}});
    const store = store2({meterNames: onlyA, ratingPeriodStart: '2026-06-01'});
    await store.refresh();

    expect(store.meterQuality[A].rejectedReadings).toBe(2);
    // 18100 now, less the value carried forward to 1 June from the 2 May reading
    // that brackets it — a real figure rather than a null.
    expect(store.categoryPeriodKwh.lighting).not.toBeNull();
    expect(store.categoryPeriodKwh.lighting).toBeGreaterThan(0);
    expect(store.categoryPeriodKwh.lighting).toBeLessThan(5000);
    expect(store.meterFailures).toEqual([]);
  });

  it('measures a coarse meter\'s resolution and stops calling one tick a gap', async () => {
    // Meters tick at different quanta — some every 1 kWh, some every 16 — so a
    // fixed threshold cannot serve both. This meter records in 16 kWh steps and
    // ticked exactly once across a twenty-day stretch of no records, which against
    // the old fixed 1 kWh default was reported as a twenty-day gap.
    //
    // Its resolution is measured from the runs of consecutive records behind each
    // probe, which cost no extra queries.
    const onlyA = {lighting: [A], pvGeneration: [], pvExport: []};
    const series = [];
    let usage = 1000;
    // Dense 16 kWh steps up to 40 days ago, then one lone tick 20 days ago.
    for (let d = 260; d >= 40; d -= 0.25) {
      series.push({at: new Date(Date.now() - d * DAY_MS), usage});
      usage += 16;
    }
    series.push({at: new Date(Date.now() - 20 * DAY_MS), usage});
    givenMeters({[A]: {series, current: usage}});

    const store = store2({meterNames: onlyA});
    await store.refresh();

    expect(store.meterQuality[A].observedTickKwh).toBe(16);
    expect(store.meterQuality[A].longestGap).toBeNull();
    expect(store.meterEstimates).toEqual([]);

    // The same fixture with measurement effectively disabled — one reading per
    // probe, so no run to measure from — falls back to the 1 kWh default and does
    // report the stretch. That contrast is the point of the feature.
    setActivePinia(createPinia());
    const fixed = store2({meterNames: onlyA, estimation: {tickSampleCount: 1}});
    await fixed.refresh();
    expect(fixed.meterQuality[A].observedTickKwh).toBeNull();
    expect(fixed.meterQuality[A].longestGap).not.toBeNull();
  });

  it('shows no gap when the accumulator never moved, whatever the record spacing', async () => {
    // Reported from site, with these exact readings: 1030.000 on 24 Jun, then
    // 1030.000 again on 14 Jul after a device restart wrote a fresh record of an
    // unchanged value. It was shown as a twenty-day gap.
    //
    // The accumulator is monotonic, so equal readings at both ends mean its value
    // everywhere between them is that same value — known exactly. Meters on
    // lightly used circuits change very infrequently and must not read as faulty.
    givenMeters({[A]: {series: [
      {at: new Date('2026-05-20T08:00:00Z'), usage: 1030},
      {at: new Date('2026-06-24T14:05:00Z'), usage: 1030},
      {at: new Date('2026-07-14T09:40:00Z'), usage: 1030}
    ], current: 1030}});
    const store = storeWith({
      enabled: true, nia: 1000, postcode: 'WD25 9NH', ratedHours: 60,
      ratingPeriodStart: '2026-06-01',
      meterNames: {carPark: [A], pvGeneration: [], pvExport: []}
    });
    await store.refresh();

    expect(store.categoryPeriodKwh.carPark).toBe(0);
    expect(store.meterQuality[A].longestGap).toBeNull();
    expect(store.meterQuality[A].unrecordedHours).toBe(0);
    expect(store.meterEstimates).toEqual([]);
    // Reported as idle instead, which is what it is: present, and consuming nothing.
    expect(store.meterIdle).toEqual([A]);
  });

  it('does not report a continuously-reporting meter as gappy', async () => {
    // Reported from site: a meter with a record every fifteen minutes and no holes
    // at all was shown as "31 days, 30 Jun–31 Jul", and downloading its history
    // for that range returned plenty of data.
    //
    // The sample pool holds roughly one reading per probed instant, so consecutive
    // pooled readings sit about a month apart however healthy the meter is. Judging
    // the boundary by that bracket's *width* rather than by the distance to the
    // nearest reading flagged every interior boundary — and the more often the
    // meter reported, the worse it looked, because a dense meter's reading lands
    // so close to the boundary that no follow-up probe is issued at all.
    const onlyA = {lighting: [A], pvGeneration: [], pvExport: []};
    givenMeters({[A]: denseMeter({days: 260, everyHours: 0.25})});
    const store = store2({meterNames: onlyA});
    await store.refresh();

    expect(store.meterQuality[A].longestGap).toBeNull();
    expect(store.meterQuality[A].unrecordedHours).toBe(0);
    expect(store.meterEstimates).toEqual([]);
    expect(store.hasEstimatedData).toBe(false);
    expect(store.categoryPeriodKwh.lighting).not.toBeNull();
  });

  it('still finds a genuine hole inside a dense series, and dates it', async () => {
    // The other half of the same fix: suppressing the false positives must not
    // suppress a real outage. Twenty days missing out of a fifteen-minute series.
    const onlyA = {lighting: [A], pvGeneration: [], pvExport: []};
    givenMeters({[A]: denseMeter({days: 260, everyHours: 0.25, holeFrom: 70, holeTo: 50})});
    const store = store2({meterNames: onlyA});
    await store.refresh();

    const gap = store.meterQuality[A].longestGap;
    expect(gap).not.toBeNull();
    expect(gap.hours / 24).toBeCloseTo(20, 0);
    const daysBack = (d) => (Date.now() - d.getTime()) / DAY_MS;
    expect(daysBack(gap.from)).toBeCloseTo(70, 0);
    expect(daysBack(gap.to)).toBeCloseTo(50, 0);
    // Mid-period, so both ends of the period are real readings and the total is
    // exact — the hole shows in the gap column, not as an estimated figure.
    expect(store.hasEstimatedData).toBe(false);
  });

  it('bounds a gap to the months it actually covers, not the whole period', async () => {
    // The defect this pins. `refresh` used to resolve only the period's two ends,
    // so the opening boundary bracketed from the last record before the period
    // straight to the next record it had fetched — skipping every record in
    // between, because the forward probe only looks `searchWindowDays` ahead. A
    // meter whose only real hole was the first 66 days of the period was reported
    // as 290 days unrecorded and 95% estimated. Sampling each month boundary
    // inside the period bounds it to what actually happened.
    //
    // B: reads before the period, silent for its first ~66 days, then normal.
    const b = {at: {[daysAgo(305)]: 1000, [daysAgo(234)]: 1200}, current: 3100};
    for (let d = 220; d >= 5; d -= 15) b.at[daysAgo(d)] = 1200 + (234 - d) * 8;
    givenMeters({[A]: {at: {[daysAgo(300)]: 100}, current: 200}, [B]: b});

    const store = store2({ratingPeriodStart: daysAgo(300)});
    await store.refresh();

    const gap = store.meterQuality[B].longestGap;
    // ~66 days, not 290. A month of slack: the gap is located to the nearest
    // month boundary, which is the granularity the period is sampled at.
    expect(gap.hours / 24).toBeGreaterThan(50);
    expect(gap.hours / 24).toBeLessThan(100);
    // And it is dated, which is the whole point — "when was this gap?" now has an
    // answer. It starts at the period start and ends when the meter came back.
    const daysBack = (d) => (Date.now() - d.getTime()) / DAY_MS;
    expect(daysBack(gap.from)).toBeCloseTo(300, -1);
    expect(daysBack(gap.to)).toBeCloseTo(234, -1);
    // And the disclosed energy is a small share, not almost all of it.
    expect(store.periodEstimatedSharePct).toBeLessThan(25);
    expect(store.categoryPeriodKwh.lighting).not.toBeNull();
  });

  it('will not project further than the estimation window', async () => {
    // B is unreachable and last recorded at the period start, months back. A rate
    // projected across that much silence is fabrication, not estimation, so the
    // boundary stays unknown and the board gets named.
    givenMeters({[A]: {at: {[ANCHOR]: 100}, current: 200}, [B]: {at: {[ANCHOR]: 50}}});
    const store = store2();
    await store.refresh();
    expect(store.categoryPeriodKwh.lighting).toBeNull();
    expect(store.unreadableMeters).toEqual([B]);
    expect(store.meterFailures[0].reason).toMatch(/estimation window/);
  });

  it('cannot estimate from a single reading', async () => {
    // Inside the window, so the reach rule does not apply — but one sample gives
    // no rate, so there is still nothing to project with. A short rating period,
    // because over a long one the reach rule fires first.
    const start = daysAgo(20);
    givenMeters({[A]: {at: {[start]: 100}, current: 200}, [B]: {at: {[start]: 50}}});
    const store = store2({ratingPeriodStart: start});
    await store.refresh();
    expect(store.categoryPeriodKwh.lighting).toBeNull();
    expect(store.meterFailures[0].reason).toMatch(/too little history/);
  });
});

/**
 * The rating period turns over every year on its anniversary, and for the four
 * weeks after that the period-to-date basis cannot be annualised. Nothing about
 * the building has changed, so nothing on the dashboard should go blank.
 */
describe('the turn of the rating period', () => {
  const A = 'site/devices/a';

  beforeEach(() => {
    setActivePinia(createPinia());
    _uiConfig.value = {};
    _meters.clear();
  });

  /**
   * A period that began on this day last year, so today is a few days into a
   * fresh one.
   *
   * @param {number} n days before the anniversary
   * @return {string} yyyy-mm-dd
   */
  function anniversaryDaysAgo(n) {
    const d = new Date();
    d.setDate(d.getDate() - n);
    d.setFullYear(d.getFullYear() - 1);
    return localDay(d);
  }

  /**
   * @param {Object} [extra]
   * @return {Object}
   */
  function store2(extra = {}) {
    return storeWith({
      enabled:           true,
      nia:               1000,
      postcode:          'WD25 9NH',
      ratedHours:        60,
      ratingPeriodStart: anniversaryDaysAgo(4),
      meterNames:        {lighting: [A], pvGeneration: [], pvExport: []},
      ...extra
    });
  }

  it('falls back to the measured months rather than blanking', async () => {
    // Reported from site: with ratingPeriodStart a year and four days back, the
    // period rolls to its anniversary and is four days old. Annualising four days
    // multiplies them by ninety, so the projection is rightly suppressed — but
    // every stat card then read "4 day(s) into the rating period; a projection
    // needs 28" for the next four weeks, even with a year of history behind it.
    //
    // The earliest month has no reading behind it, so a settled twelve-month
    // rating is not available either — which is the situation on a site whose
    // metering has any history gaps at all. The measured months are the basis.
    const gappy = risingMeter(1000);
    delete gappy.at[boundaryDates()[0]];
    givenMeters({[A]: gappy});
    const store = store2();
    await store.refreshMonthly();
    await store.refresh();

    expect(store.canProject).toBe(false);
    expect(store.projectedRating).toBeNull();
    expect(store.hasFullRatingPeriod).toBe(false);
    expect(store.headlineBasis).toBe('trailing');
    expect(store.headlineRating).not.toBeNull();
    expect(store.totalIntensity).toBeGreaterThan(0);
    // Eleven months of real measurement, rather than four days extrapolated.
    expect(store.monthsOfData).toBe(11);
    expect(store.trailingDaysCovered).toBeGreaterThan(300);
  });

  it('annualises the breakdown over the same window as the headline', async () => {
    // The breakdown chart used to annualise the period to date however short it
    // was, so four days after an anniversary it multiplied four days by ninety
    // while the gauge beside it had already fallen back to the measured months.
    // The two disagreed, and the chart was the one that was wrong.
    const gappy = risingMeter(1000);
    delete gappy.at[boundaryDates()[0]];
    givenMeters({[A]: gappy});
    const store = store2();
    await store.refreshMonthly();
    await store.refresh();

    expect(store.canProject).toBe(false);
    expect(store.intensityBasis).toBe('months');
    // The kWh behind the bars is the trailing sum, not four days of it.
    expect(store.categoryBasisKwh.lighting).toBe(store.trailingCategoryKwh.lighting);
    expect(store.categoryBasisKwh.lighting).toBeGreaterThan(0);

    // And the breakdown sums to the same intensity the headline reports, because
    // both now annualise the same window over the same area.
    const summed = store.categories
      .reduce((a, c) => a + (store.categoryIntensities[c] ?? 0), 0);
    expect(summed).toBeCloseTo(store.totalIntensity, 6);
  });

  it('uses the period to date once it is long enough to annualise', async () => {
    // The other side of the switch: mid-period, the chart should annualise the
    // period as it always did rather than the months.
    givenMeters({[A]: risingMeter(1000)});
    const store = store2({ratingPeriodStart: anniversaryDaysAgo(-200)});
    await store.refresh();
    expect(store.canProject).toBe(true);
    expect(store.hasFullRatingPeriod).toBe(false);
    expect(store.intensityBasis).toBe('period');
    expect(store.categoryBasisKwh.lighting).toBe(store.categoryPeriodKwh.lighting);
  });

  it('prefers a settled twelve months over the trailing basis', async () => {
    givenMeters({[A]: risingMeter(1000)});
    const store = store2();
    await store.refreshMonthly();
    expect(store.hasFullRatingPeriod).toBe(true);
    expect(store.headlineBasis).toBe('standing');
  });

  it('says so plainly when there are too few months to fall back on either', async () => {
    // No monthly data at all, so neither basis is available. Then, and only then,
    // the too-early message is the right thing to show.
    givenMeters({[A]: {at: {}, current: 500}});
    const store = store2();
    await store.refresh();
    expect(store.canProject).toBe(false);
    expect(store.canUseTrailing).toBe(false);
    expect(store.headlineBasis).toBeNull();
  });
});

describe('a six-category site with no meters configured', () => {
  /** The shape a real production config takes: every array empty, four of six categories targeted. */
  const CFG = {
    enabled:    true,
    nia:        12137,
    postcode:   'WD25 9NH',
    ratedHours: 60,
    meterNames: {
      hvac: [], lifts: [], commonAreaLighting: [], exteriorLighting: [],
      carPark: [], smallPower: [], pvGeneration: [], pvExport: []
    },
    dfpTargets: {hvac: 42.6, lifts: 5.6, commonAreaLighting: 7.0, smallPower: 1.9, total: 55.7}
  };

  beforeEach(() => {
    setActivePinia(createPinia());
    _uiConfig.value = {};
    _meters.clear();
  });

  it('reports no months of data rather than twelve empty ones', async () => {
    const store = storeWith(CFG);
    await store.refreshMonthly();
    // The trap: were an empty meter pool to sum to 0 rather than null, this would
    // be twelve months of zero consumption, hasFullRatingPeriod would be true,
    // and the dashboard would publish a settled six-star rating for a building
    // with no meters at all.
    expect(store.monthsOfData).toBe(0);
    expect(store.hasFullRatingPeriod).toBe(false);
    expect(store.standingRating).toBeNull();
  });

  it('reports no rating and no spurious failures from refresh', async () => {
    const store = storeWith(CFG);
    await store.refresh();
    expect(store.hasConfiguredMeters).toBe(false);
    expect(store.grossPeriodKwh).toBeNull();
    expect(store.meterFailures).toEqual([]);
    expect(store.error).toBeNull();
  });
});

describe('mapLimit', () => {
  it('preserves input order regardless of completion order', async () => {
    const out = await mapLimit([30, 10, 20], 3, async ms => {
      await new Promise(r => setTimeout(r, ms / 10));
      return ms;
    });
    expect(out).toEqual([30, 10, 20]);
  });

  it('never exceeds the limit concurrently', async () => {
    let inFlight = 0;
    let peak = 0;
    await mapLimit(Array.from({length: 40}, (_, i) => i), 6, async () => {
      inFlight++;
      peak = Math.max(peak, inFlight);
      await new Promise(r => setTimeout(r, 1));
      inFlight--;
    });
    expect(peak).toBeLessThanOrEqual(6);
  });

  it('resolves immediately on an empty list', async () => {
    const fn = vi.fn();
    expect(await mapLimit([], 6, fn)).toEqual([]);
    expect(fn).not.toHaveBeenCalled();
  });

  it('bounds concurrency through the store, not only in isolation', async () => {
    setActivePinia(createPinia());
    _meters.clear();
    _peakInFlight = 0;
    _inFlight = 0;
    const names = Array.from({length: 40}, (_, i) => `site/devices/m${i}`);
    names.forEach(n => givenMeters({[n]: {at: {'2026-01-01': 1}, current: 2}}));
    const store = storeWith({
      enabled:           true,
      ratingPeriodStart: '2026-01-01',
      meterNames:        {lighting: names, pvGeneration: [], pvExport: []}
    });
    await store.refresh();
    // The boundary reader's passes run in sequence, each with its own pool, so
    // the ceiling is the limit itself rather than a multiple of it.
    expect(_peakInFlight).toBeLessThanOrEqual(12);
  });
});

describe('splitMeterName', () => {
  it('separates the noisy prefix from the distinguishing leaf', () => {
    expect(splitMeterName('bldg-1/floors/07/devices/db-ll-n7'))
      .toEqual({prefix: 'bldg-1/floors/07/devices/', leaf: 'db-ll-n7'});
  });

  it('handles a name with no separator', () => {
    expect(splitMeterName('meter')).toEqual({prefix: '', leaf: 'meter'});
  });
});

describe('meterIdentity', () => {
  const meta = (o) => ({
    appearance: {title: o.title ?? ''},
    location:   {floor: o.floor ?? '', zone: o.zone ?? ''},
    moreMap:    o.ref ? [['ref', o.ref]] : []
  });

  it('splits the installed title into ref and name', () => {
    const id = meterIdentity('bldg-1/floors/08/devices/db-ll-8-ltg', meta({
      title: 'EM/118 - Level 08 Landlords Db Lighting',
      ref:   'EM/118',
      floor: 'Floor 08',
      zone:  'South'
    }));
    expect(id.ref).toBe('EM/118');
    // The ref and its separator are gone from the name, so the two columns do
    // not repeat each other.
    expect(id.label).toBe('Level 08 Landlords Db Lighting');
    expect(id.title).toBe('EM/118 - Level 08 Landlords Db Lighting');
    expect(id.location).toBe('Floor 08 · South');
    expect(id.hasMetadata).toBe(true);
  });

  it('leaves a title alone when it does not start with the ref', () => {
    const id = meterIdentity('a/b/lift-1', meta({title: 'Passenger Lift 1', ref: 'EM/201'}));
    expect(id.label).toBe('Passenger Lift 1');
    expect(id.ref).toBe('EM/201');
  });

  it('keeps the name when there is no ref, which 11 devices are missing', () => {
    const id = meterIdentity('a/b/lv-wshp', meta({title: 'LV WSHP', floor: 'Level 00'}));
    expect(id).toMatchObject({ref: '', label: 'LV WSHP', location: 'Level 00', hasMetadata: true});
  });

  it('falls back to the distinguishing leaf when a device has no metadata', () => {
    const id = meterIdentity('bldg-1/floors/07/devices/db-ll-n7');
    // Worse than a name, but it still identifies the board, which a blank cell
    // would not.
    expect(id).toMatchObject({ref: '', label: 'db-ll-n7', location: '', hasMetadata: false});
  });

  it('treats a title that is only the ref as the name of last resort', () => {
    expect(meterIdentity('a/b/c', meta({title: 'EM/118', ref: 'EM/118'})).label).toBe('EM/118');
  });
});

describe('meter identity in the store', () => {
  const LTG = 'bldg-1/floors/08/devices/db-ll-8-ltg';
  const PWR = 'bldg-1/floors/08/devices/db-ll-8-pwr';

  beforeEach(() => {
    setActivePinia(createPinia());
    _uiConfig.value = {};
    _meters.clear();
    _metadata.clear();
  });

  /** @return {Object} */
  const twoMeterStore = () => storeWith({
    enabled: true, nia: 16903, ratedHours: 61.1, postcode: 'B3 2BH',
    meterNames: {lighting: [LTG], smallPower: [PWR]}
  });

  it('names every configured meter, whether or not its metadata could be read', async () => {
    givenMetadata({[LTG]: {title: 'EM/118 - Level 08 Landlords Db Lighting', ref: 'EM/118'}});
    const store = twoMeterStore();
    await store.refreshMeterMetadata();

    expect(store.meterIdentities[LTG].label).toBe('Level 08 Landlords Db Lighting');
    // PWR's fetch rejected; it still has an entry, so the table renders a row.
    expect(store.meterIdentities[PWR]).toMatchObject({label: 'db-ll-8-pwr', hasMetadata: false});
    expect(store.meterLabel(LTG)).toBe('Level 08 Landlords Db Lighting');
  });

  it('names unreadable meters by their human-readable name, not their path', async () => {
    givenMetadata({
      [LTG]: {title: 'EM/118 - Level 08 Landlords Db Lighting', ref: 'EM/118'},
      [PWR]: {title: 'EM/119 - Level 08 Landlords Db Small Power', ref: 'EM/119'}
    });
    // Neither meter has any history, so both are unreadable — the state that
    // produced "awaiting db-ll-8-ltg" on screen.
    const store = twoMeterStore();
    await store.refreshMeterMetadata();
    await store.refresh();

    expect(store.unreadableMeters).toEqual([LTG, PWR]);
    expect(store.unreadableMeterLabels).toEqual([
      'Lighting: Level 08 Landlords Db Lighting',
      'Small Power: Level 08 Landlords Db Small Power'
    ]);
    expect(store.unreadableMeterLabels.join()).not.toContain('bldg-1/');
  });

  it('has an identity for every meter before any metadata is fetched', () => {
    const store = twoMeterStore();
    expect(Object.keys(store.meterIdentities)).toEqual([LTG, PWR]);
  });
});

describe('one annualisation window across the widgets', () => {
  const LTG = 'site/devices/ltg';
  const PV  = 'site/devices/pv';

  beforeEach(() => {
    setActivePinia(createPinia());
    _uiConfig.value = {};
    _meters.clear();
  });

  /**
   * A period that rolled over four days ago, with a full year of history behind
   * it. The state 3CS was in: the projection is suppressed, so every annualised
   * figure has to come from the measured months.
   *
   * @return {string} yyyy-mm-dd
   */
  function freshPeriodStart() {
    const d = new Date();
    d.setDate(d.getDate() - 4);
    d.setFullYear(d.getFullYear() - 1);
    return localDay(d);
  }

  /**
   * @param {Object} [extra]
   * @return {Object}
   */
  function pvStore(extra = {}) {
    return storeWith({
      enabled:           true,
      nia:               1000,
      postcode:          'B3 2BH',
      ratedHours:        61.1,
      ratingPeriodStart: freshPeriodStart(),
      meterNames:        {lighting: [LTG], pvGeneration: [PV], pvExport: []},
      ...extra
    });
  }

  it('annualises the energy split over the months, not the days since rollover', async () => {
    // The bug: gross and PV read `annualisationFactor` directly, so four days
    // into a period they multiplied four days by ninety while the chart beside
    // them had already fallen back to the measured months.
    givenMeters({[LTG]: risingMeter(10), [PV]: risingMeter(2)});
    const store = pvStore();
    await store.refreshMonthly();
    await store.refresh();

    expect(store.intensityBasis).toBe('months');
    const factor = 365 / store.trailingDaysCovered;
    expect(store.grossIntensity).toBeCloseTo((12 * 10 / 1000) * factor, 9);
    expect(store.pvIntensity).toBeCloseTo((12 * 2 / 1000) * factor, 9);
    // Nowhere near the period-to-date extrapolation, which at five elapsed days
    // would have been seventy-odd times a few days of energy.
    expect(store.grossIntensity).toBeLessThan(1);
  });

  it('makes the end-use bars sum to the gross the energy split draws', async () => {
    // The invariant that fails the moment two widgets pick different windows:
    // same meters, same area, so the bars can only add up to the whole if both
    // annualised the same stretch.
    givenMeters({[LTG]: risingMeter(10), [PV]: risingMeter(2)});
    const store = pvStore();
    await store.refreshMonthly();
    await store.refresh();

    const barTotal = store.categories.reduce((a, c) => a + (store.categoryIntensities[c] ?? 0), 0);
    expect(barTotal).toBeCloseTo(store.grossIntensity, 9);
  });

  it('takes the solar share over that same window', async () => {
    givenMeters({[LTG]: risingMeter(10), [PV]: risingMeter(2)});
    const store = pvStore();
    await store.refreshMonthly();
    await store.refresh();

    // Generation against gross *demand*, which is the end-use meters only — PV
    // is a reserved key and never counts toward the gross it offsets. So 24 kWh
    // of generation against 120 kWh of demand, the ratio the donut draws, and it
    // is only right if both cover the same months.
    expect(store.pvIntensity / store.grossIntensity).toBeCloseTo(24 / 120, 9);
  });

  it('shows nothing rather than a 52x extrapolation when there is no basis yet', async () => {
    // Four days in with no monthly table fetched: too early to project, and
    // nothing measured to fall back on. This used to render a figure anyway.
    givenMeters({[LTG]: risingMeter(10), [PV]: risingMeter(2)});
    const store = pvStore();
    await store.refresh();

    expect(store.canProject).toBe(false);
    expect(store.intensityBasis).toBeNull();
    expect(store.grossIntensity).toBeNull();
    expect(store.pvIntensity).toBeNull();
  });
});

describe('the design PV reference', () => {
  beforeEach(() => {
    setActivePinia(createPinia());
    _uiConfig.value = {};
    _meters.clear();
  });

  /**
   * @param {Object} dfpTargets
   * @return {Object}
   */
  function targetStore(dfpTargets) {
    return storeWith({enabled: true, nia: 16903, dfpTargets});
  }

  it('derives 3CS from the gross/net pair, matching Cundall section 6.9', () => {
    // 27,175 kWh/yr over the 16,903 m² rated area is 1.608, and totalGross minus
    // total is 1.61 — the same figure, so the reference draws with no extra
    // config. That equality is the whole basis for deriving it.
    const store = targetStore({total: 47.54, totalGross: 49.15});
    expect(store.dfpPvIntensity).toBeCloseTo(1.61, 2);
    expect(store.dfpPvIntensity).toBeCloseTo(27175 / 16903, 2);
    expect(store.dfpPvSharePct).toBeCloseTo(3.28, 2);
  });

  it('prefers an explicit target over the derivation', () => {
    const store = targetStore({total: 47.54, totalGross: 49.15, pvGeneration: 2.5});
    expect(store.dfpPvIntensity).toBe(2.5);
  });

  it('keeps an explicit target out of the breakdown bars', () => {
    // `pvGeneration` is generation, not an end use. A target key that became a
    // category would draw a bar the meters can never fill.
    const store = targetStore({total: 47.54, totalGross: 49.15, pvGeneration: 2.5});
    expect(store.categories).not.toContain('pvGeneration');
  });

  it('draws nothing rather than a zero reference when the design assumed no PV', () => {
    expect(targetStore({total: 47.54, totalGross: 47.54}).dfpPvIntensity).toBeNull();
    expect(targetStore({total: 47.54}).dfpPvIntensity).toBeNull();
    expect(targetStore({}).dfpPvIntensity).toBeNull();
    expect(targetStore({total: 47.54}).dfpPvSharePct).toBeNull();
  });
});

/**
 * The rung above the target, which the second headroom card measures against.
 *
 * The distinction worth protecting is which rating it steps off. `nextStarTarget`
 * steps off the *achieved* banded rating and drives the rating gauge; this steps
 * off the *configured target* so it pairs with `headroomPct` on the same footing.
 * Anchor them alike and two cards side by side silently measure different things.
 */
describe('stretch target: the rung above the configured target', () => {
  const A = 'site/devices/a';

  beforeEach(() => {
    setActivePinia(createPinia());
    _uiConfig.value = {};
    _meters.clear();
  });

  /**
   * A store that can compute a benchmark, and a rating once meters are given.
   *
   * @param {Object} [extra] merged over the config
   * @return {Object}
   */
  function ratedStore(extra = {}) {
    return storeWith({
      enabled:    true,
      nia:        1000,
      postcode:   'WD25 9NH',
      ratedHours: 60,
      meterNames: {lighting: [A], pvGeneration: [], pvExport: []},
      ...extra
    });
  }

  it('steps one published rung above the default 5-star target', () => {
    const store = ratedStore();
    expect(store.targetStars).toBe(5);
    expect(store.stretchTarget.stars).toBe(5.5);
    expect(store.stretchTarget.ceiling).toBe(intensityForStars(5.5, store.benchmark));
    // Tighter than the target's own ceiling, which is the point of the card.
    expect(store.stretchTarget.ceiling).toBeLessThan(store.targetStarMax);
  });

  it('steps off the target rather than off the achieved rating', async () => {
    // A 4-star target on a building rating far better than that. `nextStarTarget`
    // follows the building; this has to stay put on 4.5.
    givenMeters({[A]: risingMeter(10)});
    const store = ratedStore({targetStars: 4});
    await store.refreshMonthly();
    expect(store.bandedStars).toBe(6);
    expect(store.nextStarTarget).toBeNull();       // nothing above 6
    expect(store.stretchTarget.stars).toBe(4.5);
  });

  it('has no rung to offer above a 6-star target', () => {
    const store = ratedStore({targetStars: 6});
    expect(store.targetStars).toBe(6);
    expect(store.stretchTarget).toBeNull();
    expect(store.stretchHeadroomPct).toBeNull();
    expect(store.stretchReductionNeeded).toBeNull();
  });

  it('still names the rung when the benchmark is unavailable', () => {
    // Postcode withheld, so there is no benchmark and so no ceiling. The card
    // needs the star label regardless — "Headroom to 5.5★ — benchmark
    // unavailable" says what is wrong; a blank title does not.
    const store = ratedStore({postcode: ''});
    expect(store.benchmark).toBeNull();
    expect(store.stretchTarget.stars).toBe(5.5);
    expect(store.stretchTarget.ceiling).toBeNull();
    expect(store.stretchHeadroomPct).toBeNull();
    expect(store.stretchReductionNeeded).toBeNull();
  });

  it('steps up from the fallback when config names a rung between bands', () => {
    // 5.2 is not a published band, so `targetStars` falls back to 5 — and the
    // stretch rung has to follow the fallback, not the configured figure.
    const store = ratedStore({targetStars: 5.2});
    expect(store.targetStars).toBe(5);
    expect(store.stretchTarget.stars).toBe(5.5);
  });

  it('reports a negative headroom and a real cut for a building above the rung', async () => {
    // 20,000 kWh/month over 1,000 m² is far above every published ceiling.
    givenMeters({[A]: risingMeter(20000)});
    const store = ratedStore();
    await store.refreshMonthly();

    const ceiling = store.stretchTarget.ceiling;
    expect(store.totalIntensity).toBeGreaterThan(ceiling);
    expect(store.stretchHeadroomPct)
      .toBeCloseTo(((ceiling - store.totalIntensity) / ceiling) * 100, 6);
    expect(store.stretchHeadroomPct).toBeLessThan(0);
    expect(store.stretchReductionNeeded).toBeCloseTo(store.totalIntensity - ceiling, 6);
    // Harder than the target it is already failing, in both figures.
    expect(store.stretchHeadroomPct).toBeLessThan(store.headroomPct);
  });

  it('floors the cut at zero once the rung is already met', async () => {
    givenMeters({[A]: risingMeter(10)});
    const store = ratedStore();
    await store.refreshMonthly();

    expect(store.totalIntensity).toBeLessThan(store.stretchTarget.ceiling);
    expect(store.stretchHeadroomPct).toBeGreaterThan(0);
    // Not a negative "cut needed", which would render as a reduction to make.
    expect(store.stretchReductionNeeded).toBe(0);
  });
});
