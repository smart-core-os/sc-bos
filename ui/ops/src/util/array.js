import {nextTick} from 'vue';

/**
 * Iterates over large arrays asynchronously
 *
 * @param {Array} array
 * @param {function} cb
 * @param {boolean} batched
 */
export async function iterateLargeArray(array, cb, batched = false) {
  if (batched) {
    for (const batch of array) {
      batch.forEach(cb);
      await nextTick();
    }
    return;
  }

  for (const ele of array) {
    cb(ele);
    await nextTick();
  }
}

/**
 * Maps large arrays asynchronously
 *
 * @param {Array} array
 * @param {function} cb
 * @param {boolean} batched
 * @return {Array}
 */
export async function mapLargeArray(array, cb, batched = false) {
  const ret = [];

  if (batched) {
    for (const batch of array) {
      ret.push(...batch.map(cb));
      await nextTick();
    }

    return ret;
  }

  for (const ele of array) {
    ret.push(cb(ele));
    await nextTick();
  }

  return ret;
}

/**
 * Batches large arrays
 *
 * @param {Array} array
 * @param {number} batchSize
 * @return {Array<Array>}
 */
export function batchLargeArray(array, batchSize = 10) {
  let batches = [];
  for (let i = 0; i < array.length; i += batchSize) {
    batches.push(array.slice(i, i + batchSize));
  }
  return batches;
}