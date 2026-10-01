import {describe, it, expect} from 'vitest';
import {
  computeRating,
  adjustedBenchmark,
  climateZoneForPostcode,
  climateCorrection,
  ratedHoursFactor,
  equivalentEnergyKwh,
  serverRoomBenchmarkAdjustment,
  intensityForStars,
  headroomPct,
  bandedStarsFromBenchmarkingFactor,
  computeTenancyRating,
  tenancyAdjustedBenchmark,
  tenancyHoursFactor,
  tenancyOccupancyAdjustment,
  tenancyServerRoomKwh,
  EEF
} from './nabersRating.js';

/**
 * These tests pin the rating maths to two independent sources:
 *
 *  1. The official *NABERS UK Reverse and Simple Calculators* workbook (v2.2,
 *     `BB Simple Calc`), whose worked example ships with cached cell values —
 *     matched here to 1e-9. This is the authoritative check.
 *  2. A published Stage 4 DfP assessment for a real office base building, which
 *     states the building's rating, star thresholds and off-axis scenarios. It
 *     rounds intensities to 1 dp, so its decimal stars carry ~±0.03 of slack;
 *     its *margins* pin the thresholds tightly.
 *
 * The inputs for (2) are the assessed building's own, so they must not be
 * "tidied" into round numbers: the expectations below are that report's
 * published outputs for exactly these inputs.
 */

const REPORT_BB = {postcode: 'B3 2DE', ratedArea: 12137, ratedHours: 60};

/**
 * Assert within an explicit absolute tolerance.
 *
 * `toBeCloseTo(x, 1)` means |diff| < 0.05, which is tighter than the report
 * supports: it prints both the intensity and the margin to 1 dp, and those two
 * roundings compound to ~0.1 percentage points.
 *
 * @param {number} actual
 * @param {number} expected
 * @param {number} tolerance
 * @return {void}
 */
function expectWithin(actual, expected, tolerance) {
  expect(Math.abs(actual - expected),
    `${actual} should be within ${tolerance} of ${expected}`).toBeLessThanOrEqual(tolerance);
}

/**
 * Margin of an intensity below a star ceiling, as the DfP report expresses it.
 *
 * An alias for the shipped function rather than a local reimplementation of it:
 * as a copy, the published margins below pinned this file's arithmetic instead of
 * the arithmetic the dashboard actually renders.
 */
const marginPct = headroomPct;

describe('climate normalisation', () => {
  it('maps a postcode to its climate zone using the two-letter area', () => {
    expect(climateZoneForPostcode('WD25 9NH')).toBe(1);
    expect(climateZoneForPostcode('B3 2DE')).toBe(6);       // single-letter area
    expect(climateZoneForPostcode('b3 2de')).toBe(6);       // case-insensitive
    expect(climateZoneForPostcode('EC1A 1BB')).toBe(1);
  });

  it('returns null rather than a default for an unknown or absent postcode', () => {
    expect(climateZoneForPostcode('ZZ9 9ZZ')).toBeNull();
    expect(climateZoneForPostcode('')).toBeNull();
    expect(climateZoneForPostcode(undefined)).toBeNull();
  });

  it('computes the climate correction from degree days', () => {
    expect(climateCorrection(1)).toBeCloseTo(7.4602075, 6);
    expect(climateCorrection(6)).toBeCloseTo(5.499815, 6);
    expect(climateCorrection(99)).toBeNull();
  });
});

describe('rated hours normalisation', () => {
  it('applies the affine hours correction', () => {
    expect(ratedHoursFactor(45)).toBeCloseTo(0.8705, 12);
    expect(ratedHoursFactor(60)).toBeCloseTo(1.004, 12);
  });

  it('is 1.0 at approximately 59.6 hours per week', () => {
    expect(ratedHoursFactor(59.55)).toBeCloseTo(1, 3);
  });

  it('caps at a full week and rejects non-positive hours', () => {
    expect(ratedHoursFactor(200)).toBe(ratedHoursFactor(168));
    expect(ratedHoursFactor(0)).toBeNull();
    expect(ratedHoursFactor(undefined)).toBeNull();
  });
});

