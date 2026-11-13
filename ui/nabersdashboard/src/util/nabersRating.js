/**
 * NABERS UK Base Building energy rating maths.
 *
 * Transcribed from the official *NABERS UK Reverse and Simple Calculators*
 * workbook, `BB Simple Calc` tab. Every constant and formula below is identical
 * in workbook v1.4 and v2.2 — v2.0→v2.2 changed only the Whole Building reverse
 * tab and the scheme administrator's contact details (administration moved from
 * BRE to CIBSE Certification in March 2024) — so they are held here as one
 * versioned set, keyed by {@link NABERS_MODEL_VERSION}.
 *
 * The resulting rating is **indicative**: the method is the official one, but it
 * runs on the building's own meters rather than accredited-Assessor-validated
 * inputs, and it is not a lodged NABERS certificate.
 *
 * A missing input yields `null`, never `0` — a rating computed from absent data
 * would read as market-leading rather than as unknown.
 */

/** Which published constant set the maths below encodes. */
export const NABERS_MODEL_VERSION = 'nabers-uk-calculators-v2.2';

/**
 * Equivalent-energy factors. Delivered kWh of different fuels are never summed
 * 1:1 — each is weighted by its EEF first.
 */
export const EEF = Object.freeze({
  electricity:     1,
  gas:             0.75,
  districtHeating: 0.9,
  districtCooling: 0.4,
  condenserWater:  0.04,
  coal:            0.75,
  diesel:          0.8
});

/**
 * Coal and diesel are metered by mass/volume, so they convert to kWh before
 * EEF weighting: 22.1 MJ/kg and 38.6 MJ/L, at 3.6 MJ per kWh.
 */
export const FUEL_ENERGY_DENSITY = Object.freeze({
  coalKwhPerKg:      22.1 / 3.6,
  dieselKwhPerLitre: 38.6 / 3.6
});

/** B_M(Dataset) — the universal base-building benchmark, kWhe/m²·pa. */
export const UNIVERSAL_BENCHMARK = 136;

/** Climate correction: `0.011·HDD + 0.034·CDD − 26`, degree days base 15.5 °C. */
export const CLIMATE_HDD_COEFFICIENT = 0.011;
export const CLIMATE_CDD_COEFFICIENT = 0.034;
export const CLIMATE_CONSTANT        = -26;

/** Rated-hours correction: `0.0089·h + 0.47`, which is 1.0 at ≈59.6 h/week. */
export const HOURS_COEFFICIENT = 0.0089;
export const HOURS_CONSTANT    = 0.47;

/** The calculator caps rated hours at a full week. */
export const MAX_RATED_HOURS = 168;

/**
 * Stars per benchmarking-factor point, as the literal the official workbook
 * uses: `7 − 3.77358·BF/100`. This is a rounding of `7 − BF/26.5`; the two agree
 * to ~5e-6 stars, and the literal is kept so results match the workbook cell
 * exactly for anyone cross-checking.
 */
export const STARS_INTERCEPT      = 7;
export const STARS_PER_BF_PERCENT = 3.77358;

export const MAX_STARS = 6;

/**
 * Published half-star bands, upper-inclusive on the benchmarking factor.
 *
 * Note the workbook's own `VLOOKUP` disagrees with these labels exactly on a
 * boundary (at BF = 26.5 it returns 5.5 where the band reads 6). The published
 * band definition is used here; the discrepancy only bites on an exact tie.
 */
export const STAR_BANDS = Object.freeze([
  {maxBf: 26.5, stars: 6},
  {maxBf: 39.75, stars: 5.5},
  {maxBf: 53, stars: 5},
  {maxBf: 66.25, stars: 4.5},
  {maxBf: 79.5, stars: 4},
  {maxBf: 92.75, stars: 3.5},
  {maxBf: 106, stars: 3},
  {maxBf: 119.25, stars: 2.5},
  {maxBf: 132.5, stars: 2},
  {maxBf: 145.75, stars: 1.5},
  {maxBf: 159, stars: 1},
  {maxBf: Infinity, stars: 0}
]);

