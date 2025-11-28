import {timestampToDate} from '@/api/convpb.js';
import {closeResource, newActionTracker, newResourceValue} from '@/api/resource';
import {describeMeterReading, listMeterReadingHistory, pullMeterReading} from '@/api/sc/traits/meter';
import {format} from '@/util/number.js';
import {toQueryObject, watchResource} from '@/util/traits.js';
import {isNullOrUndef} from '@/util/types.js';
import {computed, effectScope, onScopeDispose, reactive, ref, toRefs, toValue, watch} from 'vue';

/**
 * @typedef {import('@smart-core-os/sc-bos-ui-gen/proto/meter_pb').MeterReading} MeterReading
 * @typedef {import('@smart-core-os/sc-bos-ui-gen/proto/meter_pb').MeterReadingSupport} MeterReadingSupport
 * @typedef {import('@smart-core-os/sc-bos-ui-gen/proto/meter_pb').PullMeterReadingsRequest} PullMeterReadingsRequest
 * @typedef {import('@smart-core-os/sc-bos-ui-gen/proto/meter_pb').PullMeterReadingsResponse} PullMeterReadingsResponse
 * @typedef {import('@smart-core-os/sc-bos-ui-gen/proto/meter_pb').DescribeMeterReadingRequest} DescribeMeterReadingRequest
 * @typedef {import('vue').ComputedRef} ComputedRef
 * @typedef {import('vue').Ref} Ref
 * @typedef {import('vue').ToRefs} ToRefs
 * @typedef {import('@/api/resource').ResourceValue} ResourceValue
 * @typedef {import('@/api/resource').ActionTracker} ActionTracker
 */

/**
 * @param {MaybeRefOrGetter<string|PullMeterReadingsRequest.AsObject>} query - The name of the device or a query object
 * @param {MaybeRefOrGetter<boolean>=} paused - Whether to pause the data stream
 * @return {ToRefs<ResourceValue<MeterReading.AsObject, PullMeterReadingsResponse>>}
 */
export function usePullMeterReading(query, paused = false) {
  const resource = reactive(
      /** @type {ResourceValue<MeterReading.AsObject, PullMeterReadingsResponse>} */
      newResourceValue()
  );
  onScopeDispose(() => closeResource(resource));

  const queryObject = computed(() => toQueryObject(query));

  watchResource(
      () => toValue(queryObject),
      () => toValue(paused),
      (req) => {
        pullMeterReading(req, resource);
        return () => closeResource(resource);
      }
  );

  return toRefs(resource);
}

/**
 * @param {MaybeRefOrGetter<string|DescribeMeterReadingRequest.AsObject>} query - The name of the device or a query
 *   object
 * @return {ToRefs<ActionTracker<MeterReadingSupport.AsObject>>}
 */
export function useDescribeMeterReading(query) {
  const tracker = reactive(
      /** @type {ActionTracker<MeterReadingSupport.AsObject>} */
      newActionTracker()
  );

  const queryObject = computed(() => toQueryObject(query));

  watchResource(
      () => toValue(queryObject),
      false,
      (req) => {
        describeMeterReading(req, tracker)
            .catch(() => {}); // errors are tracked by tracker
        return () => closeResource(tracker);
      }
  );

  return toRefs(tracker);
}

/**
 * Converts a usage and optional unit to a string with appropriate precision.
 *
 * @param {number | null | undefined} usage
 * @param {string} [unit]
 * @return {string}
 */
export function usageToString(usage, unit = '') {
  return format(usage, unit)
}

/**
 * @param {MaybeRefOrGetter<MeterReading.AsObject|null>} value
 * @param {MaybeRefOrGetter<MeterReadingSupport.AsObject|null>=} support
 * @return {{
 *   unit: ComputedRef<string>,
 *   usage: ComputedRef<number | undefined>,
 *   usageStr: ComputedRef<string>,
 *   usageAndUnit: ComputedRef<string>,
 *   table: ComputedRef<Array<{label:string, unit:string, value:string}>>
 * }}
 */
export function useMeterReading(value, support = null) {
  const _v = computed(() => toValue(value));
  const _s = computed(() => toValue(support));

  const unit = computed(() => {
    return _s.value?.usageUnit ?? '';
  });

  const usage = computed(() => {
    return _v.value?.usage;
  });
  const usageStr = computed(() => usageToString(usage.value));
  const usageAndUnit = computed(() => {
    let val = usageStr.value;
    if (unit.value) {
      val += ` ${unit.value}`;
    }
    return val;
  });

  const table = computed(() => {
    return [{
      label: 'Usage',
      unit: unit.value,
      value: usageStr.value
    }];
  });

  return {
    unit,
    usage,
    usageStr,
    usageAndUnit,
    table
  };
}

/**
 * Returns a ref containing the estimated meter reading at time t.
 * Name must implement MeterHistory trait aspect.
 * Results can be very inaccurate if the history records for the meter are sparse around time t.
 *
 * @param {import('vue').MaybeRefOrGetter<string>} name
 * @param {import('vue').MaybeRefOrGetter<Date>} t
 * @param {boolean} interpolate - whether to interpolate between readings before and after t, or just use the next reading after t
 * @return {import('vue').ComputedRef<null | MeterReading.AsObject>}
 */