describe('equivalent-energy weighting', () => {
  it('weights each fuel by its EEF rather than summing kWh 1:1', () => {
    expect(equivalentEnergyKwh({electricityKwh: 1000})).toBe(1000);
    expect(equivalentEnergyKwh({gasKwh: 1000})).toBe(750);
    expect(equivalentEnergyKwh({districtHeatingKwh: 1000})).toBe(900);
    expect(equivalentEnergyKwh({districtCoolingKwh: 1000})).toBe(400);
  });

  it('converts coal and diesel from mass/volume before weighting', () => {
    // 22.1 MJ/kg ÷ 3.6 × EEF 0.75
    expect(equivalentEnergyKwh({coalKg: 100})).toBeCloseTo(100 * (22.1 / 3.6) * 0.75, 9);
    // 38.6 MJ/L ÷ 3.6 × EEF 0.8
    expect(equivalentEnergyKwh({dieselLitres: 250})).toBeCloseTo(250 * (38.6 / 3.6) * 0.8, 9);
  });

  it('carries the published EEF set', () => {
    expect(EEF).toEqual({
      electricity: 1, gas: 0.75, districtHeating: 0.9,
      districtCooling: 0.4, condenserWater: 0.04, coal: 0.75, diesel: 0.8
    });
  });

  it('treats a server room as a benchmark uplift, not as rated energy', () => {
    expect(serverRoomBenchmarkAdjustment({chilledWaterKwhth: 10000}, 2500))
      .toBeCloseTo((10000 * 0.4) / 2500, 9);
    expect(serverRoomBenchmarkAdjustment(null, 2500)).toBe(0);
  });
});

describe('the official workbook worked example (BB Simple Calc v2.2)', () => {
  const fuels = {
    electricityKwh: 300000, gasKwh: 47000, districtHeatingKwh: 0,
    districtCoolingKwh: 0, coalKg: 0, dieselLitres: 250
  };
  const rating = computeRating({fuels, ratedArea: 2500, ratedHours: 45, postcode: 'WD25 9NH'});

  it('reproduces the benchmark (cell C61)', () => {
    expect(adjustedBenchmark({postcode: 'WD25 9NH', ratedHours: 45}))
      .toBeCloseTo(124.88211062875, 9);
  });

  it('reproduces equivalent kWh (C62) and intensity (C63)', () => {
    expect(equivalentEnergyKwh(fuels)).toBeCloseTo(337394.44444444444, 6);
    expect(rating.intensity).toBeCloseTo(134.95777777777778, 9);
  });

  it('reproduces the benchmarking factor (C64) and both star figures (C65, C66)', () => {
    expect(rating.benchmarkingFactor).toBeCloseTo(108.06814290557658, 9);
    expect(rating.stars).toBeCloseTo(2.9219621729437435, 9);
    expect(rating.bandedStars).toBe(2.5);
  });
});

describe('published DfP assessment: base building baseline', () => {
  const benchmark = adjustedBenchmark(REPORT_BB);

  it('derives the benchmark from postcode and rated hours', () => {
    expect(benchmark).toBeCloseTo(142.066, 3);
  });

  it('reproduces the published 5.0-star threshold of 75.3 kWh/m²', () => {
    expect(intensityForStars(5, benchmark)).toBeCloseTo(75.3, 1);
  });

  it('reproduces the published rating of 5.5 stars / 5.52 decimal', () => {
    // Report "Total with diesel" 675,659.96 kWh over 12,137 m² -> 55.67 kWh/m².
    const rating = computeRating({...REPORT_BB, equivalentKwh: 675659.96});
    expect(rating.intensity).toBeCloseTo(55.67, 2);
    expect(rating.stars).toBeCloseTo(5.52, 2);
    expect(rating.bandedStars).toBe(5.5);
  });

  it('reproduces the published 26.1% margin against 5.0 stars', () => {
    expectWithin(marginPct(55.7, intensityForStars(5, benchmark)), 26.1, 0.1);
  });
});

describe('headroomPct', () => {
  it('goes negative above the ceiling rather than clamping at zero', () => {
    // The reading the stretch-target card depends on: a rung not reached yet is a
    // gap to quantify, and clamping it to 0 would report the gap as closed.
    expect(headroomPct(57.0, 38.01)).toBeLessThan(0);
    expectWithin(headroomPct(76.02, 38.01), -100, 0.01);
  });

  it('is zero exactly on the ceiling', () => {
    expect(headroomPct(76.03, 76.03)).toBe(0);
  });

  it('has no figure without a usable ceiling', () => {
    for (const ceiling of [null, undefined, 0, -10, NaN]) {
      expect(headroomPct(55.3, ceiling)).toBeNull();
    }
  });

  it('has no figure without an intensity', () => {
    for (const intensity of [null, undefined, NaN]) {
      expect(headroomPct(intensity, 76.03)).toBeNull();
    }
  });
});