/** The 18 UK climate zones: heating/cooling degree days, base 15.5 °C. */
export const CLIMATE_ZONES = Object.freeze({
  1:  {name: 'London (Thames Valley)', hdd: 1771.6125, cdd: 410.955},
  2:  {name: 'South Eastern', hdd: 2035.72, cdd: 317.44},
  3:  {name: 'Southern', hdd: 1942.82, cdd: 248.125},
  4:  {name: 'South Western', hdd: 1694.105, cdd: 209.605},
  5:  {name: 'Severn Valley', hdd: 1890.885, cdd: 252.435},
  6:  {name: 'Midlands', hdd: 2117.505, cdd: 241.39},
  7:  {name: 'West Pennines', hdd: 2211.675, cdd: 197.855},
  8:  {name: 'North Western', hdd: 2358.1, cdd: 136.625},
  9:  {name: 'Borders', hdd: 2286.585, cdd: 101.525},
  10: {name: 'North Eastern', hdd: 2297.54, cdd: 197.05},
  11: {name: 'East Pennines', hdd: 2112.725, cdd: 250.09},
  12: {name: 'East Anglia', hdd: 2131.65, cdd: 276.625},
  13: {name: 'Glasgow', hdd: 2370.35, cdd: 116.2},
  14: {name: 'Edinburgh', hdd: 2448.805, cdd: 113.565},
  15: {name: 'Aberdeen', hdd: 2513.27, cdd: 91.045},
  16: {name: 'Wales', hdd: 2004.92, cdd: 101.025},
  17: {name: 'Northern Ireland', hdd: 2204.785, cdd: 124.035},
  18: {name: 'North West Scotland', hdd: 2404.055, cdd: 35.085}
});

/** Postcode area → climate zone id. 121 areas, per `Climate_pcode_xref`. */
export const POSTCODE_CLIMATE_ZONE = Object.freeze({
  AL: 1, AB: 15, B: 6, BA: 5, BB: 7, BD: 10, BH: 3, BL: 7, BN: 2, BR: 1, BS: 5, BT: 17, CA: 8,
  CB: 12, CF: 5, CH: 7, CM: 12, CO: 12, CR: 1, CT: 2, CV: 6, CW: 7, DA: 1, DD: 14, DE: 6, DG: 8,
  DH: 10, DL: 10, DN: 11, DT: 3, DY: 6, E: 1, EC: 1, EH: 9, EN: 1, EX: 4, FK: 13, FY: 7, G: 13,
  GL: 5, GU: 1, HA: 1, HD: 11, HG: 10, HP: 1, HR: 6, HS: 18, HU: 11, HX: 11, IG: 1, IP: 12,
  IV: 18, KA: 13, KT: 1, KW: 18, KY: 14, L: 7, LA: 8, LD: 16, LE: 6, LL: 16, LN: 11, LS: 11,
  LU: 1, M: 7, ME: 2, MK: 1, ML: 13, N: 1, NE: 9, NG: 11, NN: 6, NP: 5, NR: 12, NW: 1, OL: 7,
  OX: 1, PA: 13, PE: 12, PH: 14, PL: 4, PO: 3, PR: 7, RG: 1, RH: 2, RM: 1, S: 11, SA: 16, SE: 1,
  SG: 1, SK: 7, SL: 1, SM: 1, SN: 5, SO: 3, SP: 5, SR: 10, SS: 12, ST: 6, SW: 1, SY: 16, TA: 5,
  TD: 9, TF: 6, TN: 2, TQ: 4, TR: 4, TS: 10, TW: 1, UB: 1, W: 1, WA: 7, WC: 1, WD: 1, WF: 11,
  WN: 7, WR: 6, WS: 6, WV: 6, YO: 10, ZE: 18
});

/**
 * Resolve a UK postcode to its climate zone id.
 *
 * Follows the calculator's rule: use the two-letter area prefix when both of the
 * first two characters are letters, otherwise the single-letter prefix.
 *
 * @param {string} postcode
 * @return {number|null} zone id, or null when the postcode is absent/unknown
 */
export function climateZoneForPostcode(postcode) {
  if (typeof postcode !== 'string') return null;
  const pc = postcode.trim().toUpperCase();
  if (pc.length < 1) return null;
  const twoLetterArea = /^[A-Z]{2}/.test(pc);
  const area = twoLetterArea ? pc.slice(0, 2) : pc.slice(0, 1);
  return POSTCODE_CLIMATE_ZONE[area] ?? null;
}

/**
 * Climate correction for a zone, in kWhe/m²·pa, added to the universal benchmark.
 *
 * @param {number|null} zoneId
 * @return {number|null}
 */
