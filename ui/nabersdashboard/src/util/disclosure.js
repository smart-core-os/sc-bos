/**
 * What to tell a reader about estimated energy.
 *
 * NABERS permits estimated data only where it is disclosed, so the disclosure is
 * part of the figure rather than decoration around it. It was written out
 * longhand in seven places — the banner, the rating gauge caveat, the monthly
 * report's note and its CSV preamble, the breakdown chart's CSV, the meter
 * quality column help, and the per-month tooltip — and all seven said the same
 * thing:
 *
 * > projected forward past the last reading of a meter that failed a live read,
 * > inflated by a configured uplift so a substituted value cannot understate
 * > consumption
 *
 * That describes one of the two mechanisms that set `estimated`. The other, a
 * register that stepped backwards by less than its allowance, reports the span as
 * **zero** and discloses the size of the step: it understates, which is the exact
 * opposite of what those seven sentences promised, and it is the direction the
 * NABERS method forbids of a substituted value. A month withheld only by a
 * register correction was telling an assessor the figure had been conservatively
 * inflated.
 *
 * So the wording lives here, keyed on the mechanism that
 * {@link module:util/meterEstimation} actually recorded, and the sites render it
 * rather than restating it.
 *
 * @module util/disclosure
 */

/**
 * One sentence naming what was substituted and which way it errs.
 *
 * @param {import('./meterEstimation.js').EstimationKind} kind
 * @return {string}
 */
export function estimationMechanism(kind) {
  switch (kind) {
    case 'projected':
      return 'It was projected forward past the last reading of a meter that failed a live ' +
        'read and is therefore known to be offline, at that meter\'s own mean rate, inflated ' +
        'by a configured uplift so a substituted value cannot understate consumption. The ' +
        'projection is refused outright once the silence exceeds the estimation window.';
    case 'regressed':
      return 'A meter\'s cumulative register stepped backwards by less than its allowance — a ' +
        're-registration, a stale write, or a read landing mid-update — so the affected span ' +
        'reports zero for that meter and discloses the size of the step as the energy at ' +
        'stake. This UNDERSTATES by whatever the meter genuinely consumed, so the figure is ' +
        'indicative only and a submission must use the FM provider\'s verified readings for ' +
        'the affected month.';
    case 'mixed':
      return 'Some of it was projected forward past an offline meter\'s last reading, inflated ' +
        'so it cannot understate; the rest is a cumulative register that stepped backwards by ' +
        'less than its allowance, whose span reports zero and so UNDERSTATES by whatever that ' +
        'meter genuinely consumed. Affected meters are named per board, and a submission must ' +
        'use the FM provider\'s verified readings for the affected months.';
    default:
      // Reached when a caller has energy to disclose but no recorded mechanism,
      // which should not happen. Say the true, weak thing rather than guess.
      return 'It rests on a value that was substituted rather than measured. Affected meters ' +
        'are named per board.';
  }
}

/**
 * The short label used where there is no room for a sentence, such as a table
 * legend or a column heading.
 *
 * @param {import('./meterEstimation.js').EstimationKind} kind
 * @return {string}
 */
export function estimationMechanismLabel(kind) {
  switch (kind) {
    case 'projected': return 'projected past an offline meter';
    case 'regressed': return 'zeroed across a backwards register step';
    case 'mixed':     return 'projected past an offline meter, and zeroed across a backwards step';
    default:          return 'substituted rather than measured';
  }
}

/**
 * What is always true of a stretch with no records, whichever mechanism applied.
 *
 * Carried alongside the estimation disclosure because the two are routinely
 * confused: a carried-forward boundary leaves a visible hole in the history and
 * yet produces an exact figure, so it belongs in the data-quality column and not
 * in the estimated one.
 *
 * @type {string}
 */
export const CARRY_FORWARD_NOTE =
  'Where a meter recorded nothing either side of a month boundary, its accumulator value at ' +
  'that boundary is the last reading before it, carried forward. This is a measurement rather ' +
  'than an estimate: the history automation records a reading only when it changes, so a ' +
  'stretch with no records is a stretch in which every poll read that same value. Consumption ' +
  'is therefore attributed to the month in which the meter was next seen to move, and the total ' +
  'across any set of whole months is unaffected by where the boundary fell.';
