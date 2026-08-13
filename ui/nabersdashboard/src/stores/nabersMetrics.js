import {defineStore} from 'pinia';
import {ref, computed} from 'vue';
import {startOfMonth, subMonths, differenceInDays, getDaysInMonth, format} from 'date-fns';
import {StatusCode} from 'grpc-web';
import {useUiConfigStore} from './uiConfig.js';
import {getMeterReading, getFirstMeterReadingInPeriod} from '@/api/sc/traits/meter.js';
import {mapLimit, MAX_CONCURRENT_READS} from '@/util/concurrency.js';
import {readBoundaries, periodInstants} from '@/util/meterBoundaries.js';
import {
  estimationOptions, boundaryDelta, spanDelta, sumDeltas, estimatedSharePct
} from '@/util/meterEstimation.js';
import {
  computeTenancyRating,
  missingTenancyRatingInputs,
  tenancyAdjustedBenchmark,
  intensityForStars,
  equivalentEnergyKwh
} from '@/util/nabersRating.js';

/** A NABERS rating is a complete measurement over a full 12 months. */
const MONTHS_IN_RATING_PERIOD = 12;

/** Below this many days into the period, annualising is not meaningful. */
const MIN_PROJECTION_DAYS = 28;

export const useNabersMetricsStore = defineStore('nabersdashboard:nabersMetrics', () => {
  const uiConfig = useUiConfigStore();

  const lightingPeriodKwh  = ref(null);
  const equipmentPeriodKwh = ref(null);
  const lightingEstimatedKwh  = ref(0);
  const equipmentEstimatedKwh = ref(0);
  const loading            = ref(false);
  const error              = ref(null);

  const cfg = computed(() => uiConfig.config ?? {});
  const lightingNames  = computed(() => cfg.value.nabersLightingMeterNames ?? []);
  const equipmentNames = computed(() => cfg.value.nabersEquipmentMeterNames ?? []);

  /** Gap-filling settings, shared with the base building boundary. */
  const estimation = computed(() => estimationOptions(cfg.value.nabersEstimation));

  // ── Which meters had their data estimated ────────────────────────────────────
  /** @type {import('vue').Ref<Array<{name: string, stream: string, hours: number}>>} */
  const meterEstimates = ref([]);

  /** @type {import('vue').Ref<Array<{name: string, stream: string, reason: string}>>} */
  const meterFailures = ref([]);

  // ── Rating inputs ────────────────────────────────────────────────────────────
  // The tenancy benchmark needs rated area, rated hours and the occupied
  // workstation count. It needs no postcode: unlike base building, the tenancy
  // method applies no climate correction.
  const ratedArea = computed(() => cfg.value.nabersNIA ?? null);
  const ratedHours = computed(() => cfg.value.nabersRatedHours ?? null);
  const occupiedWorkstations = computed(() => cfg.value.nabersOccupiedWorkstations ?? null);

  const benchmark = computed(() => tenancyAdjustedBenchmark({
    ratedHours:           ratedHours.value,
    ratedArea:            ratedArea.value,
    occupiedWorkstations: occupiedWorkstations.value
  }));

  /** Which configuration-side rating inputs are absent. */
  const missingInputs = computed(() => missingTenancyRatingInputs({
    equivalentKwh: 0,
    ratedArea:     ratedArea.value,
    ratedHours:    ratedHours.value
  }));

  const hasConfiguredMeters = computed(() =>
    lightingNames.value.length > 0 || equipmentNames.value.length > 0
  );

  /**
   * A configured-but-unreadable stream makes the total unknown. Coercing it to
   * zero would delete a whole end use and flatter the rating.
   */
  const unreadableStreams = computed(() => {
    const missing = [];
    if (lightingNames.value.length && lightingPeriodKwh.value === null) missing.push('lighting');
    if (equipmentNames.value.length && equipmentPeriodKwh.value === null) missing.push('equipment');
    return missing;
  });

  const totalPeriodKwh = computed(() => {
    if (!hasConfiguredMeters.value || unreadableStreams.value.length > 0) return null;
    return (lightingPeriodKwh.value ?? 0) + (equipmentPeriodKwh.value ?? 0);
  });

  /** Estimated share of the period-to-date figure, as a percentage. */
  const periodEstimatedSharePct = computed(() => estimatedSharePct([
    {kwh: lightingPeriodKwh.value, estimatedKwh: lightingEstimatedKwh.value},
    {kwh: equipmentPeriodKwh.value, estimatedKwh: equipmentEstimatedKwh.value}
  ]));

  /** Estimated meter names, for the disclosure caveat. */
  const estimatedMeterLabels = computed(() => {
    const acc = {};
    for (const e of meterEstimates.value) (acc[e.stream] ??= []).push(e.name);
    return Object.entries(acc).map(([stream, names]) => `${stream}: ${names.join(', ')}`);
  });

  // ── The rating period ────────────────────────────────────────────────────────
  const ratingPeriodStart = computed(() => {
    const now = new Date();
    return new Date(now.getFullYear(), 0, 1);
  });

  const elapsedDays = computed(() =>
    Math.max(1, differenceInDays(new Date(), ratingPeriodStart.value) + 1)
  );

  const annualisationFactor = computed(() => 365 / elapsedDays.value);
  const canProject = computed(() => elapsedDays.value >= MIN_PROJECTION_DAYS);

  // ── Per-end-use intensities, for the benchmark chart ─────────────────────────
  const lightingIntensity = computed(() =>
    (lightingPeriodKwh.value !== null && ratedArea.value)
      ? (lightingPeriodKwh.value / ratedArea.value) * annualisationFactor.value
      : null
  );

  const equipmentIntensity = computed(() =>
    (equipmentPeriodKwh.value !== null && ratedArea.value)
      ? (equipmentPeriodKwh.value / ratedArea.value) * annualisationFactor.value
      : null
  );

  // ── Projection: a straight-line forecast, not a rating ───────────────────────
  const projectedRating = computed(() => {
    if (totalPeriodKwh.value === null || !canProject.value) return null;
    return computeTenancyRating({
      equivalentKwh: equivalentEnergyKwh({electricityKwh: totalPeriodKwh.value}) *
        annualisationFactor.value,
      ratedArea:            ratedArea.value,
      ratedHours:           ratedHours.value,
      occupiedWorkstations: occupiedWorkstations.value
    });
  });

  // ── Current standing: the rating proper, no annualisation ────────────────────
  const monthlyData    = ref([]);
  const monthlyLoading = ref(false);

  const monthsOfData = computed(() => monthlyData.value.filter(m => m.hasData).length);

  const hasFullRatingPeriod = computed(() =>
    monthlyData.value.length === MONTHS_IN_RATING_PERIOD &&
    monthsOfData.value === MONTHS_IN_RATING_PERIOD
  );

  const trailing12Kwh = computed(() => {
    if (!hasFullRatingPeriod.value) return null;
    return monthlyData.value.reduce((acc, m) => acc + m.totalKwh, 0);
  });

  const standingRating = computed(() => {
    if (trailing12Kwh.value === null) return null;
    return computeTenancyRating({
      equivalentKwh:        equivalentEnergyKwh({electricityKwh: trailing12Kwh.value}),
      ratedArea:            ratedArea.value,
      ratedHours:           ratedHours.value,
      occupiedWorkstations: occupiedWorkstations.value
    });
  });

  /** Estimated share of the trailing-12-month figure, as a percentage. */
  const monthlyEstimatedSharePct = computed(() =>
    estimatedSharePct(monthlyData.value.map(m => ({kwh: m.totalKwh, estimatedKwh: m.estimatedKwh ?? 0})))
  );

  // ── Trailing months: a basis that survives the turn of the rating period ─────
  // Annualising four days multiplies them by ninety, so the projection is
  // suppressed just after each anniversary and — unless all twelve months happen
  // to be complete — every figure blanks for the following four weeks. The
  // measured months behind us are a far better basis than four days.
  const trailingMonths = computed(() =>
    monthlyData.value.filter(m => m.hasData && m.totalKwh !== null)
  );

  const trailingKwh = computed(() =>
    trailingMonths.value.reduce((acc, m) => acc + m.totalKwh, 0)
  );

  const trailingDaysCovered = computed(() =>
    trailingMonths.value.reduce((acc, m) => acc + getDaysInMonth(m.month), 0)
  );

  const canUseTrailing = computed(() => trailingDaysCovered.value >= MIN_PROJECTION_DAYS);

  const trailingRating = computed(() => {
    if (!canUseTrailing.value) return null;
    return computeTenancyRating({
      equivalentKwh:        equivalentEnergyKwh({electricityKwh: trailingKwh.value}) *
        (365 / trailingDaysCovered.value),
      ratedArea:            ratedArea.value,
      ratedHours:           ratedHours.value,
      occupiedWorkstations: occupiedWorkstations.value
    });
  });

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

  const headlineIsProjection = computed(() => headlineBasis.value !== 'standing');

  /** The estimated share of whichever figure the dashboard is actually showing. */
  const estimatedShare = computed(() =>
    headlineBasis.value === 'projection'
      ? periodEstimatedSharePct.value
      : monthlyEstimatedSharePct.value
  );

  const hasEstimatedData = computed(() => (estimatedShare.value ?? 0) > 0);

  const currentStars   = computed(() => headlineRating.value?.stars ?? null);
  const bandedStars    = computed(() => headlineRating.value?.bandedStars ?? null);
  const totalIntensity = computed(() => headlineRating.value?.intensity ?? null);

  // ── Star thresholds, derived from the benchmark rather than configured ───────
  const starCeilings = computed(() => {
    if (benchmark.value === null) return {};
    return [6, 5.5, 5, 4.5, 4, 3.5, 3, 2.5, 2, 1.5, 1].reduce((acc, stars) => {
      acc[stars] = intensityForStars(stars, benchmark.value);
      return acc;
    }, {});
  });

  const nextStarTarget = computed(() => {
    const stars = bandedStars.value;
    if (stars === null || stars >= 6) return null;
    const rungs = Object.keys(starCeilings.value).map(Number).sort((a, b) => a - b);
    const next = rungs.find(r => r > stars) ?? null;
    if (next === null) return null;
    return {stars: next, ceiling: starCeilings.value[next]};
  });

  const nextStarThreshold = computed(() => nextStarTarget.value?.ceiling ?? null);

  const reductionNeeded = computed(() => {
    if (totalIntensity.value === null || nextStarTarget.value === null) return null;
    return Math.max(0, totalIntensity.value - nextStarTarget.value.ceiling);
  });

  const progressToNextStar = computed(() => {
    const rating = headlineRating.value;
    if (rating === null || nextStarTarget.value === null) return 0;
    const bandTop = nextStarTarget.value.ceiling;
    const bandBottom = starCeilings.value[bandedStars.value] ??
      Math.max(...Object.values(starCeilings.value));
    if (bandTop == null || !Number.isFinite(bandBottom) || bandBottom <= bandTop) return 0;
    return Math.min(100, Math.max(0, ((bandBottom - rating.intensity) / (bandBottom - bandTop)) * 100));
  });

  // ── Fetch helpers ────────────────────────────────────────────────────────────
  /**
   * Total consumption for a pool of meters across one interval.
   *
   * `sumDeltas` is strict: one unreadable meter makes the pool unknown. This
   * store previously summed whatever it could read and reported the result as
   * complete, so three dead meters of four looked like a real — and flatteringly
   * low — figure. That was the one failure mode the base building store had
   * already been fixed for, and gap filling makes the strict rule affordable
   * here too: a recoverable gap is now recovered rather than discarded.
   *
   * An empty pool is null, not 0.
   *
   * @param {string[]} meterNames
   * @param {Date} from
   * @param {Date} to
   * @param {import('@/util/meterBoundaries.js').BoundaryTable} table
   * @return {{delta: import('@/util/meterEstimation.js').BoundaryDelta, estimated: string[]}}
   */
  function poolDelta(meterNames, from, to, table) {
    if (!meterNames?.length) return {delta: sumDeltas([]), estimated: []};
    const estimated = [];
    const deltas = meterNames.map(n => {
      const d = boundaryDelta(table.get(n, from), table.get(n, to), from, to, estimation.value);
      if (d.estimated) estimated.push(n);
      return d;
    });
    return {delta: sumDeltas(deltas), estimated};
  }

  /**
   * Rebuild the rolling 12-month table from month-boundary readings.
   *
   * @return {Promise<void>}
   */
  async function refreshMonthly() {
    if (!cfg.value.nabersEnabled || !hasConfiguredMeters.value) return;
    const area = ratedArea.value;

    monthlyLoading.value = true;
    try {
      const now         = new Date();
      const monthStarts = Array.from({length: 13}, (_, i) => startOfMonth(subMonths(now, 12 - i)));

      const table = await readBoundaries(
        [...lightingNames.value, ...equipmentNames.value], monthStarts, estimation.value);

      const result = [];
      for (let i = 0; i < 12; i++) {
        const from = monthStarts[i];
        const to   = monthStarts[i + 1];
        const l = poolDelta(lightingNames.value, from, to, table);
        const e = poolDelta(equipmentNames.value, from, to, table);
        // An unconfigured stream contributes a real 0; a configured but
        // unreadable one contributes null and makes the month unknown.
        const lKwh = lightingNames.value.length ? l.delta.kwh : 0;
        const eKwh = equipmentNames.value.length ? e.delta.kwh : 0;
        const hasData = lKwh !== null && eKwh !== null;
        const estimatedKwh = hasData
          ? (lightingNames.value.length ? l.delta.estimatedKwh : 0) +
            (equipmentNames.value.length ? e.delta.estimatedKwh : 0)
          : 0;
        const totalKwh = hasData ? lKwh + eKwh : null;
        result.push({
          label:              format(monthStarts[i], 'MMM yy'),
          month:              monthStarts[i],
          lightingIntensity:  (lKwh !== null && area) ? lKwh / area : null,
          equipmentIntensity: (eKwh !== null && area) ? eKwh / area : null,
          totalKwh,
          hasData,
          quality:            !hasData ? 'missing' : (estimatedKwh > 0 ? 'estimated' : 'actual'),
          estimatedKwh,
          estimatedPct:       totalKwh ? (estimatedKwh / totalKwh) * 100 : 0,
          estimatedMeters:    [...l.estimated, ...e.estimated]
        });
      }
      monthlyData.value = result;
    } catch (e) {
      console.warn('NABERS: monthly refresh failed', e);
    } finally {
      monthlyLoading.value = false;
    }
  }

  // ── After-hours ──────────────────────────────────────────────────────────────
  const afterHoursKwh = ref(null);

  const isAfterHours = computed(() => {
    const endHour = cfg.value.nabersOperatingHoursEnd ?? 17;
    return new Date().getHours() >= endHour;
  });

  /**
   * Equipment energy consumed since operating hours ended today.
   *
   * @return {Promise<void>}
   */
  async function refreshAfterHours() {
    if (!cfg.value.nabersEnabled || !equipmentNames.value.length) return;

    const endHour = cfg.value.nabersOperatingHoursEnd ?? 17;
    const now     = new Date();

    if (now.getHours() < endHour) {
      afterHoursKwh.value = 0;
      return;
    }

    const opEnd = new Date(now);
    opEnd.setHours(endHour, 0, 0, 0);
    const opEndPlus15 = new Date(opEnd.getTime() + 15 * 60 * 1000);

    // A single evening's standby figure, so this stays on the live trait and the
    // narrow 15-minute window: there is nothing to interpolate over, and a
    // conservative estimate of one evening would be noise, not disclosure.
    const perMeterDeltas = await mapLimit(equipmentNames.value, MAX_CONCURRENT_READS,
      async name => {
        try {
          const [opEndRec, current] = await Promise.all([
            getFirstMeterReadingInPeriod(name, opEnd, opEndPlus15),
            getMeterReading(name)
          ]);
          const base = opEndRec?.meterReading?.usage ?? null;
          if (base === null || current?.usage == null || current.usage < base) return null;
          return current.usage - base;
        } catch (e) {
          if (e?.code !== StatusCode.NOT_FOUND) console.warn('NABERS: after-hours refresh failed', e);
          return null;
        }
      });
    const readable = perMeterDeltas.filter(v => v !== null);
    afterHoursKwh.value = readable.length === perMeterDeltas.length && readable.length > 0
      ? readable.reduce((a, b) => a + b, 0)
      : null;
  }

  // ── Period-to-date consumption ───────────────────────────────────────────────
  /**
   * Refresh period-to-date consumption for lighting and equipment.
   *
   * Both ends come from history rather than the live trait, for the same reason
   * as the base building store: `GetMeterReading` hands back a dead meter's last
   * cached accumulator with no hint that it is stale, which understates
   * consumption silently.
   *
   * Never throws. It used to rethrow anything that was not NOT_FOUND, which
   * rejected the top-level `Promise.all` and set `error` — replacing the whole
   * tenancy section with an alert because one meter hit a transient deadline.
   *
   * @return {Promise<void>}
   */
  async function refresh() {
    if (!cfg.value.nabersEnabled || !hasConfiguredMeters.value) return;

    loading.value = true;
    error.value   = null;

    try {
      const start = ratingPeriodStart.value;
      const now   = new Date();
      const work = [
        ...lightingNames.value.map(name => ({name, stream: 'lighting'})),
        ...equipmentNames.value.map(name => ({name, stream: 'equipment'}))
      ];

      // Every month boundary inside the period, not just its two ends — see
      // `periodInstants`. The energy needs only the ends, but a gap's extent and
      // its share of the energy are only as accurate as the sampling.
      const instants = periodInstants(start, now);
      const table = await readBoundaries(work.map(w => w.name), instants, estimation.value);

      const failures = [];
      const estimates = [];
      /**
       * @param {string[]} names
       * @param {string} stream
       * @return {import('@/util/meterEstimation.js').BoundaryDelta}
       */
      const streamDelta = (names, stream) => {
        const deltas = names.map(name => {
          const d = spanDelta(instants.map(t => table.get(name, t)), start, now, estimation.value);
          if (d.kwh === null) failures.push({name, stream, reason: d.reason ?? 'unreadable'});
          else if (d.estimated) estimates.push({name, stream, hours: d.estimatedHours});
          return d;
        });
        return sumDeltas(deltas);
      };

      const l = streamDelta(lightingNames.value, 'lighting');
      const e = streamDelta(equipmentNames.value, 'equipment');

      lightingPeriodKwh.value     = lightingNames.value.length ? l.kwh : 0;
      equipmentPeriodKwh.value    = equipmentNames.value.length ? e.kwh : 0;
      lightingEstimatedKwh.value  = lightingNames.value.length ? l.estimatedKwh : 0;
      equipmentEstimatedKwh.value = equipmentNames.value.length ? e.estimatedKwh : 0;

      meterFailures.value  = failures;
      meterEstimates.value = estimates;
      if (failures.length) {
        console.warn(
          `NABERS: ${failures.length} of ${work.length} tenancy meters unreadable`,
          failures.map(f => `${f.name}: ${f.reason}`)
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
    // rating inputs & benchmark
    ratedArea, ratedHours, occupiedWorkstations, benchmark, missingInputs,
    hasConfiguredMeters, unreadableStreams,
    // estimation / disclosure
    estimation, meterFailures, meterEstimates, estimatedMeterLabels,
    periodEstimatedSharePct, monthlyEstimatedSharePct, estimatedShare, hasEstimatedData,
    // energy
    lightingIntensity, equipmentIntensity, totalPeriodKwh,
    // rating
    standingRating, projectedRating, trailingRating,
    headlineRating, headlineIsProjection, headlineBasis,
    trailingMonths, trailingDaysCovered, canUseTrailing,
    currentStars, bandedStars, totalIntensity,
    monthsOfData, hasFullRatingPeriod, elapsedDays, canProject,
    starCeilings, nextStarTarget, nextStarThreshold, reductionNeeded, progressToNextStar,
    loading, error, refresh,
    // monthly
    monthlyData, monthlyLoading, refreshMonthly,
    // after-hours
    afterHoursKwh, isAfterHours, refreshAfterHours
  };
});