describe('published DfP assessment: off-axis scenarios', () => {
  const benchmark = adjustedBenchmark(REPORT_BB);
  const ceiling5 = intensityForStars(5, benchmark);
  const ceiling45 = intensityForStars(4.5, benchmark);

  const scenarios = [
    {id: 'S01', intensity: 53.8, decimal: 5.58, banded: 5.5, margin5: 28.6, margin45: null},
    {id: 'S02', intensity: 58.7, decimal: 5.45, banded: 5, margin5: 22.1, margin45: 37.7},
    {id: 'S03', intensity: 64.7, decimal: 5.29, banded: 5, margin5: 14.0, margin45: 31.2},
    {id: 'S04', intensity: 59.3, decimal: 5.44, banded: 5, margin5: 21.3, margin45: 37.0},
    {id: 'S05', intensity: 59.8, decimal: 5.42, banded: 5, margin5: 20.6, margin45: 36.5},
    {id: 'COMB', intensity: 74.2, decimal: 5.00, banded: 5, margin5: 1.5, margin45: 21.1}
  ];

  it.each(scenarios)('$id rates as published', ({intensity, decimal, banded, margin5, margin45}) => {
    const rating = computeRating({...REPORT_BB, equivalentKwh: intensity * REPORT_BB.ratedArea});
    expectWithin(rating.stars, decimal, 0.03);
    expect(rating.bandedStars).toBe(banded);
    expectWithin(marginPct(intensity, ceiling5), margin5, 0.1);
    if (margin45 !== null) {
      expectWithin(marginPct(intensity, ceiling45), margin45, 0.1);
    }
  });
});

describe('star bands', () => {
  it('places each half-star at its published benchmarking factor', () => {
    expect(bandedStarsFromBenchmarkingFactor(26.5)).toBe(6);
    expect(bandedStarsFromBenchmarkingFactor(26.6)).toBe(5.5);
    expect(bandedStarsFromBenchmarkingFactor(53)).toBe(5);
    expect(bandedStarsFromBenchmarkingFactor(159)).toBe(1);
    expect(bandedStarsFromBenchmarkingFactor(159.1)).toBe(0);
  });

  it('spaces the ladder evenly, with the 6-star ceiling one step from zero', () => {
    const benchmark = adjustedBenchmark(REPORT_BB);
    const step = intensityForStars(6, benchmark);
    expect(step).toBeCloseTo(37.65, 2);
    // Each whole star is two half-star steps; 5★ is twice the 6★ ceiling.
    expect(intensityForStars(5, benchmark)).toBeCloseTo(step * 2, 6);
    expect(intensityForStars(4, benchmark)).toBeCloseTo(step * 3, 6);
    expect(intensityForStars(3, benchmark)).toBeCloseTo(step * 4, 6);
  });
});