export function useMeterReadingAt(name, t, interpolate = false) {
  const usageAtT = ref(/** @type {null|number} */ null);
  const producedAtT = ref(/** @type {null|number} */ null);
  const readingAtT = computed(() => {
    if (usageAtT.value === null) return null;
    if (producedAtT.value === null) return /** @type {MeterReading.AsObject} */ {usage: usageAtT.value};
    return /** @type {MeterReading.AsObject} */ {usage: usageAtT.value, produced: producedAtT.value};
  });

  const fetching = ref(/** @type {null | {t: Date, cancel: () => void}} */ null);
  watch([() => toValue(name), () => toValue(t)], async ([name, t]) => {
    const cancelled = () => {
      if (fetching.value === null) return true;
      return fetching.value.t.getTime() !== t.getTime();
    };
    if (fetching.value !== null) {
      if (fetching.value.t.getTime() === t.getTime()) {
        return; // already working on it
      }
      fetching.value.cancel();
    }

    // if we don't have all the info, don't do anything
    if (isNullOrUndef(name) || isNullOrUndef(t)) {
      fetching.value = null;
      return;
    }

    // note: currently no way to cancel unary RPCs,
    // see: https://github.com/grpc/grpc-web/issues/946
    fetching.value = {t, cancel: () => {}};
    try {
      let before, after;
      // trying to interpolate with the reading before can really slow things down as startTime is not set in getReadingBefore
      // only do this if we care about the graph being super accurate. for most cases this won't matter
      if (interpolate) {
        [before, after] = await Promise.all([
          getReadingBefore(name, t),
          getReadingOnOrAfter(name, t)
        ]);
      } else {
        [before, after] = await Promise.all([
          null,
          getReadingOnOrAfter(name, t)
        ]);
      }

      if (cancelled()) return;
      usageAtT.value = interpolateReading(before, after, t);
      producedAtT.value = interpolateReading(before, after, t, 'produced');
    } catch (e) {
      if (!cancelled()) {
        console.warn('Failed to get meter', name, 'reading at', t, e.message ?? e);
      }
    } finally {
      if (!cancelled()) {
        fetching.value = null;
      }
    }
  }, {immediate: true});

  return readingAtT;
}

/**
 * Returns an of meter readings, one for each of the times in ts.
 * Times in the future will result in a null returned reading at that index.
 *
 * @param {import('vue').MaybeRefOrGetter<string>} name
 * @param {import('vue').MaybeRefOrGetter<Date[]>} ts
 * @return {import('vue').ComputedRef<Array<null | MeterReading.AsObject>>}
 */
export function useMeterReadingsAt(name, ts) {
  /**
   * @typedef {Object} Watcher
   * @property {function(): void} stop
   * @property {import('vue').ComputedRef<null | MeterReading.AsObject>} reading
   */
  /** @type {Record<number, Watcher>} */
  const trackers = {}; // keyed by Date.getTime()
  onScopeDispose(() => Object.values(trackers).forEach(({stop}) => stop()));

  return computed(() => {
    const _ts = toValue(ts);
    const toStop = Object.fromEntries(Object.keys(trackers).map(k => [k, true]));

    for (const t of _ts) {
      const k = t.getTime();
      const tracker = trackers[k];
      if (tracker) {
        delete toStop[k];
        continue;
      }
      const scope = effectScope();
      scope.run(() => {
        const reading = useMeterReadingAt(name, t);
        trackers[k] = {stop: () => scope.stop(), reading};
      });
    }

    for (const k of Object.keys(toStop)) {
      trackers[k].stop();
      delete trackers[k];
    }

    return _ts.map(t => {
      const k = t.getTime();
      return trackers[k]?.reading.value ?? null;
    });
  });
}

/**
 * Returns the newest meter reading record before t, if there is one.
 *
 * @param {string} name
 * @param {Date} t
 * @return {Promise<MeterReadingRecord.AsObject | undefined>}
 */
async function getReadingBefore(name, t) {
  const res = await listMeterReadingHistory({
    name,
    pageSize: 1,
    orderBy: 'recordTime desc',
    period: {endTime: t} // this could be very slow without a start time
  });
  return res.meterReadingRecordsList?.[0];
};

/**
 * Returns the oldest meter reading record at or after t.
 *
 * @param {string} name
 * @param {Date} t
 * @return {Promise<MeterReadingRecord.AsObject | undefined>}
 */
async function getReadingOnOrAfter(name, t) {
  const res = await listMeterReadingHistory({name, pageSize: 1, period: {startTime: t}});
  return res.meterReadingRecordsList?.[0];
}


/**
 * Returns the estimated meter reading using field, at `at`, between the two known readings.
 *
 * @param {MeterReadingRecord.AsObject | undefined} a
 * @param {MeterReadingRecord.AsObject | undefined} b
 * @param {Date} at
 * @param {string} field
 * @return {number | null}
 */
function interpolateReading(a, b, at, field = 'usage') {
  if (a && b) {
    // meters only decrease if they are reset, if they are reset use the later reading
    if (a.meterReading[field] > b.meterReading[field]) return b.meterReading[field];
    const dt = (at.getTime() - timestampToDate(a.recordTime)) /
        (timestampToDate(b.recordTime) - timestampToDate(a.recordTime));
    return (dt * (b.meterReading[field] - a.meterReading[field])) + a.meterReading[field];
  } else if (a) {
    return a.meterReading[field];
  } else if (b) {
    return b.meterReading[field];
  } else {
    return null;
  }
}