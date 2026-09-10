import {describe, expect, it} from 'vitest';
import {
  DFP_RECOMMENDED_MARGIN_PCT, DFP_SEVERITY_COLOR,
  dfpSeverity, dfpSeverityColor, headroomSeverity
} from './dfpSeverity.js';

/**
 * 3 Chamberlain Square as deployed: a B3 postcode and 61.1 rated hours give a
 * benchmark of 143.45 kWhe/m², so the 5-star ceiling is 76.03 and a building at
 * 55.3 holds 27.3% headroom while sitting 16.3% over its 47.54 design target.
 *
 * This is the case that exposed the three-ladder problem, so it is pinned.
 */
const THREE_CS = {
  intensity:    55.3,
  designTarget: 47.54,
  headroomPct:  27.26,
  worstScenario: 54.25
};

describe('dfpSeverity', () => {
  it('grades 3CS as watch: over design, but the rating is not in question', () => {
    expect(dfpSeverity(THREE_CS)).toBe('watch');
  });

  it('does not escalate to risk merely for clearing every modelled scenario', () => {
    expect(THREE_CS.intensity).toBeGreaterThan(THREE_CS.worstScenario);
    expect(dfpSeverity(THREE_CS)).not.toBe('risk');
  });

  it('is good at or below the design target', () => {
    expect(dfpSeverity({...THREE_CS, intensity: 47.54})).toBe('good');
    expect(dfpSeverity({...THREE_CS, intensity: 40})).toBe('good');
  });

  it('turns to risk once the recommended headroom is gone', () => {
    expect(dfpSeverity({...THREE_CS, headroomPct: 25})).toBe('watch');
    expect(dfpSeverity({...THREE_CS, headroomPct: 24.9})).toBe('risk');
    expect(dfpSeverity({...THREE_CS, headroomPct: -5})).toBe('risk');
  });

  it('reports risk even on target, when the design itself lacks the margin', () => {
    // Meeting a design that was never far enough below the ceiling is still a
    // rating at risk, so risk outranks good rather than the other way round.
    expect(dfpSeverity({intensity: 70, designTarget: 70, headroomPct: 8})).toBe('risk');
  });

  it('honours a site-specific recommended margin', () => {
    expect(dfpSeverity({...THREE_CS, recommendedMarginPct: 30})).toBe('risk');
    expect(dfpSeverity({...THREE_CS, recommendedMarginPct: 10})).toBe('watch');
  });

  it('defaults the recommended margin to the DfP figure', () => {
    expect(DFP_RECOMMENDED_MARGIN_PCT).toBe(25);
    expect(dfpSeverity({intensity: 55.3, designTarget: 47.54, headroomPct: 24}))
      .toBe(dfpSeverity({...THREE_CS, headroomPct: 24, recommendedMarginPct: 25}));
  });

  it('falls back to watch with a design target but no target rating', () => {
    // The tenancy boundary: over its reference, and nothing to say the rating
    // is at risk, so it must not be coloured as failure.
    expect(dfpSeverity({intensity: 55.3, designTarget: 47.54})).toBe('watch');
    expect(dfpSeverity({intensity: 40, designTarget: 47.54})).toBe('good');
  });

  it('is unknown without an intensity, or with nothing to grade against', () => {
    expect(dfpSeverity({})).toBe('unknown');
    expect(dfpSeverity({intensity: null, designTarget: 47.54})).toBe('unknown');
    expect(dfpSeverity({intensity: 55.3})).toBe('unknown');
    expect(dfpSeverity()).toBe('unknown');
  });
});

describe('headroomSeverity', () => {
  it('keeps 3CS green: 27.3% headroom holds the target rating', () => {
    expect(headroomSeverity(THREE_CS.headroomPct)).toBe('good');
  });

  it('agrees with dfpSeverity on where risk starts', () => {
    for (const headroomPct of [40, 25.1, 25, 24.9, 0, -10]) {
      const shared = dfpSeverity({...THREE_CS, headroomPct});
      expect(headroomSeverity(headroomPct) === 'risk').toBe(shared === 'risk');
    }
  });

  it('has no watch band — it only answers whether the rating holds', () => {
    expect(headroomSeverity(26)).toBe('good');
    expect(headroomSeverity(24)).toBe('risk');
  });

  it('is unknown without a figure', () => {
    expect(headroomSeverity(null)).toBe('unknown');
    expect(headroomSeverity(undefined)).toBe('unknown');
    expect(headroomSeverity(NaN)).toBe('unknown');
  });
});

describe('dfpSeverityColor', () => {
  it('maps each rung to its palette entry', () => {
    expect(dfpSeverityColor('good')).toBe(DFP_SEVERITY_COLOR.good);
    expect(dfpSeverityColor('watch')).toBe(DFP_SEVERITY_COLOR.watch);
    expect(dfpSeverityColor('risk')).toBe(DFP_SEVERITY_COLOR.risk);
  });

  it('falls back to the neutral colour, never to a verdict', () => {
    expect(dfpSeverityColor('nonsense')).toBe(DFP_SEVERITY_COLOR.unknown);
    expect(dfpSeverityColor(undefined)).toBe(DFP_SEVERITY_COLOR.unknown);
  });
});