export function climateCorrection(zoneId) {
  const zone = CLIMATE_ZONES[zoneId];
  if (!zone) return null;
  return CLIMATE_HDD_COEFFICIENT * zone.hdd + CLIMATE_CDD_COEFFICIENT * zone.cdd + CLIMATE_CONSTANT;
}

/**
 * Shared affine hours multiplier. The coefficient is common to both boundaries;
 * only the constant differs (0.47 base building, 0.51 tenancy).
 *
 * @param {number|null} ratedHours
 * @param {number} constant
 * @return {number|null}
 */
function affineHoursFactor(ratedHours, constant) {
  if (!Number.isFinite(ratedHours) || ratedHours <= 0) return null;
  return HOURS_COEFFICIENT * Math.min(ratedHours, MAX_RATED_HOURS) + constant;
}

/**
 * The rated-hours multiplier applied to the climate-corrected benchmark.
 *
 * @param {number|null} ratedHours hours per week of requested comfort conditions
 * @return {number|null}
 */
export function ratedHoursFactor(ratedHours) {
  return affineHoursFactor(ratedHours, HOURS_CONSTANT);
}

/**
 * The server-room benchmark adjustment: thermal energy serving a server room
 * raises the benchmark rather than counting as rated energy.
 *
 * @param {{heatingHotWaterKwhth?: number, chilledWaterKwhth?: number,
 *          condenserWaterKwhth?: number}} thermal
 * @param {number} ratedArea m²
 * @return {number} kWhe/m²·pa, 0 when there is no server-room thermal energy
 */
export function serverRoomBenchmarkAdjustment(thermal, ratedArea) {
  if (!thermal || !Number.isFinite(ratedArea) || ratedArea <= 0) return 0;
  const weighted =
    (thermal.heatingHotWaterKwhth ?? 0) * EEF.districtHeating +
    (thermal.chilledWaterKwhth ?? 0) * EEF.districtCooling +
    (thermal.condenserWaterKwhth ?? 0) * EEF.condenserWater;
  return weighted / ratedArea;
}

/**
 * B_M(x,h) — the benchmark this building is rated against.
 *
 * @param {Object} opts
 * @param {string} opts.postcode
 * @param {number} opts.ratedHours hours/week
 * @param {number} [opts.serverRoomAdjustment] kWhe/m²·pa, default 0
 * @return {number|null} null when postcode or hours are missing/invalid
 */
export function adjustedBenchmark({postcode, ratedHours, serverRoomAdjustment = 0}) {
  const correction = climateCorrection(climateZoneForPostcode(postcode));
  const hoursFactor = ratedHoursFactor(ratedHours);
  if (correction === null || hoursFactor === null) return null;
  return (UNIVERSAL_BENCHMARK + correction) * hoursFactor + (serverRoomAdjustment ?? 0);
}

/**
 * Total equivalent-energy kWh across the rated fuels.
 *
 * Units are named explicitly because coal and diesel are metered by mass and
 * volume, not energy.
 *
 * @param {{electricityKwh?: number, gasKwh?: number, districtHeatingKwh?: number,
 *          districtCoolingKwh?: number, coalKg?: number, dieselLitres?: number}} fuels
 * @return {number} kWhe
 */
export function equivalentEnergyKwh(fuels) {
  if (!fuels) return 0;
  return (
    (fuels.electricityKwh ?? 0) * EEF.electricity +
    (fuels.gasKwh ?? 0) * EEF.gas +
    (fuels.districtHeatingKwh ?? 0) * EEF.districtHeating +
    (fuels.districtCoolingKwh ?? 0) * EEF.districtCooling +
    (fuels.coalKg ?? 0) * FUEL_ENERGY_DENSITY.coalKwhPerKg * EEF.coal +
    (fuels.dieselLitres ?? 0) * FUEL_ENERGY_DENSITY.dieselKwhPerLitre * EEF.diesel
  );
}

/**
 * BF — the benchmarking factor as a percentage. 100 is exactly on benchmark;
 * lower is better.
 *
 * @param {number|null} intensity kWhe/m²·pa
 * @param {number|null} benchmark kWhe/m²·pa
 * @return {number|null}
 */
export function benchmarkingFactor(intensity, benchmark) {
  if (!Number.isFinite(intensity) || !Number.isFinite(benchmark) || benchmark <= 0) return null;
  return (intensity / benchmark) * 100;
}

