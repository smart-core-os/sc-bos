import {useInterval} from '@/composables/time.js';
import {useMeterReadingAt, usePullMeterReading} from '@/traits/meter/meter.js';
import {startOfDay, startOfHour, startOfMinute, startOfMonth, startOfYear, sub} from 'date-fns';
import {computed, effectScope, onScopeDispose, shallowRef, toValue, watch} from 'vue';

/**
 * @typedef {import('@smart-core-os/sc-bos-ui-gen/proto/smartcore/bos/meter/v1/meter_pb').MeterReading} MeterReading
 */

/**
 * @typedef {Object} MeterWindowOptions
 * @property {import('vue').MaybeRefOrGetter<string>} [period] - 'minute', 'hour', 'day', 'month' or 'year'
 * @property {import('vue').MaybeRefOrGetter<number|string>} [offset] - used via Math.abs, {period: 'day', offset: 1}
 *   means yesterday, and so on
 * @property {import('vue').MaybeRefOrGetter<number>} [refreshInterval] - how often to recompute the window, in ms
 * @property {import('vue').MaybeRefOrGetter<boolean>} [alignToPeriod] - snap the window to calendar boundaries
 * @property {ReturnType<typeof useMeterWindow>} [window] - an existing window to measure over, instead of
 *   building one from the options above. Pass this when several meters must share one window.
 */

// Start of the calendar period, keyed by `period` (e.g. startOfDay -> 12am local time).
const startOfPeriod = {
  minute: startOfMinute,
  hour: startOfHour,
  day: startOfDay,
  month: startOfMonth,
  year: startOfYear,
};

/**
 * Returns the time window a meter delta should be measured over.
 *
 * When `alignToPeriod` is true the window snaps to calendar boundaries for `period` (e.g. from
 * 12am local time for 'day'); when false it is a rolling window of one `period` ending now.
 *
 * @param {MeterWindowOptions} [opts]
 * @return {{
 *   start: import('vue').ComputedRef<Date>,
 *   end: import('vue').ComputedRef<Date>,
 *   endIsLive: import('vue').ComputedRef<boolean>
 * }}
 */
export function useMeterWindow(opts = {}) {
  const {period, offset, refreshInterval, alignToPeriod} = opts;

  const _period = computed(() => toValue(period) ?? 'hour');
  const _offset = computed(() => -Math.abs(parseInt(toValue(offset)) || 0));
  const _alignToPeriod = computed(() => Boolean(toValue(alignToPeriod)));

  // Tick drives the refresh — the window recomputes each interval.
  const tick = useInterval(() => toValue(refreshInterval) ?? 60 * 1000);
  const endIsLive = computed(() => _offset.value === 0);

  // Reference point: `offset` periods before now (offset 0 -> the current period).
  const reference = computed(() => { tick.value; return sub(new Date(), {[`${_period.value}s`]: -_offset.value}); });

  // When aligned, the window is the calendar period containing the reference: it counts
  // from that period's start (e.g. 12am today) up to a live "now" for the current period,
  // or to the start of the next period for a past one. Otherwise it is a rolling window
  // of one `period` ending at the reference.
  const start = computed(() => _alignToPeriod.value
      ? startOfPeriod[_period.value](reference.value)
      : sub(reference.value, {[`${_period.value}s`]: 1}));
  const end = computed(() => {
    if (endIsLive.value) return reference.value;
    return _alignToPeriod.value
        ? startOfPeriod[_period.value](sub(reference.value, {[`${_period.value}s`]: -1}))
        : reference.value;
  });

  return {start, end, endIsLive};
}

/**
 * Returns the meter readings at each end of a `useMeterWindow`.
 *
 * The end reading comes from a live pull when the window ends now, and from history when it
 * ends in the past. The readings are returned rather than a difference so callers can pick
 * the field they care about — see `generatedBetween`.
 *
 * @param {import('vue').MaybeRefOrGetter<string>} name
 * @param {MeterWindowOptions} [opts]
 * @return {{
 *   start: import('vue').ComputedRef<Date>,
 *   end: import('vue').ComputedRef<Date>,
 *   startReading: import('vue').ComputedRef<null | MeterReading.AsObject>,
 *   endReading: import('vue').ComputedRef<null | MeterReading.AsObject>
 * }}
 */
export function useMeterDelta(name, opts = {}) {
  const {start, end, endIsLive} = opts.window ?? useMeterWindow(opts);

  // Meter names are often optional props defaulting to '', which is not a device we can ask about.
  const _name = computed(() => toValue(name) || null);

  const startReading = useMeterReadingAt(_name, start);

  // The end reading swaps source as the window moves between live and historic, so it lives in
  // its own scope we can tear down. It's held via shallowRef so the inner ref isn't unwrapped —
  // useMeterReadingAt returns a read-only computed, which we couldn't write through.
  const endSource = shallowRef(/** @type {null | import('vue').Ref<null | MeterReading.AsObject>} */ null);
  let endScope = null;
  const stopEndScope = () => {
    endScope?.stop();
    endScope = null;
  };
  onScopeDispose(stopEndScope);

  watch([endIsLive, _name], ([endIsLive, name]) => {
    stopEndScope();
    if (!name) {
      endSource.value = null;
      return;
    }
    endScope = effectScope();
    endScope.run(() => {
      if (endIsLive) {
        const {value: reading} = usePullMeterReading(_name);
        endSource.value = reading;
      } else {
        endSource.value = useMeterReadingAt(_name, end);
      }
    });
  }, {immediate: true});

  return {
    start,
    end,
    startReading,
    endReading: computed(() => endSource.value?.value ?? null)
  };
}

/**
 * Energy generated between two readings of a generation meter.
 *
 * A PV meter reports its output in `usage` — either a dedicated generation
 * meter or a zone meterGroup named `generated`. Some setups instead expose it
 * as `produced` on the consumption meter, so fall back to that.
 *
 * @param {null | MeterReading.AsObject} start
 * @param {null | MeterReading.AsObject} end
 * @return {number}
 */
export function generatedBetween(start, end) {
  const usage = (end?.usage ?? 0) - (start?.usage ?? 0);
  if (usage !== 0) return usage;
  return (end?.produced ?? 0) - (start?.produced ?? 0);
}
