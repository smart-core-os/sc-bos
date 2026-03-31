import {isNullOrUndef} from '@/util/types.js';

export const MAX_INT32 = 2147483647; // 2^31 - 1

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
 * @example
 * format(1234.5678) // "1,235"
 * format(0)         // "0"
 * format(null)      // "-"
 * format(0.0123)    // "0.012"
 * format(0.000123)  // "~0"
 *
 * @param {number|null|undefined} num
 * @param {string} [unit]
 * @return {string}
 */
export function format(num, unit = '') {
  const usageStr = (() => {
    if (isNullOrUndef(num)) return '-';
    if (num === 0) return '0';
    if (Math.abs(num) < 0.001) return '~0'
    if (Math.abs(num) < 0.01) return num.toPrecision(1);
    if (Math.abs(num) < 100) return num.toPrecision(2);
    return num.toLocaleString(undefined, {maximumFractionDigits: 0});
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