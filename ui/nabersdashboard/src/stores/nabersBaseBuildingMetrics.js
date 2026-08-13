import {defineStore} from 'pinia';
import {ref, computed} from 'vue';
import {
  startOfMonth, subMonths, addYears, differenceInDays, getDaysInMonth, format, parseISO, isValid
} from 'date-fns';
import {useUiConfigStore} from './uiConfig.js';
import {getMeterReading} from '@/api/sc/traits/meter.js';
import {getMetadata} from '@/api/sc/traits/metadata.js';
import {mapLimit, MAX_CONCURRENT_READS} from '@/util/concurrency.js';
import {describeRpcError} from '@/util/rpcError.js';
import {readBoundaries, periodInstants} from '@/util/meterBoundaries.js';
import {
  estimationOptions, boundaryDelta, spanDelta, sumDeltas, estimatedSharePct
} from '@/util/meterEstimation.js';
import {
  computeRating,
  missingRatingInputs,
  adjustedBenchmark,
  intensityForStars,
  equivalentEnergyKwh,
  // Aliased because this store exposes a `headroomPct` of its own — the figure
  // for *this* building against *its* target — and that name is its public
  // surface. `computeRating` sets the precedent for the prefix.
  headroomPct as computeHeadroomPct
} from '@/util/nabersRating.js';
import {DFP_RECOMMENDED_MARGIN_PCT} from '@/util/dfpSeverity.js';

// `mapLimit` moved to util/concurrency.js so the tenancy store can share it.
// Re-exported because it was part of this module's surface before the move.
export {mapLimit};

/**
 * Fallback end-use categories, used only when a config supplies no `meterNames`
 * and no `dfpTargets`. The live set is derived from those two objects, because
 * which end uses a building can meter separately is a property of the building,
 * not of this code — one site splits six ways, another eleven.
 */
export const BB_CATEGORIES = ['hvac', 'lifts', 'commonAreaLighting', 'exteriorLighting', 'carPark', 'smallPower'];

/** Keys inside `meterNames` that are generation, not a rated end use. */
const RESERVED_METER_KEYS = new Set(['pvGeneration', 'pvExport']);

/** Keys inside `dfpTargets` that are roll-ups, not an end use. */
const TARGET_ONLY_KEYS = new Set(['total', 'totalGross']);

/**
 * The published half-star rungs, best first. There is no 0.5 band, so the
 * ladder stops at 1 rather than stepping down to 0.5.
 */
const STAR_RUNGS = [6, 5.5, 5, 4.5, 4, 3.5, 3, 2.5, 2, 1.5, 1];

/**
 * Display names for every end use we know about. A config may override any of
 * these via `nabersBaseBuilding.categoryLabels`; anything absent falls back to
 * {@link humanizeCategory} so a raw camelCase key never reaches the screen.
 */
export const BB_CATEGORY_LABELS = {
  // Six-category buildings.
  hvac:               'HVAC',
  lifts:              'Lifts',
  commonAreaLighting: 'Common Lighting',
  exteriorLighting:   'Exterior',
  carPark:            'Car Park',
  smallPower:         'Small Power',
  // End uses as reported in full by the NABERS UK DfP method.
  lighting:           'Lighting',
  server:             'Server',
  other:              'Other',
  dhw:                'DHW',
  centralAhu:         'Central AHU',
  terminalUnitFans:   'Terminal Fans',
  pumps:              'Pumps',
  cooling:            'Cooling',
  heating:            'Heating',
  coolingHeating:     'Cooling + Heating',
  dehum:              'Dehumidification',
  // Generation, reported alongside the end uses in the meter quality table.
  pvGeneration:       'PV Generation',
  pvExport:           'PV Export'
};

/**
 * Turn a camelCase config key into something readable, for end uses this build
 * has no label for.
 *
 * @param {string} key
 * @return {string}
 */
export function humanizeCategory(key) {
  const spaced = String(key).replace(/([a-z0-9])([A-Z])/g, '$1 $2');
  return spaced.charAt(0).toUpperCase() + spaced.slice(1);
}

/**
 * Split a Smart Core name into a dimmable prefix and its distinguishing leaf.
 *
 * `bldg-1/floors/07/devices/db-ll-n7` differs from its sixteen siblings only in
 * the final segment, so the leaf is the whole of what a person needs and the
 * prefix is noise. Truncating such names from the right discards precisely the
 * part that identifies the board.
 *
 * @param {string} name
 * @return {{prefix: string, leaf: string}}
 */
export function splitMeterName(name) {
  const s = String(name);
  const i = s.lastIndexOf('/');
  return i < 0 ? {prefix: '', leaf: s} : {prefix: s.slice(0, i + 1), leaf: s.slice(i + 1)};
}

/**
 * The distinguishing tail of a Smart Core name, for prose.
 *
 * @param {string} name
 * @return {string}
 */
export function shortMeterName(name) {
  return splitMeterName(name).leaf;
}

/**
 * What a person should be shown for a meter, from its device metadata.
 *
 * The EMS specification asks for a unique meter ID and a human-readable name
 * covering location, floor and end use, in the dashboard and in every export. All
 * of that is already on the device, so this is a presentation of `Metadata`
 * rather than a second place to maintain names.
 *
 * The installed titles carry the ref inline, `EM/118 - Level 08 Landlords Db
 * Lighting`, because that is the form the electrical contractor's markup uses and
 * it is what an assessor cross-references. Ref and name are shown as separate
 * columns, so the ref is stripped off the front of the name to stop the two
 * repeating each other; the raw title is kept for anyone who wants it whole.
 *
 * Falls back to the Smart Core name's last segment, which is at least the part
 * that distinguishes one board from its siblings, when a device has no metadata
 * or could not be read.
 *
 * @param {string} name
 * @param {Metadata.AsObject} [meta]
 * @return {{name: string, ref: string, label: string, title: string,
 *           floor: string, zone: string, location: string, hasMetadata: boolean}}
 */
export function meterIdentity(name, meta) {
  const str = (v) => (typeof v === 'string' ? v.trim() : '');
  const title = str(meta?.appearance?.title);
  const ref   = str((meta?.moreMap ?? []).find(([k]) => k === 'ref')?.[1]);
  const floor = str(meta?.location?.floor);
  const zone  = str(meta?.location?.zone);

  let label = title;
  if (ref && label.startsWith(ref)) {
    // Trim the ref and whatever separator follows it, hyphen, dash or colon.
    label = label.slice(ref.length).replace(/^[\s\-–—:]+/, '');
  }

  return {
    name,
    ref,
    title,
    floor,
    zone,
    location: [floor, zone].filter(Boolean).join(' · '),
    label: label || title || shortMeterName(name),
    hasMetadata: Boolean(title || ref)
  };
}

