export const MILLISECOND = 1;
export const SECOND = 1000 * MILLISECOND;
export const MINUTE = 60 * SECOND;
export const HOUR = 60 * MINUTE;
export const DAY = 24 * HOUR;

/**
 * Compares two dates in ascending order.
 * Null values are compared after non-null values.
 *
 * @param {Date | null | undefined} a
 * @param {Date | null | undefined} b
 * @return {number}
 */
export function cmpAsc(a, b) {
  if (!a) return 1;
  if (!b) return -1;
  return a.getTime() - b.getTime();
}

/**
 * Compares two dates in descending order.
 * Null values are compared after non-null values.
 *
 * @param {Date | null | undefined} a
 * @param {Date | null | undefined} b
 * @return {number}
 */
export function cmpDesc(a, b) {
  if (!a && !b) return 0;
  if (!a) return 1;
  if (!b) return -1;
  return b.getTime() - a.getTime();
}

/**
 * Truncates the given date to the nearest round.
 * A date of 12:34 rounded down with 15minutes will return 12:30.
 *
 * @param {Date} date
 * @param {number} round
 * @return {Date}
 */
export function roundDown(date, round) {
  return new Date(date.getTime() - (date.getTime() % round));
}

/**
 * Truncates the given date to the nearest round.
 * A date of 12:34 rounded up with 15minutes will return 12:45.
 *
 * @param {Date} date
 * @param {number} round
 * @return {Date}
 */
export function roundUp(date, round) {
  const d = new Date(date.getTime() - (date.getTime() % round));
  if (d < date) {
    return new Date(d.getTime() + round);
  }
  return d;
}

/**
 * Formats the given date as an `in / time ago` string.
 * The date is formatted as `in` when the date is in the future.
 * The date is formatted as `time ago` when the date is in the past.
 * E.g. `in 5 minutes` or `5 minutes ago`.
 *
 * @param {Date} date
 * @param {Date} now
 * @return {string}
 */
export function formatTimeAgo(date, now) {
  return formatTimeAgoRounded(date, now, 0);
}

/**
 * Formats the given date as an `in / time ago` string.
 * The date is formatted as `in` when the date is in the future.
 * The date is formatted as `time ago` when the date is in the past.
 * The granularity sets the smallest unit that will be displayed.
 * E.g. `in 5 minutes` or `5 minutes ago`.
 *
 * @param {Date} date
 * @param {Date} now
 * @param {number} granularity - in milliseconds
 * @return {string}
 */
export function formatTimeAgoRounded(date, now, granularity) {
  let diff = now - date;

  // Adding a small buffer to account for minimal future time differences
  const bufferMs= 1000;
  if (diff < 0 && Math.abs(diff) < bufferMs) {
    diff = 0;
  }

  if (granularity > 0) {
    diff = Math.trunc(diff / granularity) * granularity;
  }

  const rtf = new Intl.RelativeTimeFormat('en', {numeric: 'auto'});

  if (Math.abs(diff) < MINUTE) {
    return rtf.format(-Math.floor(diff / SECOND), 'second');
  } else if (Math.abs(diff) < HOUR) {
    return rtf.format(-Math.floor(diff / MINUTE), 'minute');
  } else if (Math.abs(diff) < DAY) {
    return rtf.format(-Math.floor(diff / HOUR), 'hour');
  } else {
    return rtf.format(-Math.floor(diff / DAY), 'day');
  }
}