/**
 * The continuous decimal star rating, clamped to the 0–6 scale.
 *
 * @param {number|null} bf
 * @return {number|null}
 */
export function starsFromBenchmarkingFactor(bf) {
  if (!Number.isFinite(bf)) return null;
  const raw = STARS_INTERCEPT - (STARS_PER_BF_PERCENT * bf) / 100;
  return Math.min(MAX_STARS, Math.max(0, raw));
}

/**
 * The official half-star figure for a benchmarking factor.
 *
 * @param {number|null} bf
 * @return {number|null}
 */
export function bandedStarsFromBenchmarkingFactor(bf) {
  if (!Number.isFinite(bf)) return null;
  return STAR_BANDS.find(band => bf <= band.maxBf)?.stars ?? 0;
}

/**
 * The intensity a given star rating allows for this benchmark — the inverse of
 * the rating, used to draw thresholds and to state how far off target a building
 * is in the units its meters report.
 *
 * @param {number} stars
 * @param {number|null} benchmark
 * @return {number|null} kWhe/m²·pa ceiling for that rating
 */
export function intensityForStars(stars, benchmark) {
  if (!Number.isFinite(stars) || !Number.isFinite(benchmark) || benchmark <= 0) return null;
  const band = STAR_BANDS.find(b => b.stars === stars);
  if (!band || !Number.isFinite(band.maxBf)) return null;
  return (band.maxBf / 100) * benchmark;
}

/**
 * Margin of an intensity below a ceiling, as a percentage of that ceiling.
 *
 * Lives here rather than in util/dfpSeverity.js because it is rating arithmetic —
 * dfpSeverity's job is grading a figure, not producing one. It was written out
 * separately in the store, in the scenario gauge and in this module's own test,
 * which is three places for one definition to drift in.
 *
 * Goes negative above the ceiling. That is the useful reading when the ceiling is
 * a rung the building has not reached yet rather than one it is holding, so it is
 * deliberately not clamped at zero.
 *
 * @param {number|null} intensity kWhe/m²·pa
 * @param {number|null} ceiling the threshold to measure against, kWhe/m²·pa
 * @return {number|null} percent below the ceiling, null if either is unusable
 */
export function headroomPct(intensity, ceiling) {
  if (!Number.isFinite(intensity) || !Number.isFinite(ceiling) || ceiling <= 0) return null;
  return ((ceiling - intensity) / ceiling) * 100;
}

/**
 * Which required rating inputs are absent. Drives the gauge's
 * "inputs incomplete" state so it can name what is missing rather than showing
 * a fabricated figure.
 *
 * @param {Object} inputs
 * @param {number} [inputs.equivalentKwh] weighted energy over the window
 * @param {number} [inputs.ratedArea] m² NIA
 * @param {number} [inputs.ratedHours] hours/week
 * @param {string} [inputs.postcode] building postcode
 * @return {string[]}
 */
export function missingRatingInputs({equivalentKwh, ratedArea, ratedHours, postcode} = {}) {
  const missing = [];
  if (!Number.isFinite(equivalentKwh)) missing.push('metered energy');
  if (!Number.isFinite(ratedArea) || ratedArea <= 0) missing.push('rated area');
  if (!Number.isFinite(ratedHours) || ratedHours <= 0) missing.push('rated hours');
  if (climateZoneForPostcode(postcode) === null) missing.push('postcode');
  return missing;
}

/**
 * Run the full NABERS UK base-building method.
 *
 * @param {Object} opts
 * @param {Object} [opts.fuels] consumption over the window, by fuel (see
 *   {@link equivalentEnergyKwh}); omit and pass `equivalentKwh` instead if
 *   already weighted
 * @param {number} [opts.equivalentKwh] pre-weighted kWhe, overrides `fuels`
 * @param {number} opts.ratedArea m² NIA
 * @param {number} opts.ratedHours hours/week
 * @param {string} opts.postcode
 * @param {number} [opts.serverRoomAdjustment] kWhe/m²·pa
 * @return {{stars: number, bandedStars: number, benchmarkingFactor: number,
 *           intensity: number, benchmark: number, equivalentKwh: number,
 *           climateZone: number, modelVersion: string}|null}
 *   null when any required input is missing — never a zero-energy rating
 */