/** A NABERS rating is a complete measurement over a full 12 months. */
const MONTHS_IN_RATING_PERIOD = 12;

/**
 * Below this many days into the rating period, annualising is meaningless — on
 * 2 January a straight-line projection multiplies two days of data by 182 — so
 * no projection is offered at all.
 */
const MIN_PROJECTION_DAYS = 28;

export const useNabersBaseBuildingStore = defineStore('nabersdashboard:nabersBaseBuilding', () => {
  const uiConfig = useUiConfigStore();

  const categoryPeriodKwh = ref({});
  const pvGenerationKwh   = ref(null);
  const pvExportKwh       = ref(null);
  const loading           = ref(false);
  const error             = ref(null);

  const bbCfg        = computed(() => uiConfig.config?.nabersBaseBuilding ?? {});
  const meterCfg     = computed(() => bbCfg.value.meterNames ?? {});
  const dfpTargets   = computed(() => bbCfg.value.dfpTargets ?? {});
  const scenarios    = computed(() => bbCfg.value.scenarios ?? []);

  /**
   * Gap-filling settings. Defaults live in `estimationOptions`, so a config
   * written before this feature existed still gets estimation — which is the
   * point: the dashboards already in the field are the ones with the gaps.
   */
  const estimation = computed(() => estimationOptions(bbCfg.value.estimation));

  // ── End-use categories, from config ──────────────────────────────────────────
  // The union of `meterNames` and `dfpTargets` keys, in config order, so the
  // author controls bar order and a target with no meter yet still draws its
  // reference bar. `_`-prefixed keys are the provenance comments this config
  // family uses in-band, and must not become phantom categories.
  const categories = computed(() => {
    const keys = [...new Set([...Object.keys(meterCfg.value), ...Object.keys(dfpTargets.value)])]
      .filter(k => !k.startsWith('_') && !RESERVED_METER_KEYS.has(k) && !TARGET_ONLY_KEYS.has(k));
    return keys.length ? keys : BB_CATEGORIES;
  });

  const categoryLabels = computed(() => ({
    ...BB_CATEGORY_LABELS,
    ...(bbCfg.value.categoryLabels ?? {})
  }));

  /**
   * @param {string} cat
   * @return {string}
   */
  const labelFor = (cat) => categoryLabels.value[cat] ?? humanizeCategory(cat);

  // ── The configured meters ────────────────────────────────────────────────────
  /**
   * Every configured meter paired with the end use it reports to, in config
   * order: end uses first, then generation and export.
   *
   * One list, built once, because three refreshes and the metadata fetch all need
   * the same set and had drifted into building it three times over.
   *
   * @type {import('vue').ComputedRef<Array<{cat: string, name: string}>>}
   */
  const meterWork = computed(() => [
    ...categories.value.flatMap(cat =>
      (meterCfg.value[cat] ?? []).map(name => ({cat, name}))),
    ...(meterCfg.value.pvGeneration ?? []).map(name => ({cat: 'pvGeneration', name})),
    ...(meterCfg.value.pvExport ?? []).map(name => ({cat: 'pvExport', name}))
  ]);

  /** Distinct meter names, for anything fetched per device rather than per pool. */
  const meterNames = computed(() => [...new Set(meterWork.value.map(w => w.name))]);

  // ── Meter identity, from device metadata ─────────────────────────────────────
  // The EMS specification requires the dashboard and its exports to show each
  // meter's human-readable name, its unique ref and its reporting category, not
  // the Smart Core path. The first two come from the device itself, so they are
  // fetched rather than restated in this dashboard's config.
  /** @type {import('vue').Ref<Object<string, Metadata.AsObject>>} */
  const meterMetadata = ref({});

  /** Name → what to show for it. Present for every configured meter, read or not. */
  const meterIdentities = computed(() =>
    Object.fromEntries(meterNames.value.map(n => [n, meterIdentity(n, meterMetadata.value[n])]))
  );

  /**
   * The human-readable name of a meter, for prose.
   *
   * @param {string} name
   * @return {string}
   */
  const meterLabel = (name) => meterIdentities.value[name]?.label ?? shortMeterName(name);

  /**
   * Fetch metadata for every configured meter.
   *
   * Separate from the reading refreshes and much cheaper: names and refs change
   * when the building is re-commissioned, not every day, and a device whose meter
   * is dead still has a name — which is the whole point, since the meters that
   * need naming most are the ones being reported as unreadable.
   *
   * @return {Promise<void>}
   */
  async function refreshMeterMetadata() {
    const names = meterNames.value;
    const metas = await mapLimit(names, MAX_CONCURRENT_READS, async (name) => {
      // A missing name falls back to the path's last segment, which is worse but
      // still identifies the board. Not worth failing the dashboard over.
      return getMetadata(name).catch(() => null);
    });

    const acc = {};
    names.forEach((n, i) => {
      if (metas[i]) acc[n] = metas[i];
    });
    meterMetadata.value = acc;

    const unnamed = names.filter(n => !meterIdentity(n, acc[n]).hasMetadata);
    if (unnamed.length) {
      console.warn(
        `NABERS Base Building: ${unnamed.length} of ${names.length} meters have no ` +
        'metadata title or ref; showing their Smart Core names', unnamed);
    }
  }

  // ── NABERS rating inputs ─────────────────────────────────────────────────────
  // Rated area, rated hours and postcode are the normalisation inputs the
  // official method needs; `?? null` (not `|| null`) so a genuine 0 is not
  // silently swapped for a default.
  const ratedArea = computed(() => bbCfg.value.nia ?? uiConfig.config?.nabersNIA ?? null);
  const ratedHours = computed(() => bbCfg.value.ratedHours ?? null);
  const postcode = computed(() => bbCfg.value.postcode ?? null);
  const serverRoomAdj = computed(() => bbCfg.value.serverRoomAdj ?? 0);

  const carbonFactor = computed(() =>
    bbCfg.value.carbonFactor ?? uiConfig.config?.nabersCarbonFactor ?? null
  );

  /** The benchmark this building is rated against, from its own inputs. */
  const benchmark = computed(() => adjustedBenchmark({
    postcode:             postcode.value,
    ratedHours:           ratedHours.value,
    serverRoomAdjustment: serverRoomAdj.value
  }));

  /** Which rating inputs are absent, so the gauge can name them. */
  const missingInputs = computed(() => missingRatingInputs({
    // energy is reported separately via `hasMeteredEnergy`; here we only want
    // the configuration-side inputs, so pass a placeholder for energy.
    equivalentKwh: 0,
    ratedArea:     ratedArea.value,
    ratedHours:    ratedHours.value,
    postcode:      postcode.value
  }));

  // ── Metered energy, null-safe ────────────────────────────────────────────────
  // A category with no configured meters contributes nothing and is not an
  // error. A category whose meters are all unreadable is *unknown*, and makes
  // the whole figure unknown — coercing it to 0 would delete the building's
  // largest end use and flatter the rating.
  const configuredCategories = computed(() =>
    categories.value.filter(cat => (meterCfg.value[cat] ?? []).length > 0)
  );

  const hasConfiguredMeters = computed(() => configuredCategories.value.length > 0);

  const unreadableCategories = computed(() =>
    configuredCategories.value.filter(cat => categoryPeriodKwh.value[cat] == null)
  );

  const unreadableLabels = computed(() => unreadableCategories.value.map(labelFor));

  // ── Which individual meters were unreadable ──────────────────────────────────
  // `unreadableCategories` says which end uses are unknown. When an end use is a
  // single zone aggregate that is the whole story, but when it is seventeen
  // distribution boards "Terminal Fans unreadable" is not something an engineer
  // can act on. These name the board.
  /** @type {import('vue').Ref<Array<{name: string, category: string, reason: string}>>} */
  const meterFailures = ref([]);

  /** Names of individual meters that could not be read, in config order. */
  const unreadableMeters = computed(() => meterFailures.value.map(f => f.name));

  /**
   * Why each meter's period figure could not be produced, by name.
   *
   * Distinct from `meterStatuses`, which is a *live* reachability probe. A meter
   * can answer right now and still be unusable for the rating — most commonly
   * because its history does not reach back to the period start — and until this
   * was surfaced the two views contradicted each other: the quality table showed
   * such a meter as OK with a current reading while it silently blanked every
   * figure on the dashboard.
   */
  const meterFailureReasons = computed(() =>
    Object.fromEntries(meterFailures.value.map(f => [f.name, f.reason]))
  );

  /** Unreadable meter names, grouped by end use. */
  const unreadableMetersByCategory = computed(() => {
    const acc = {};
    for (const f of meterFailures.value) {
      (acc[f.category] ??= []).push(f.name);
    }
    return acc;
  });

  /**
   * One line per affected end use, e.g.
   * "Terminal Fans: Level 07 North FCU Board, Level 09 South FCU Board".
   */
  const unreadableMeterLabels = computed(() =>
    Object.entries(unreadableMetersByCategory.value)
      .map(([cat, names]) => `${labelFor(cat)}: ${names.map(meterLabel).join(', ')}`)
  );

  // ── Which meters had their data estimated ────────────────────────────────────
  // The disclosure NABERS requires. A meter appears here when a gap in its
  // history had to be bridged to produce the period-to-date figure; the hours
  // are how much of the period sat inside that gap.
  /** @type {import('vue').Ref<Array<{name: string, category: string, hours: number}>>} */
  const meterEstimates = ref([]);

  /**
   * Meters that answered a live read but have recorded nothing recent.
   *
   * Distinct from both a failure and an estimate. `pkg/auto/history` records a
   * meter reading only when it changes, so a present meter whose accumulator has
   * not moved writes nothing at all — its figure is exact and there is nothing to
   * disclose. It is surfaced because a meter that ought to be consuming and reads
   * idle is a commissioning fault worth chasing.
   *
   * @type {import('vue').Ref<Array<{name: string, category: string, hours: number}>>}
   */
  const meterIdles = ref([]);

  /** Names of idle meters, for the quality table. */
  const meterIdle = computed(() => meterIdles.value.map(i => i.name));

  /**
   * Per-meter data quality over the rating period, for the quality table.
   *
   * `estimatedHours` is the stretch the reported total actually rests on;
   * `unrecordedHours` is how much of the period has no recorded history behind it,
   * which can be substantial even where the total came out exact; `longestGap` is
   * the single widest such stretch and when it was, which is the one an engineer
   * can go and investigate. Populated for every configured meter, not only the
   * ones flagged as estimated.
   *
   * `observedTickKwh` is the meter's measured resolution — the smallest step it
   * has been seen to take — which is what sets its idle threshold.
   *
   * @type {import('vue').Ref<Object<string, {estimatedHours: number,
   *   unrecordedHours: number, longestGap: {from: Date, to: Date, hours: number}|null,
   *   observedTickKwh: number|null, rejectedReadings: number}>>}
   */
  const meterQuality = ref({});

  /** Kept as the quality table's prop name; now carries `unrecordedHours` too. */
  const meterEstimation = computed(() => meterQuality.value);

  /** Estimated meter names grouped by end use, mirroring the unreadable list. */
  const estimatedMetersByCategory = computed(() => {
    const acc = {};
    for (const e of meterEstimates.value) (acc[e.category] ??= []).push(e.name);
    return acc;
  });

  /** One line per affected end use, e.g. "Lifts: Passenger Lift 1, Passenger Lift 2". */
  const estimatedMeterLabels = computed(() =>
    Object.entries(estimatedMetersByCategory.value)
      .map(([cat, names]) => `${labelFor(cat)}: ${names.map(meterLabel).join(', ')}`)
  );

  /** Estimated kWh by end use over the period to date. */
  const categoryEstimatedKwh = ref({});

  /** Gross electricity over the period to date, or null when not yet knowable. */
  const grossPeriodKwh = computed(() => {
    if (!hasConfiguredMeters.value) return null;
    if (unreadableCategories.value.length > 0) return null;
    return configuredCategories.value
      .reduce((acc, cat) => acc + categoryPeriodKwh.value[cat], 0);
  });

  /** Estimated share of the period-to-date figure, as a percentage. */
  const periodEstimatedSharePct = computed(() =>
    estimatedSharePct(configuredCategories.value.map(cat => ({
      kwh:          categoryPeriodKwh.value[cat],
      estimatedKwh: categoryEstimatedKwh.value[cat] ?? 0
    })))
  );

  // ── On-site generation ───────────────────────────────────────────────────────
  // Per the NABERS on-site-renewables ruling only *self-consumed* generation
  // reduces rated electricity; generation exported or on-sold cannot be
  // deducted. With export metering we measure the self-consumed part; without
  // it we fall back to total generation and flag the figure as assumed.
  const hasExportMetering = computed(() => (meterCfg.value.pvExport ?? []).length > 0);

  const pvSelfConsumedKwh = computed(() => {
    if (pvGenerationKwh.value === null) return null;
    if (!hasExportMetering.value) return pvGenerationKwh.value;
    if (pvExportKwh.value === null) return null;
    return Math.max(0, pvGenerationKwh.value - pvExportKwh.value);
  });

  /** True when the PV deduction assumes no export because none is metered. */
  const pvDeductionAssumed = computed(() =>
    (meterCfg.value.pvGeneration ?? []).length > 0 && !hasExportMetering.value
  );

  const netPeriodKwh = computed(() => {
    if (grossPeriodKwh.value === null) return null;
    return Math.max(0, grossPeriodKwh.value - (pvSelfConsumedKwh.value ?? 0));
  });

  // ── The rating period ────────────────────────────────────────────────────────
  // The projection covers the in-progress rating period: the 12 months from the
  // most recent anniversary of the building's rating-period start, so elapsed
  // days never exceed a year. The start is the certificate's anchor when the
  // building is certified, else the configured occupancy start, else the
  // calendar year.
  const ratingPeriodAnchor = computed(() => {
    // An explicit period start wins; otherwise the certificate anchors it. The
    // occupancy certificate is deliberately not used here — it starts the
    // new-build eligibility clock, not the rating period.
    for (const iso of [bbCfg.value.ratingPeriodStart, bbCfg.value.certificateIssueDate]) {
      if (!iso) continue;
      const d = parseISO(iso);
      if (isValid(d)) return d;
    }
    return null;
  });

  const ratingPeriodStart = computed(() => {
    const now = new Date();
    const anchor = ratingPeriodAnchor.value;
    if (!anchor || anchor > now) return new Date(now.getFullYear(), 0, 1);
    // Walk anniversaries forward to the most recent one at or before now.
    let start = anchor;
    while (addYears(start, 1) <= now) start = addYears(start, 1);
    return start;
  });

  const elapsedDays = computed(() =>
    Math.max(1, differenceInDays(new Date(), ratingPeriodStart.value) + 1)
  );

  // ── Projection: a straight-line forecast, not a rating ───────────────────────
  const annualisationFactor = computed(() => 365 / elapsedDays.value);

  /** Whether enough of the period has elapsed for annualising to mean anything. */
  const canProject = computed(() => elapsedDays.value >= MIN_PROJECTION_DAYS);

  /** Equivalent-energy kWh annualised from the period to date. */
  const annualisedEquivalentKwh = computed(() => {
    if (netPeriodKwh.value === null || !canProject.value) return null;
    // All base-building categories are electrical today. Routing through the
    // EEF weighting keeps the maths correct if a gas, thermal or diesel
    // category is added later — delivered kWh of different fuels are never
    // summed 1:1.
    return equivalentEnergyKwh({electricityKwh: netPeriodKwh.value}) * annualisationFactor.value;
  });

  const projectedRating = computed(() => computeRating({
    equivalentKwh:        annualisedEquivalentKwh.value,
    ratedArea:            ratedArea.value,
    ratedHours:           ratedHours.value,
    postcode:             postcode.value,
    serverRoomAdjustment: serverRoomAdj.value
  }));

  // ── Current standing: the rating proper, no annualisation ────────────────────
  const monthlyData    = ref([]);
  const monthlyLoading = ref(false);

  const monthsOfData = computed(() => monthlyData.value.filter(m => m.hasData).length);

  const hasFullRatingPeriod = computed(() =>
    monthlyData.value.length === MONTHS_IN_RATING_PERIOD &&
    monthsOfData.value === MONTHS_IN_RATING_PERIOD
  );

  /** Trailing 12 months of measured net energy, or null until 12 months exist. */
  const trailing12NetKwh = computed(() => {
    if (!hasFullRatingPeriod.value) return null;
    return monthlyData.value.reduce((acc, m) => acc + m.netKwh, 0);
  });

  const standingRating = computed(() => {
    if (trailing12NetKwh.value === null) return null;
    return computeRating({
      equivalentKwh:        equivalentEnergyKwh({electricityKwh: trailing12NetKwh.value}),
      ratedArea:            ratedArea.value,
      ratedHours:           ratedHours.value,
      postcode:             postcode.value,
      serverRoomAdjustment: serverRoomAdj.value
    });
  });

  /** Estimated share of the trailing-12-month figure, as a percentage. */
  const monthlyEstimatedSharePct = computed(() =>
    estimatedSharePct(monthlyData.value.map(m => ({kwh: m.netKwh, estimatedKwh: m.estimatedKwh ?? 0})))
  );

  // ── Trailing months: a basis that survives the turn of the rating period ─────
  // The period-to-date projection falls off a cliff at each anniversary. A period
  // starting 1 August is four days old on 4 August, and annualising four days
  // multiplies them by ninety — so the projection is suppressed and, unless all
  // twelve months happen to be complete, every figure on the dashboard blanks for
  // the next four weeks.
  //
  // The rolling table already holds the measured months behind us, which is a far
  // better basis than four days. Whatever months have data are summed and
  // annualised over the days they actually cover, which is the average measured
  // rate rather than a guess.
  const trailingMonths = computed(() =>
    monthlyData.value.filter(m => m.hasData && m.netKwh !== null)
  );

  const trailingNetKwh = computed(() =>
    trailingMonths.value.reduce((acc, m) => acc + m.netKwh, 0)
  );

  const trailingDaysCovered = computed(() =>
    trailingMonths.value.reduce((acc, m) => acc + getDaysInMonth(m.month), 0)
  );

  /** Enough measured months to annualise from, by the same bar as the projection. */
  const canUseTrailing = computed(() => trailingDaysCovered.value >= MIN_PROJECTION_DAYS);

  const trailingRating = computed(() => {
    if (!canUseTrailing.value) return null;
    return computeRating({
      equivalentKwh:        equivalentEnergyKwh({electricityKwh: trailingNetKwh.value}) *
        (365 / trailingDaysCovered.value),
      ratedArea:            ratedArea.value,
      ratedHours:           ratedHours.value,
      postcode:             postcode.value,
      serverRoomAdjustment: serverRoomAdj.value
    });
  });

  /**
   * The figure the dashboard leads with, and which of the three bases produced it.
   *
   * In order of authority: the settled twelve months, then the current period
   * annualised, then the measured months behind us. The last exists so the turn of
   * a rating period does not blank the dashboard — these figures are indicative,
   * and a rate measured over real months beats one extrapolated from four days.
   */
  const headlineRating = computed(() =>
    standingRating.value ?? projectedRating.value ?? trailingRating.value
  );

  /** @type {import('vue').ComputedRef<'standing'|'projection'|'trailing'|null>} */
  const headlineBasis = computed(() => {
    if (standingRating.value !== null) return 'standing';
    if (projectedRating.value !== null) return 'projection';
    if (trailingRating.value !== null) return 'trailing';
    return null;
  });

  /** Retained for callers that only need "is this the settled figure". */
  const headlineIsProjection = computed(() => headlineBasis.value !== 'standing');

  /**
   * The estimated share of whichever figure the dashboard is actually showing.
   *
   * Tracks the headline rather than picking one basis, because quoting the
   * period-to-date share beside a settled 12-month rating would describe a
   * different number from the one on screen. Only the period-to-date basis reads
   * the period share; both month-based bases read the monthly one.
   */
  const estimatedShare = computed(() =>
    headlineBasis.value === 'projection'
      ? periodEstimatedSharePct.value
      : monthlyEstimatedSharePct.value
  );

  /** Whether anything on screen rests on estimated data. */
  const hasEstimatedData = computed(() => (estimatedShare.value ?? 0) > 0);

  const currentStars     = computed(() => headlineRating.value?.stars ?? null);
  const bandedStars      = computed(() => headlineRating.value?.bandedStars ?? null);
  const totalIntensity   = computed(() => headlineRating.value?.intensity ?? null);

  // ── Star thresholds, derived from the benchmark rather than configured ───────
  const starCeilings = computed(() => {
    if (benchmark.value === null) return {};
    return STAR_RUNGS.reduce((acc, stars) => {
      acc[stars] = intensityForStars(stars, benchmark.value);
      return acc;
    }, {});
  });

  const fiveStarMax = computed(() => starCeilings.value[5] ?? null);
  const fourStarMax = computed(() => starCeilings.value[4] ?? null);

  /**
   * The rating this building is being held to, which is what {@link headroomPct}
   * is measured against. Five stars unless a config says otherwise, and it must
   * name a published rung — there is no intensity ceiling for a figure between
   * two bands, so an unrecognised value falls back rather than blanking the card.
   *
   * Validated against {@link STAR_RUNGS} rather than against `starCeilings`, so
   * the label still reads correctly on a config with no usable benchmark.
   */
  const targetStars = computed(() => {
    const configured = bbCfg.value.targetStars;
    return STAR_RUNGS.includes(configured) ? configured : 5;
  });

  /** The intensity ceiling of the target rating, kWhe/m²·pa. */
  const targetStarMax = computed(() => starCeilings.value[targetStars.value] ?? null);

  /** How much headroom below that ceiling a DfP assessment expects to see kept. */
  const recommendedMarginPct = computed(() => {
    const configured = bbCfg.value.recommendedMarginPct;
    return Number.isFinite(configured) ? configured : DFP_RECOMMENDED_MARGIN_PCT;
  });

  /**
   * One rung above the target, and the ceiling that rung needs — the stretch
   * target, for the card that asks what a better rating would cost rather than
   * whether the committed one holds.
   *
   * Anchored on {@link targetStars}, deliberately *not* on {@link bandedStars}
   * like {@link nextStarTarget} is. The two answer different questions: this one
   * pairs with {@link headroomPct} on the same target, whereas `nextStarTarget`
   * tracks where the building has actually got to and is what the rating gauge
   * narrates. Anchoring both on the achieved rating would have put two cards side
   * by side measuring against different ratings.
   *
   * The rung comes from {@link STAR_RUNGS} rather than from `starCeilings`, for
   * the reason given on `targetStars`: the star *label* has to read correctly
   * even on a config with no usable benchmark, and only the ceiling goes null.
   * `filter().pop()` takes the smallest rung above the target, the rungs being
   * held best-first.
   */
  const stretchTarget = computed(() => {
    const rung = STAR_RUNGS.filter(r => r > targetStars.value).pop() ?? null;
    if (rung === null) return null; // already targeting the top rung
    return {stars: rung, ceiling: starCeilings.value[rung] ?? null};
  });

  /** Headroom below the stretch rung's ceiling. Negative until it is reached. */
  const stretchHeadroomPct = computed(() =>
    computeHeadroomPct(totalIntensity.value, stretchTarget.value?.ceiling ?? null)
  );

  /** The cut that rung needs, in the units the meters report. Floored at 0. */
  const stretchReductionNeeded = computed(() => {
    const ceiling = stretchTarget.value?.ceiling ?? null;
    if (totalIntensity.value === null || ceiling === null) return null;
    return Math.max(0, totalIntensity.value - ceiling);
  });

  /**
   * The next half-star up, and the intensity ceiling it needs.
   *
   * The rungs come from the published bands rather than arithmetic: there is no
   * 0.5-star band, so the step up from 0 stars is 1, not 0.5.
   */
  const nextStarTarget = computed(() => {
    const stars = bandedStars.value;
    if (stars === null || stars >= 6) return null;
    const rungs = Object.keys(starCeilings.value).map(Number).sort((a, b) => a - b);
    const next = rungs.find(r => r > stars) ?? null;
    if (next === null) return null;
    return {stars: next, ceiling: starCeilings.value[next]};
  });

  const reductionNeeded = computed(() => {
    if (totalIntensity.value === null || nextStarTarget.value === null) return null;
    return Math.max(0, totalIntensity.value - nextStarTarget.value.ceiling);
  });

  /**
   * Progress across the band the building currently sits in: 0 % on entering it,
   * 100 % on reaching the next rung's ceiling.
   */
  const progressToNextStar = computed(() => {
    const rating = headlineRating.value;
    if (rating === null || nextStarTarget.value === null) return 0;
    // Ceiling of the rung above (the target) and of the current rung (the floor
    // of this band). The current rung is `bandedStars`, which may be 0 — for
    // which there is no ceiling, so fall back to the worst defined rung.
    const bandTop = nextStarTarget.value.ceiling;
    const bandBottom = starCeilings.value[bandedStars.value] ??
      Math.max(...Object.values(starCeilings.value));
    if (bandTop == null || !Number.isFinite(bandBottom) || bandBottom <= bandTop) return 0;
    const span = bandBottom - bandTop;
    return Math.min(100, Math.max(0, ((bandBottom - rating.intensity) / span) * 100));
  });

  // ── Per-category intensities, for the breakdown chart ────────────────────────
  /**
   * Which window the breakdown intensities annualise.
   *
   * Mirrors the headline's preference — complete months, else the period to date,
   * else the measured months — but deliberately does *not* read `headlineBasis`,
   * because that is null whenever a rating cannot be computed and an intensity
   * needs no postcode or rated hours to be meaningful.
   *
   * Without this the chart annualised the period to date however short it was, so
   * four days after an anniversary it multiplied four days by ninety while the
   * gauge beside it had already fallen back to the measured months. The two
   * disagreed, and the chart was the one that was wrong.
   *
   * @type {import('vue').ComputedRef<'months'|'period'|null>}
   */
  const intensityBasis = computed(() => {
    if (hasFullRatingPeriod.value) return 'months';
    if (canProject.value) return 'period';
    if (canUseTrailing.value) return 'months';
    return null;
  });

  /** Per end use, summed over the months behind us that have data. */
  const trailingCategoryKwh = computed(() => {
    const acc = {};
    for (const cat of categories.value) {
      acc[cat] = trailingMonths.value.reduce((a, m) => a + (m.byCategory?.[cat] ?? 0), 0);
    }
    return acc;
  });

  /**
   * Gross demand and self-consumed generation over the same months.
   *
   * `trailingMonths` only admits rows with data, and a row has data exactly when
   * its gross figure is non-null, so neither sum can silently drop a month.
   */
  const trailingGrossKwh = computed(() =>
    trailingMonths.value.reduce((a, m) => a + m.grossKwh, 0)
  );

  const trailingPvKwh = computed(() =>
    trailingMonths.value.reduce((a, m) => a + (m.pvKwh ?? 0), 0)
  );

  /**
   * The factor that turns the selected window into a year.
   *
   * One computed rather than the expression repeated per widget: an energy split
   * that annualised the period to date while the chart beside it had fallen back
   * to the measured months was two different years presented as one screen.
   */
  const basisAnnualisationFactor = computed(() => {
    if (intensityBasis.value === null) return null;
    if (intensityBasis.value !== 'months') return annualisationFactor.value;
    return trailingDaysCovered.value > 0 ? 365 / trailingDaysCovered.value : null;
  });

  /**
   * Annualise a measured figure over the selected window.
   *
   * @param {number|null} kwh measured over that window
   * @return {number|null} kWh/m²·pa, or null when there is no usable basis
   */
  function annualisedIntensity(kwh) {
    const factor = basisAnnualisationFactor.value;
    if (kwh === null || kwh === undefined || factor === null || !ratedArea.value) return null;
    return (kwh / ratedArea.value) * factor;
  }

  /**
   * The measured kWh behind each intensity — whichever window was selected.
   *
   * Exposed so an export can show the figure the bar was actually derived from.
   * Quoting the period-to-date kWh beside an intensity annualised from the months
   * would be two different measurements presented as one.
   */
  const categoryBasisKwh = computed(() =>
    intensityBasis.value === 'months' ? trailingCategoryKwh.value : categoryPeriodKwh.value
  );

  const categoryIntensities = computed(() => {
    const blank = Object.fromEntries(categories.value.map(cat => [cat, null]));
    if (basisAnnualisationFactor.value === null) return blank;

    const result = {};
    for (const cat of categories.value) {
      result[cat] = annualisedIntensity(categoryBasisKwh.value[cat] ?? null);
    }
    return result;
  });

  // Gross demand and self-consumed generation on the SAME window as the
  // breakdown chart and the rating, which the energy split did not used to get:
  // it read `annualisationFactor` directly, so it always extrapolated the period
  // to date. Seven days into a fresh period that multiplied a week of August by
  // 52 and called it a year, and the solar share it drew was a week of peak
  // summer generation rather than an annual split.
  const pvIntensity = computed(() => annualisedIntensity(
    intensityBasis.value === 'months' ? trailingPvKwh.value : pvSelfConsumedKwh.value
  ));

  /** Gross annualised intensity, before the on-site generation offset. */
  const grossIntensity = computed(() => annualisedIntensity(
    intensityBasis.value === 'months' ? trailingGrossKwh.value : grossPeriodKwh.value
  ));

  // ── Derived reporting figures ────────────────────────────────────────────────
  // Emissions intensity is per plain floor area, deliberately not the rating's
  // normalised denominator, and emission factors never enter the rating.
  const carbonIntensity = computed(() =>
    (totalIntensity.value !== null && carbonFactor.value !== null)
      ? totalIntensity.value * carbonFactor.value
      : null
  );

  /**
   * The on-site generation the DfP design assumed, kWh/m²·pa.
   *
   * Explicit when a config states `dfpTargets.pvGeneration`, otherwise derived
   * from the pair a DfP report already carries: `totalGross` is the design's
   * demand before on-site generation and `total` is after it, so the difference
   * is the generation. At 3CS that is 49.15 − 47.54 = 1.61, and Cundall section
   * 6.9's 27,175 kWh/yr over the 16,903 m² rated area is 1.608 — the same
   * figure, so the reference needs no configuring to be drawn.
   *
   * `pvGeneration` is in RESERVED_METER_KEYS and the category list filters the
   * union of both config objects, so stating it here cannot add a phantom bar to
   * the breakdown chart.
   */
  const dfpPvIntensity = computed(() => {
    const explicit = dfpTargets.value.pvGeneration;
    if (Number.isFinite(explicit)) return explicit > 0 ? explicit : null;
    const gross = dfpTargets.value.totalGross;
    const net   = dfpTargets.value.total;
    if (!Number.isFinite(gross) || !Number.isFinite(net)) return null;
    return gross > net ? gross - net : null;
  });

  /**
   * That generation as a share of the design's gross demand, in percent.
   *
   * The denominator is the design's own gross, not the building's, so this is
   * what the donut would read if the building matched its design exactly.
   */
  const dfpPvSharePct = computed(() => {
    const gross = dfpTargets.value.totalGross;
    if (dfpPvIntensity.value === null || !Number.isFinite(gross) || gross <= 0) return null;
    return (dfpPvIntensity.value / gross) * 100;
  });

  const dfpDiffPct = computed(() => {
    const target = dfpTargets.value.total ?? null;
    if (totalIntensity.value === null || target === null || target === 0) return null;
    return ((totalIntensity.value - target) / target) * 100;
  });

  /**
   * Headroom below the intensity ceiling of the target rating, as a percentage
   * of that ceiling. This is the margin a DfP report quotes per scenario.
   *
   * Measured against the NABERS benchmark, *not* against the DfP modelled total,
   * and the two routinely disagree in direction: at 3CS a building 16.3% over
   * its as-built design prediction still holds 27.3% headroom, because the
   * design was aiming far below a ceiling the stock benchmark places at 76.0
   * kWhe/m². {@link dfpDiffPct} is the comparison against the design; this one
   * only ever answers "does the target rating hold".
   */
  const headroomPct = computed(() =>
    computeHeadroomPct(totalIntensity.value, targetStarMax.value)
  );

  // ── Fetch helpers ────────────────────────────────────────────────────────────
  // Summation is `sumDeltas` from util/meterEstimation.js, which keeps the strict
  // rule this store has always applied — one unknown meter makes the whole pool
  // unknown — while also carrying the estimated-energy accounting through.

  /**
   * Total consumption for a pool of meters across one interval.
   *
   * Per meter, deliberately, rather than differencing two summed boundaries. A
   * sum across meters at one instant is only comparable to a sum at another
   * instant if exactly the same meters contributed to both; when they do not,
   * the difference is not a consumption figure at all. A meter absent at the
   * earlier boundary and present at the later one would drop its entire lifetime
   * cumulative total into a single month.
   *
   * An empty pool is null, not 0. Zero would assert a real month of no
   * consumption: twelve of those would satisfy `hasFullRatingPeriod` and publish
   * a settled six-star rating for a building with no meters configured.
   *
   * @param {string[]} meterNames
   * @param {Date} from
   * @param {Date} to
   * @param {import('@/util/meterBoundaries.js').BoundaryTable} table
   * @return {{delta: import('@/util/meterEstimation.js').BoundaryDelta,
   *           estimated: string[], unreadable: string[],
   *           failures: Array<{name: string, reason: string}>}}
   */
  function poolDelta(meterNames, from, to, table) {
    if (!meterNames?.length) {
      return {delta: sumDeltas([]), estimated: [], unreadable: [], failures: []};
    }
    const estimated = [];
    // Named as well as counted, and each with its OWN reason. `sumDeltas`
    // short-circuits on the first failing meter, so the pool carries one reason
    // however many boards failed — which read as "these 17 meters all did this" when
    // the 17 had done several different things, and left the one that mattered
    // indistinguishable from the sixteen that did not.
    const unreadable = [];
    const failures = [];
    const deltas = meterNames.map(n => {
      const d = boundaryDelta(table.get(n, from), table.get(n, to), from, to, estimation.value);
      if (d.estimated) estimated.push(n);
      if (d.kwh == null) {
        unreadable.push(n);
        failures.push({name: n, reason: d.reason ?? 'unreadable'});
      }
      return d;
    });
    return {delta: sumDeltas(deltas), estimated, unreadable, failures};
  }

  /**
   * Rebuild the rolling 12-month table from month-boundary meter readings.
   *
   * @return {Promise<void>}
   */
  async function refreshMonthly() {
    if (!bbCfg.value.enabled) return;
    const area = ratedArea.value;

    monthlyLoading.value = true;
    try {
      const now         = new Date();
      const monthStarts = Array.from({length: 13}, (_, i) => startOfMonth(subMonths(now, 12 - i)));
      const allMeterNames = categories.value.flatMap(cat => meterCfg.value[cat] ?? []);
      const pvNames       = meterCfg.value.pvGeneration ?? [];
      const exportNames   = meterCfg.value.pvExport ?? [];

      // Names are deduplicated for fetching only; each pool is reassembled from
      // its own raw list below, so a name configured under two categories still
      // counts twice exactly as it did before.
      const table = await readBoundaries(
        [...allMeterNames, ...pvNames, ...exportNames], monthStarts, estimation.value);

      const result = [];
      for (let i = 0; i < 12; i++) {
        const from = monthStarts[i];
        const to   = monthStarts[i + 1];
        const gross = poolDelta(allMeterNames, from, to, table);
        // An unknown generation month deducts nothing, which understates the PV
        // credit rather than overstating it.
        const generationKwh = poolDelta(pvNames, from, to, table).delta.kwh ?? 0;
        const exportedKwh = exportNames.length
          ? (poolDelta(exportNames, from, to, table).delta.kwh ?? 0)
          : 0;
        const pvKwh = Math.max(0, generationKwh - exportedKwh);
        const grossKwh = gross.delta.kwh;
        const netKwh = grossKwh !== null ? Math.max(0, grossKwh - pvKwh) : null;
        // Estimated energy is attributed to the gross figure. Scaling it down by
        // the PV deduction would understate the disclosure, and the PV credit is
        // itself never estimated upward.
        const estimatedKwh = grossKwh !== null ? Math.min(gross.delta.estimatedKwh, netKwh) : 0;
        // Per end use as well as in total. The boundary table is already fetched,
        // so this is arithmetic rather than more queries, and it is what lets the
        // breakdown chart annualise the same window the rating does instead of
        // always extrapolating the period to date.
        //
        // Where `grossKwh` is non-null every meter read — it is a strict sum — so
        // within a month that has data no end use can be unknown.
        const byCategory = {};
        for (const cat of categories.value) {
          const names = meterCfg.value[cat] ?? [];
          // Unmetered contributes a real 0, exactly as the period-to-date figure
          // treats it; configured but unreadable stays null.
          byCategory[cat] = names.length ? poolDelta(names, from, to, table).delta.kwh : 0;
        }
        result.push({
          label:          format(monthStarts[i], 'MMM yy'),
          month:          monthStarts[i],
          grossKwh,
          pvKwh,
          netKwh,
          byCategory,
          totalIntensity: (netKwh !== null && area) ? netKwh / area : null,
          // An estimated month has data — that is the whole point of estimating
          // it — so it counts toward `hasFullRatingPeriod` and lets a rating
          // settle. `quality` is what keeps that honest on screen.
          hasData:        grossKwh !== null,
          quality:        grossKwh === null ? 'missing' : (gross.delta.estimated ? 'estimated' : 'actual'),
          // Why the month has no figure, and which meter caused it. Without this the
          // table showed a bare "✗ Missing" against a month the month-end report had
          // figures for, which reads as the dashboard being broken rather than as one
          // named board having done something.
          //
          // `reason` is the pool's own, which is the first failing meter's; `failures`
          // carries every board with its own. Both, because the summary line wants one
          // sentence and anyone actually fixing this needs all of them.
          reason:         gross.delta.reason,
          unreadableMeters: gross.unreadable,
          failures:       gross.failures,
          estimatedKwh,
          estimatedPct:   (netKwh) ? (estimatedKwh / netKwh) * 100 : 0,
          estimatedHours: gross.delta.estimatedHours,
          estimatedMeters: gross.estimated
        });
      }
      monthlyData.value = result;
    } catch (e) {
      console.warn('NABERS Base Building: monthly refresh failed', e);
    } finally {
      monthlyLoading.value = false;
    }
  }

  // ── Meter quality ────────────────────────────────────────────────────────────
  const meterStatuses = ref({});

  /**
   * Probe every configured meter for connectivity and its latest reading.
   *
   * @return {Promise<void>}
   */
  async function refreshMeterStatuses() {
    const work = meterWork.value;

    const results = await mapLimit(work, MAX_CONCURRENT_READS, async ({cat, name}) => {
      try {
        const reading = await getMeterReading(name);
        return {ok: true, value: reading?.usage ?? null, category: cat};
      } catch (e) {
        return {ok: false, value: null, error: describeRpcError(e), category: cat};
      }
    });

    // Assembled from the ordered work list, so the table renders in config order
    // rather than in whatever order the reads happened to resolve, which
    // reshuffled every row on each refresh.
    const statuses = {};
    work.forEach(({name}, i) => {
      statuses[name] = results[i];
    });
    meterStatuses.value = statuses;
  }

  // No summary computed here any more. A count of live-reachable meters read as an
  // overall health figure while telling only half the story — "7 of 7 OK" appeared
  // beside a dashboard showing no rating, because reachability and period
  // usability are different questions. The quality table counts its own rows,
  // which carry both.

  // ── Period-to-date consumption ───────────────────────────────────────────────
  /**
   * Refresh period-to-date consumption for every configured category.
   *
   * Both ends of the period come from history rather than the live trait. The
   * live `GetMeterReading` returns a dead meter's last cached accumulator with
   * no hint that it is stale, which silently understates consumption and
   * flatters the rating; a history read carries a timestamp, so a meter that
   * stopped a fortnight ago is visible as such and its tail is estimated
   * conservatively instead.
   *
   * @return {Promise<void>}
   */
  async function refresh() {
    if (!bbCfg.value.enabled) return;

    loading.value = true;
    error.value   = null;

    try {
      const pvNames     = meterCfg.value.pvGeneration ?? [];
      const exportNames = meterCfg.value.pvExport ?? [];
      const start = ratingPeriodStart.value;
      const now   = new Date();

      // One flat work list across every end use plus generation, so the limit
      // governs the real total rather than being applied per category.
      const work = meterWork.value;

      // Sampled at every month boundary inside the period, not just its two ends.
      // The total only needs the ends — it is a difference of cumulative readings —
      // but a gap's extent is only as accurate as the sampling. With two probes the
      // opening boundary bracketed across every record in between that was never
      // fetched, so a period whose only real hole was 66 days reported 290 days and
      // 95% estimated. Interior probes bound each gap to about a month, and are the
      // only thing that can see a hole in the middle of the period at all.
      const instants = periodInstants(start, now);
      const table = await readBoundaries(work.map(w => w.name), instants, estimation.value);

      const byCat = new Map();
      const failures = [];
      const estimates = [];
      const idle = [];
      const quality = {};
      work.forEach(({cat, name}) => {
        const d = spanDelta(instants.map(t => table.get(name, t)), start, now, estimation.value);
        if (!byCat.has(cat)) byCat.set(cat, []);
        byCat.get(cat).push(d);
        // Tracked for every meter, not only the estimated ones: history the period
        // has no record of is worth chasing whether or not it happened to leave the
        // total exact, and it is what the quality table's gap column reports.
        quality[name] = {
          estimatedHours:  d.estimatedHours,
          unrecordedHours: d.unrecordedHours,
          longestGap:      d.longestGap,
          // The meter's measured resolution, which sets its own idle threshold.
          // Surfaced so a bad measurement shows itself rather than quietly
          // shifting a threshold, and because it is useful commissioning detail.
          observedTickKwh: table.tickFor(name),
          // Readings a cumulative accumulator could not have produced, dropped
          // before any figure was derived. Recorded because silently discarding
          // data is not something a rating should do without saying so.
          rejectedReadings: table.rejectedFor(name)
        };
        if (d.kwh === null) failures.push({name, category: cat, reason: d.reason ?? 'unreadable'});
        else if (d.estimated) estimates.push({name, category: cat, hours: d.estimatedHours});
        // Reachable but with nothing recent recorded. Its figure is exact — the
        // accumulator has not moved — so this is neither a failure nor an
        // estimate, but it is worth seeing in the quality table.
        else if (table.isIdle(name)) idle.push({name, category: cat, hours: d.unrecordedHours});
      });
      meterQuality.value = quality;

      const results = {};
      const estimatedResults = {};
      for (const cat of categories.value) {
        // Not configured contributes 0 — a real "nothing metered here", as for an
        // unmetered end use. Configured but with *any* meter unreadable is null.
        const deltas = byCat.get(cat);
        const summed = deltas ? sumDeltas(deltas) : null;
        results[cat] = summed ? summed.kwh : 0;
        estimatedResults[cat] = summed ? summed.estimatedKwh : 0;
      }
      categoryPeriodKwh.value    = results;
      categoryEstimatedKwh.value = estimatedResults;

      pvGenerationKwh.value = pvNames.length ? sumDeltas(byCat.get('pvGeneration')).kwh : 0;
      pvExportKwh.value     = exportNames.length ? sumDeltas(byCat.get('pvExport')).kwh : 0;

      meterFailures.value  = failures;
      meterEstimates.value = estimates;
      meterIdles.value     = idle;
      // Deliberately not `error.value`: the section renders an alert in place of
      // the whole dashboard when that is set, so one dead board would blank every
      // end use. The failures are surfaced by name instead, here and in the meter
      // quality table.
      if (failures.length) {
        console.warn(
          `NABERS Base Building: ${failures.length} of ${work.length} meters unreadable`,
          failures.map(f => `${f.name}: ${f.reason}`)
        );
      }
      if (estimates.length) {
        console.info(
          `NABERS Base Building: ${estimates.length} of ${work.length} meters had gaps filled`,
          estimates.map(e => `${e.name}: ${Math.round(e.hours)} h`)
        );
      }
    } catch (e) {
      // Only a programming or config fault can reach here now; meter reads all
      // resolve.
      error.value = e;
    } finally {
      loading.value = false;
    }
  }

  return {
    BB_CATEGORIES,
    categories, configuredCategories, categoryLabels,
    loading, error,
    // rating inputs & benchmark
    ratedArea, ratedHours, postcode, serverRoomAdj, benchmark, missingInputs,
    carbonFactor,
    // energy
    categoryPeriodKwh, grossPeriodKwh, netPeriodKwh,
    pvGenerationKwh, pvSelfConsumedKwh, pvDeductionAssumed, hasExportMetering,
    hasConfiguredMeters, unreadableCategories, unreadableLabels,
    meterFailures, meterFailureReasons,
    unreadableMeters, unreadableMetersByCategory, unreadableMeterLabels,
    // identity
    meterWork, meterNames, meterMetadata, meterIdentities, meterLabel,
    // estimation / disclosure
    estimation, categoryEstimatedKwh, meterEstimates, meterEstimation, meterQuality,
    meterIdles, meterIdle,
    estimatedMetersByCategory, estimatedMeterLabels,
    periodEstimatedSharePct, monthlyEstimatedSharePct, estimatedShare, hasEstimatedData,
    // rating
    standingRating, projectedRating, trailingRating,
    headlineRating, headlineIsProjection, headlineBasis,
    trailingMonths, trailingDaysCovered, canUseTrailing,
    currentStars, bandedStars, totalIntensity,
    monthsOfData, hasFullRatingPeriod, ratingPeriodStart, elapsedDays, canProject,
    starCeilings, fiveStarMax, fourStarMax, nextStarTarget,
    targetStars, targetStarMax, recommendedMarginPct,
    stretchTarget, stretchHeadroomPct, stretchReductionNeeded,
    reductionNeeded, progressToNextStar,
    // reporting
    categoryIntensities, intensityBasis, trailingCategoryKwh, categoryBasisKwh,
    pvIntensity, grossIntensity,
    carbonIntensity, dfpDiffPct, headroomPct,
    dfpPvIntensity, dfpPvSharePct,
    monthlyData, monthlyLoading, meterStatuses,
    // config passthrough
    nia: ratedArea, dfpTargets, scenarios, bbCfg,
    refresh, refreshMonthly, refreshMeterStatuses, refreshMeterMetadata
  };
});
