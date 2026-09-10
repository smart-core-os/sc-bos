/**
 * The one severity ladder behind every "how is this building doing against its
 * Design for Performance case" colour on the dashboard.
 *
 * It exists because three widgets graded the same building on three different
 * scales. At 3CS on 55.3 kWhe/m² the headroom card read green (27.3% below the
 * 5-star ceiling), "vs DfP target" read red (any overshoot of the modelled
 * total was red), and the scenario gauge read red (past the worst modelled
 * scenario) — three colours for one building, two of them saying "failing"
 * about a rating of 5.55 stars.
 *
 * The ladder they now share:
 *
 *   good  — at or below the DfP modelled total. The design case is being met.
 *   watch — above the modelled total, but still holding the headroom a DfP
 *           assessment expects below the target rating's intensity ceiling.
 *           Worth chasing operationally; the rating is not in question.
 *   risk  — that headroom is gone, so the target rating itself is at risk. This
 *           is the only state that should read as failure.
 *
 * Being past every modelled off-axis scenario is deliberately *not* its own
 * colour. It is a real finding, and the gauge states it in words, but it says
 * little about whether the rating holds: at 3CS the whole scenario spread
 * (45.97 to 54.25 kWhe/m²) sits inside the bottom two thirds of the 5-star
 * band, so clearing every scenario costs about 0.2 stars.
 */

/**
 * The headroom below the target rating's intensity ceiling that a Design for
 * Performance assessment expects a building to keep. Sites override it with
 * `nabersBaseBuilding.recommendedMarginPct`.
 */
export const DFP_RECOMMENDED_MARGIN_PCT = 25;

/** @typedef {'good'|'watch'|'risk'|'unknown'} DfpSeverity */

/** The palette, held here so the cards and the gauge cannot drift apart. */
export const DFP_SEVERITY_COLOR = Object.freeze({
  good:    '#4caf50',
  watch:   '#f7a12c',
  risk:    '#f89c9b',
  unknown: '#ffffff'
});

/**
 * @param {DfpSeverity} severity
 * @return {string} hex colour
 */
export function dfpSeverityColor(severity) {
  return DFP_SEVERITY_COLOR[severity] ?? DFP_SEVERITY_COLOR.unknown;
}

/**
 * Grade a building against its DfP case.
 *
 * Risk is tested before good, so a design whose own modelled total sits inside
 * the recommended margin is reported as at risk even while the building is
 * meeting it. That is the useful answer: the shortfall is in the design, and
 * hitting it exactly still would not secure the rating.
 *
 * `headroomPct` is what separates watch from risk. A caller with no target
 * rating to measure against — the tenancy boundary has none — passes null and
 * gets watch for anything over the design target, which is the honest reading:
 * without a target rating there is no evidence the rating is at risk.
 *
 * @param {Object} opts
 * @param {number|null} [opts.intensity] the building's own figure, kWhe/m²·pa
 * @param {number|null} [opts.designTarget] the DfP modelled total, kWhe/m²·pa
 * @param {number|null} [opts.headroomPct] percent below the target rating's ceiling
 * @param {number} [opts.recommendedMarginPct] the headroom expected, default 25
 * @return {DfpSeverity}
 */
export function dfpSeverity({
  intensity = null,
  designTarget = null,
  headroomPct = null,
  recommendedMarginPct = DFP_RECOMMENDED_MARGIN_PCT
} = {}) {
  if (!Number.isFinite(intensity)) return 'unknown';
  // Nothing to grade against is not the same as grading well.
  if (!Number.isFinite(designTarget) && !Number.isFinite(headroomPct)) return 'unknown';
  if (Number.isFinite(headroomPct) && headroomPct < recommendedMarginPct) return 'risk';
  if (Number.isFinite(designTarget) && intensity <= designTarget) return 'good';
  return 'watch';
}

/**
 * Grade the headroom figure on its own terms.
 *
 * Deliberately not {@link dfpSeverity}: that answers "how is this building
 * doing against its design case", where being over the modelled total is enough
 * for watch. This card answers the narrower "does the target rating hold", and
 * on that question a building over its design but 27% below the ceiling is
 * genuinely good news, not a caution.
 *
 * The two agree on where risk starts, which is the part that was contradicting
 * itself — so the cards can differ without ever disagreeing.
 *
 * @param {number|null} headroomPct percent below the target rating's ceiling
 * @param {number} [recommendedMarginPct] default 25
 * @return {DfpSeverity}
 */
export function headroomSeverity(headroomPct, recommendedMarginPct = DFP_RECOMMENDED_MARGIN_PCT) {
  if (!Number.isFinite(headroomPct)) return 'unknown';
  return headroomPct >= recommendedMarginPct ? 'good' : 'risk';
}