export function computeRating({
  fuels,
  equivalentKwh,
  ratedArea,
  ratedHours,
  postcode,
  serverRoomAdjustment = 0
} = {}) {
  const kwhe = Number.isFinite(equivalentKwh) ? equivalentKwh : (fuels ? equivalentEnergyKwh(fuels) : null);
  if (missingRatingInputs({equivalentKwh: kwhe, ratedArea, ratedHours, postcode}).length > 0) return null;

  const benchmark = adjustedBenchmark({postcode, ratedHours, serverRoomAdjustment});
  if (benchmark === null) return null;

  const intensity = kwhe / ratedArea;
  const bf = benchmarkingFactor(intensity, benchmark);
  if (bf === null) return null;

  return {
    stars:              starsFromBenchmarkingFactor(bf),
    bandedStars:        bandedStarsFromBenchmarkingFactor(bf),
    benchmarkingFactor: bf,
    intensity,
    benchmark,
    equivalentKwh:      kwhe,
    climateZone:        climateZoneForPostcode(postcode),
    modelVersion:       NABERS_MODEL_VERSION
  };
}

// ───────────────────────────────────────────────────────────────────────────────
// Tenancy boundary
//
// From the same workbook's `Tenancy Simple Calc` tab. The BF → stars core and the
// EEF set are shared with base building, but the benchmark is built differently:
//
//   * a different universal benchmark (72, not 136);
//   * a different hours constant (0.51, not 0.47);
//   * an occupancy-density adjustment in place of base building's server-room one;
//   * **no climate correction at all** — the tab computes it but never uses it.
//
// Applying the base-building benchmark to a tenancy, or vice versa, is therefore
// wrong by construction; the two must not share a code path.
// ───────────────────────────────────────────────────────────────────────────────

/** B_M(Dataset) for the tenancy boundary, kWhe/m²·pa. */
export const TENANCY_UNIVERSAL_BENCHMARK = 72;

/** Tenancy hours correction: `0.0089·h + 0.51`. */
export const TENANCY_HOURS_CONSTANT = 0.51;

/** Occupancy adjustment: `4000 × (−0.01 + 0.15·d)`, d = workstations per m². */
export const TENANCY_OCCUPANCY_SCALE       = 4000;
export const TENANCY_OCCUPANCY_CONSTANT    = -0.01;
export const TENANCY_OCCUPANCY_COEFFICIENT = 0.15;

/** The calculator's stand-in when the workstation count is unknown. */
export const DEFAULT_WORKSTATIONS_PER_SQM = 1 / 20;

/**
 * The tenancy rated-hours multiplier.
 *
 * @param {number|null} ratedHours hours/week the tenancy is ≥20% occupied
 * @return {number|null}
 */
export function tenancyHoursFactor(ratedHours) {
  return affineHoursFactor(ratedHours, TENANCY_HOURS_CONSTANT);
}

/**
 * The occupancy-density adjustment to the tenancy benchmark.
 *
 * Density is *workstations per m²*. The workbook labels this cell "sqm per
 * occupied workstation" but computes `workstations ÷ area`, so the label is the
 * reciprocal of the quantity; the computation is what is reproduced here.
 *
 * @param {Object} opts
 * @param {number} [opts.occupiedWorkstations] count; omitted falls back to 1 per 20 m²
 * @param {number} opts.ratedArea m² NIA
 * @return {number|null} kWhe/m²·pa adjustment, or null when area is unusable
 */
export function tenancyOccupancyAdjustment({occupiedWorkstations, ratedArea} = {}) {
  if (!Number.isFinite(ratedArea) || ratedArea <= 0) return null;
  const density = (Number.isFinite(occupiedWorkstations) && occupiedWorkstations > 0)
    ? occupiedWorkstations / ratedArea
    : DEFAULT_WORKSTATIONS_PER_SQM;
  return TENANCY_OCCUPANCY_SCALE *
    (TENANCY_OCCUPANCY_CONSTANT + TENANCY_OCCUPANCY_COEFFICIENT * density);
}

/**
 * The benchmark a tenancy is rated against. Note there is no climate term.
 *
 * @param {Object} opts
 * @param {number} opts.ratedHours hours/week
 * @param {number} opts.ratedArea m² NIA
 * @param {number} [opts.occupiedWorkstations] count
 * @return {number|null}
 */