describe('tenancy boundary', () => {
  it('uses its own hours constant, not the base building one', () => {
    // 0.0089·40 + 0.51 = 0.866, where base building would give 0.826.
    expect(tenancyHoursFactor(40)).toBeCloseTo(0.866, 12);
    expect(ratedHoursFactor(40)).toBeCloseTo(0.826, 12);
  });

  it('adjusts the benchmark by occupancy density', () => {
    // 150 workstations over 2000 m² -> 0.075/m² -> 4000 × (−0.01 + 0.01125) = 5.
    expect(tenancyOccupancyAdjustment({occupiedWorkstations: 150, ratedArea: 2000}))
      .toBeCloseTo(5, 9);
    // Unknown count falls back to 1 per 20 m² -> 4000 × (−0.01 + 0.0075) = −10.
    expect(tenancyOccupancyAdjustment({ratedArea: 2000})).toBeCloseTo(-10, 9);
    expect(tenancyOccupancyAdjustment({ratedArea: 0})).toBeNull();
  });

  describe('the official workbook worked example (Tenancy Simple Calc v2.2)', () => {
    const fuels = {
      electricityKwh: 400000, gasKwh: 47000, districtHeatingKwh: 0,
      districtCoolingKwh: 0, coalKg: 0, dieselLitres: 250
    };
    const args = {ratedArea: 2000, ratedHours: 40, occupiedWorkstations: 150};

    it('reproduces the benchmark (cell C62)', () => {
      expect(tenancyAdjustedBenchmark(args)).toBeCloseTo(66.682, 9);
    });

    it('reproduces intensity (C65), BF (C66) and the decimal rating (C68)', () => {
      const rating = computeTenancyRating({...args, fuels});
      expect(rating.equivalentKwh).toBeCloseTo(437394.44444444444, 6);
      expect(rating.intensity).toBeCloseTo(218.69722222222222, 9);
      expect(rating.benchmarkingFactor).toBeCloseTo(327.97040014130079, 9);
      // The workbook leaves this negative (−5.376); the published scale floors at 0.
      expect(rating.stars).toBe(0);
      expect(rating.bandedStars).toBe(0);
    });
  });

  it('applies no climate correction — the tab computes one but never uses it', () => {
    // Two tenancies identical but for postcode must rate identically.
    const args = {equivalentKwh: 77000, ratedArea: 2013, ratedHours: 50,
      occupiedWorkstations: 134};
    const london = computeTenancyRating({...args, postcode: 'EC1A 1BB'});
    const aberdeen = computeTenancyRating({...args, postcode: 'AB10 1AA'});
    expect(london.benchmark).toBe(aberdeen.benchmark);
    expect(london.stars).toBe(aberdeen.stars);
  });

  it('differs from the base-building benchmark for the same inputs', () => {
    const shared = {ratedArea: 2013, ratedHours: 50};
    const tenancy = tenancyAdjustedBenchmark({...shared, occupiedWorkstations: 134});
    const baseBuilding = adjustedBenchmark({...shared, postcode: 'B3 2DE'});
    // ~68.7 vs ~129.5: applying one to the other would be wrong by ~1.9x.
    expect(tenancy).toBeCloseTo(68.70, 1);
    expect(baseBuilding).toBeCloseTo(129.47, 1);
    expect(baseBuilding / tenancy).toBeGreaterThan(1.8);
  });

  it('counts base-building server-room thermal energy as rated energy', () => {
    // EEF-weighted kWh, NOT divided by area — see the note on tenancyServerRoomKwh.
    expect(tenancyServerRoomKwh({heatingHotWaterKwhth: 10, chilledWaterKwhth: 10,
      condenserWaterKwhth: 10})).toBeCloseTo(10 * 0.9 + 10 * 0.4 + 10 * 0.04, 9);
    expect(tenancyServerRoomKwh(null)).toBe(0);

    const args = {ratedArea: 2000, ratedHours: 40, occupiedWorkstations: 150};
    const without = computeTenancyRating({...args, equivalentKwh: 100000});
    const with_ = computeTenancyRating({...args, equivalentKwh: 100000,
      serverRoomThermal: {chilledWaterKwhth: 20000}});
    expect(with_.equivalentKwh - without.equivalentKwh).toBeCloseTo(8000, 9);
    expect(with_.stars).toBeLessThan(without.stars);
  });

  it('returns null rather than a rating when inputs are missing', () => {
    expect(computeTenancyRating({ratedArea: 2013, ratedHours: 50})).toBeNull();
    expect(computeTenancyRating({equivalentKwh: 77000, ratedHours: 50})).toBeNull();
    expect(computeTenancyRating({equivalentKwh: 77000, ratedArea: 2013})).toBeNull();
  });
});

describe('null-safety: a missing input is never a flattering rating', () => {
  it('returns null when no energy figure is supplied', () => {
    expect(computeRating(REPORT_BB)).toBeNull();
  });

  it('returns null when rated area, rated hours or postcode are missing', () => {
    expect(computeRating({...REPORT_BB, ratedArea: 0, equivalentKwh: 500000})).toBeNull();
    expect(computeRating({...REPORT_BB, ratedHours: undefined, equivalentKwh: 500000})).toBeNull();
    expect(computeRating({...REPORT_BB, postcode: undefined, equivalentKwh: 500000})).toBeNull();
    expect(computeRating({...REPORT_BB, postcode: 'ZZ9 9ZZ', equivalentKwh: 500000})).toBeNull();
  });

  it('still rates a genuine measured zero, which is data rather than absence', () => {
    // Distinct from the null cases above: here the meters read, and read zero.
    expect(computeRating({...REPORT_BB, equivalentKwh: 0}).stars).toBe(6);
  });

  it('clamps the star scale to 0..6', () => {
    expect(computeRating({...REPORT_BB, equivalentKwh: 1}).stars).toBeLessThanOrEqual(6);
    expect(computeRating({...REPORT_BB, equivalentKwh: 1e9}).stars).toBeGreaterThanOrEqual(0);
  });
});
