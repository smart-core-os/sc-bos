/**
 * How many meter reads may be in flight at once, per refresh function.
 *
 * There was no limit at all. That was survivable when each end use was a single
 * zone aggregate, but a building that lists its individual meters can easily
 * reach seventy-odd names, and mounting the section then fired `refresh` (2N),
 * `refreshMonthly` (13N) and `refreshMeterStatuses` (N) together — over a
 * thousand concurrent gRPC-web calls, which is itself a reliable way to
 * manufacture the deadline errors the per-meter tolerance has to absorb.
 *
 * Each refresh function gets its own pool rather than sharing one FIFO gate, so
 * the headline rating is not queued behind hundreds of month-boundary reads.
 */
export const MAX_CONCURRENT_READS = 12;

/**
 * Map over `items` with at most `limit` calls to `fn` in flight, preserving
 * input order in the result.
 *
 * A worker pool rather than lockstep batches, so one slow meter delays only its
 * own worker. `fn` must resolve rather than reject: a rejection would kill its
 * worker and leave holes in the result. Every caller wraps a helper that
 * returns a value for failure instead of throwing.
 *
 * @param {T[]} items
 * @param {number} limit
 * @param {function(T, number): Promise<R>} fn
 * @return {Promise<R[]>}
 * @template T,R
 */
export async function mapLimit(items, limit, fn) {
  const out = new Array(items.length);
  let next = 0;
  const worker = async () => {
    for (let i = next++; i < items.length; i = next++) {
      out[i] = await fn(items[i], i);
    }
  };
  // An empty list spawns no workers and resolves immediately.
  await Promise.all(Array.from({length: Math.min(limit, items.length)}, worker));
  return out;
}