export function tenancyAdjustedBenchmark({ratedHours, ratedArea, occupiedWorkstations} = {}) {
  const hoursFactor = tenancyHoursFactor(ratedHours);
  const occupancy = tenancyOccupancyAdjustment({occupiedWorkstations, ratedArea});
  if (hoursFactor === null || occupancy === null) return null;
  return (TENANCY_UNIVERSAL_BENCHMARK + occupancy) * hoursFactor;
}

/**
 * EEF-weighted server-room thermal energy supplied by the base building, which
 * counts toward the *tenancy's* rated energy.
 *
 * Divergence from the workbook, deliberate: the `Tenancy Simple Calc` tab
 * divides this term by the rated area and then adds it to total kWh, which is
 * subsequently divided by area again — so the contribution lands in kWh/m²
 * rather than kWh and is scaled by 1/area². That is a unit error carried over
 * from the base-building tab, where the equivalent term genuinely is an
 * adjustment to the benchmark. The tab's own note ("to prevent double counting, any
 * energy consumption included in this subsection should not be included in the
 * Energy Consumption section above") makes the intent plain: this is rated
 * energy. It is therefore added here as un-normalised kWhe. Only tenancies with
 * a base-building-fed server room are affected.
 *
 * @param {{heatingHotWaterKwhth?: number, chilledWaterKwhth?: number,
 *          condenserWaterKwhth?: number}} [thermal]
 * @return {number} kWhe
 */
export function tenancyServerRoomKwh(thermal) {
  if (!thermal) return 0;
  return (thermal.heatingHotWaterKwhth ?? 0) * EEF.districtHeating +
    (thermal.chilledWaterKwhth ?? 0) * EEF.districtCooling +
    (thermal.condenserWaterKwhth ?? 0) * EEF.condenserWater;
}

/**
 * Which required tenancy rating inputs are absent.
 *
 * @param {Object} inputs
 * @param {number} [inputs.equivalentKwh] weighted energy over the window
 * @param {number} [inputs.ratedArea] m² NIA
 * @param {number} [inputs.ratedHours] hours/week
 * @return {string[]}
 */
export function missingTenancyRatingInputs({equivalentKwh, ratedArea, ratedHours} = {}) {
  const missing = [];
  if (!Number.isFinite(equivalentKwh)) missing.push('metered energy');
  if (!Number.isFinite(ratedArea) || ratedArea <= 0) missing.push('rated area');
  if (!Number.isFinite(ratedHours) || ratedHours <= 0) missing.push('rated hours');
  return missing;
}

/**
 * Run the NABERS UK tenancy method.
 *
 * @param {Object} opts
 * @param {Object} [opts.fuels] consumption by fuel (see {@link equivalentEnergyKwh})
 * @param {number} [opts.equivalentKwh] pre-weighted kWhe, overrides `fuels`
 * @param {number} opts.ratedArea m² NIA
 * @param {number} opts.ratedHours hours/week
 * @param {number} [opts.occupiedWorkstations] count
 * @param {Object} [opts.serverRoomThermal] base-building-fed server room energy
 * @return {{stars: number, bandedStars: number, benchmarkingFactor: number,
 *           intensity: number, benchmark: number, equivalentKwh: number,
 *           modelVersion: string}|null} null when any required input is missing
 */
export function computeTenancyRating({
  fuels,
  equivalentKwh,
  ratedArea,
  ratedHours,
  occupiedWorkstations,
  serverRoomThermal
} = {}) {
  const base = Number.isFinite(equivalentKwh)
    ? equivalentKwh
    : (fuels ? equivalentEnergyKwh(fuels) : null);
  const kwhe = base === null ? null : base + tenancyServerRoomKwh(serverRoomThermal);
  if (missingTenancyRatingInputs({equivalentKwh: kwhe, ratedArea, ratedHours}).length > 0) return null;

  const benchmark = tenancyAdjustedBenchmark({ratedHours, ratedArea, occupiedWorkstations});
  if (benchmark === null) return null;

  const intensity = kwhe / ratedArea;
  const bf = benchmarkingFactor(intensity, benchmark);
  if (bf === null) return null;

  return {
    stars:              starsFromBenchmarkingFactor(bf),
    bandedStars:        bandedStarsFromBenchmarkingFactor(bf),
    benchmarkingFactor: bf,
    intensity,
    benchmark,
    equivalentKwh:      kwhe,
    modelVersion:       NABERS_MODEL_VERSION
  };
}
