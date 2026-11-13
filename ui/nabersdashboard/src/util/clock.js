/**
 * Wall-clock time, as a reactive value.
 *
 * This exists because `new Date()` inside a `computed` is a trap, and this
 * dashboard walked into it. Vue caches a computed until one of its *reactive*
 * dependencies changes, and the system clock is not one — so a computed built on
 * `new Date()` is evaluated once, on first read, and then holds that instant for
 * as long as the page is open. Config is loaded once and never written again, so
 * nothing in the dependency chain ever dirties it.
 *
 * That is survivable on a page someone opens, reads and closes. This is a
 * fullscreen dashboard designed to be left on a display, so it is exactly the
 * wrong place for it: `elapsedDays` froze at whatever it was when the page
 * loaded while `refresh` went on fetching energy up to a live `now`, so the
 * annualised figures — and the star rating drawn from them — inflated a little
 * further every day. `isAfterHours` never flipped at 17:00. `ratingPeriodStart`
 * never rolled over at the anniversary.
 *
 * So the clock is a ref, and it is ticked. Anything that needs to know what time
 * it is now reads {@link now}. Anything that needs to know what time a figure was
 * measured to should record that instant itself rather than reading this, since
 * the two drift apart between refreshes, and a divisor that does not match its
 * numerator is how the original bug did its damage.
 *
 * @module util/clock
 */

import {ref} from 'vue';

/**
 * How often the clock advances by default.
 *
 * A minute, which is fine for everything reading it: the coarsest consumer wants
 * to know the day has changed and the finest wants to know the hour has. Polling
 * faster would cost re-renders to no purpose.
 */
export const DEFAULT_TICK_MS = 60 * 1000;

/**
 * The current time, reactive.
 *
 * Written only by {@link tick}. Treat it as read-only everywhere else.
 *
 * @type {import('vue').Ref<Date>}
 */
export const now = ref(new Date());

/**
 * Advance the clock to real time.
 *
 * Exported so a caller that is about to read the clock's dependents can make
 * sure it is current first — `refresh` does this before deriving a window — and
 * so tests that pin the system time can bring the ref into line with it.
 *
 * @return {Date} the new value, for callers that want it in hand
 */
export function tick() {
  now.value = new Date();
  return now.value;
}

let _timer = null;
let _users = 0;

/**
 * Start ticking, if nothing else already is.
 *
 * Reference-counted, so two callers starting it independently do not leave a
 * stray interval behind when one of them stops.
 *
 * @param {number} [intervalMs]
 * @return {function(): void} stops this caller's hold on the clock
 */
export function startClock(intervalMs = DEFAULT_TICK_MS) {
  _users++;
  if (_timer === null) {
    tick();
    _timer = setInterval(tick, intervalMs);
  }
  let released = false;
  return () => {
    if (released) return;
    released = true;
    stopClock();
  };
}

/**
 * Release one hold on the clock, stopping it when the last one goes.
 */
export function stopClock() {
  _users = Math.max(0, _users - 1);
  if (_users === 0 && _timer !== null) {
    clearInterval(_timer);
    _timer = null;
  }
}
