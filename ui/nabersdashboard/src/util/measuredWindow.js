/**
 * The window an annualised figure was measured over, and how to say it.
 *
 * Every "kWh/m² NIA/yr" on the dashboard is a measured stretch scaled to a year,
 * and which stretch is not obvious from the number: a rolling twelve months and
 * a week of August both render as an annual intensity. So the widgets that draw
 * one state their window, and they state it from here rather than each phrasing
 * it themselves — two widgets describing the same window in different words is
 * how a reader ends up believing they are looking at different measurements.
 *
 * The basis itself is chosen once, by `intensityBasis` in the base building
 * store. This only renders the choice.
 */

/**
 * @typedef {Object} MeasuredWindow
 * @property {number} days how much of a year the figure actually covers
 * @property {string} label a sentence fragment naming the window
 */

/**
 * @param {Object} opts
 * @param {'months'|'period'|null} [opts.intensityBasis] which window was used
 * @param {number} [opts.elapsedDays] days since the rating period began
 * @param {number} [opts.monthsOfData] complete months behind us with data
 * @param {number} [opts.trailingDaysCovered] days those months add up to
 * @return {MeasuredWindow}
 */
export function measuredWindow({
  intensityBasis = 'period',
  elapsedDays = 0,
  monthsOfData = 0,
  trailingDaysCovered = 0
} = {}) {
  if (intensityBasis === 'months') {
    return {
      days:  trailingDaysCovered,
      label: `the ${monthsOfData} measured month(s) behind us (${trailingDaysCovered} days)`
    };
  }
  return {
    days:  elapsedDays,
    label: `the ${elapsedDays} day(s) since the rating period began`
  };
}
