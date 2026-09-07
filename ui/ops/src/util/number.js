import {isNullOrUndef} from '@/util/types.js';

export const MAX_INT32 = 2147483647; // 2^31 - 1

/**
 * Format a byte count as a human-readable string (B / KB / MB / GB).
 * Returns '—' for null/undefined values.
 *
 * @param {number|null|undefined} n
 * @return {string}
 */
export function formatBytes(n) {
  if (n == null) return '—';
  if (n < 1024) return `${n} B`;
  if (n < 1024 ** 2) return `${(n / 1024).toFixed(1)} KB`;
  if (n < 1024 ** 3) return `${(n / 1024 ** 2).toFixed(1)} MB`;
  return `${(n / 1024 ** 3).toFixed(2)} GB`;
}

/**
 * Cap a number between min and max.
 *
 * @param {number} value
 * @param {number} min
 * @param {number} max
 * @return {number}
 */
export function cap(value, min, max) {
  return Math.max(min, Math.min(max, value));
}

/**
 * Scale a number from one range to another.
 *
 * @param {number} value
 * @param {number} fromMin
 * @param {number} fromMax
 * @param {number} toMin
 * @param {number} toMax
 * @return {number}
 */
export function scale(value, fromMin, fromMax, toMin, toMax) {
  return toMin + (toMax - toMin) * ((value - fromMin) / (fromMax - fromMin));
}

/**
 * Round a number to a specified number of decimal places.
 *
 * @param {number} num
 * @param {number} decimals
 * @return {number}
 */
export function roundTo(num, decimals) {
  if (decimals < 0) {
    return num;
  }

  if (decimals === 0) {
    return Math.round(num);
  }

  const factor = Math.pow(10, decimals);
  return Math.round(num * factor) / factor;
}

/**
 * Returns a string representation of a number, formatted for display.
 *
 * Rounds to at most 1 decimal place below 100, and to a whole number above.
 *
 * @example
 * format(1234.5678) // "1,235"
 * format(99.5)      // "99.5"
 * format(21.5)      // "21.5"
 * format(0)         // "0"
 * format(null)      // "-"
 * format(0.0123)    // "~0"
 *
 * @param {number|null|undefined} num
 * @param {string} [unit]
 * @return {string}
 */
export function format(num, unit = '') {
  const usageStr = (() => {
    if (isNullOrUndef(num)) return '-';
    if (num === 0) return '0';
    const abs = Math.abs(num);
    // Anything below 0.05 rounds to "0.0" at 1dp, which reads as an exact zero.
    if (abs < 0.05) return '~0';
    // toLocaleString rather than toPrecision: toPrecision switches to exponential
    // notation once the rounded exponent reaches the requested precision, so a
    // 99.5 kWh reading used to render as "1.0e+2".
    return num.toLocaleString(undefined, {maximumFractionDigits: abs < 100 ? 1 : 0});
  })();
  if (unit) {
    let sp = ' ';
    if (unit === '%' || unit === '"' || unit === '\'' || unit[0] === '°') {
      sp = '';
    }
    return `${usageStr}${sp}${unit}`;
  }
  return usageStr;
}