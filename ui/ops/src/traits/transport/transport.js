import {closeResource, newActionTracker, newResourceValue} from '@/api/resource.js';
import {describeTransport, listTransportHistory, pullTransport} from '@/api/sc/traits/transport.js';
import {toQueryObject, watchResource} from '@/util/traits.js';
import {isNullOrUndef} from '@/util/types.js';
import {Transport} from '@smart-core-os/sc-bos-ui-gen/proto/transport_pb';
import {computed, onScopeDispose, reactive, ref, toRefs, toValue, watch} from 'vue';

/**
 * @typedef {import('@smart-core-os/sc-bos-ui-gen/proto/transport_pb').DescribeTransportRequest} DescribeTransportRequest
 * @typedef {import('@smart-core-os/sc-bos-ui-gen/proto/transport_pb').PullTransportRequest} PullTransportRequest
 * @typedef {import('@smart-core-os/sc-bos-ui-gen/proto/transport_pb').PullTransportResponse} PullTransportResponse
 * @typedef {import('@smart-core-os/sc-bos-ui-gen/proto/transport_pb').Transport} Transport
 * @typedef {import('@smart-core-os/sc-bos-ui-gen/proto/transport_pb').TransportSupport} TransportSupport
 * @typedef {import('vue').UnwrapNestedRefs} UnwrapNestedRefs
 * @typedef {import('vue').ToRefs} ToRefs
 * @typedef {import('vue').ComputedRef} ComputedRef
 * @typedef {import('vue').MaybeRefOrGetter} MaybeRefOrGetter
 * @typedef {import('@/api/resource').ResourceValue} ResourceValue
 */

/**
 * @param {MaybeRefOrGetter<string|PullTransportRequest.AsObject>} query
 * @param {MaybeRefOrGetter<boolean>=} paused
 * @return {ToRefs<ResourceValue<Transport.AsObject, PullTransportResponse>>}
 */
export function usePullTransport(query, paused = false) {
  const resource = reactive(
      /** @type {ResourceValue<Transport.AsObject, PullTransportResponse>} */
      newResourceValue()
  );
  onScopeDispose(() => closeResource(resource));

  const queryObject = computed(() => toQueryObject(query));

  watchResource(
      () => toValue(queryObject),
      () => toValue(paused),
      (req) => {
        pullTransport(req, resource);
        return () => closeResource(resource);
      }
  );

  return toRefs(resource);
}

/**
 * @param {MaybeRefOrGetter<string|DescribeTransportRequest.AsObject>} query - The name of the device or a query
 *   object
 * @return {ToRefs<ActionTracker<TransportSupport.AsObject>>}
 */
export function useDescribeTransport(query) {
  const tracker = reactive(
      /** @type {ActionTracker<TransportSupport.AsObject>} */
      newActionTracker()
  );

  const queryObject = computed(() => toQueryObject(query));

  watchResource(
      () => toValue(queryObject),
      false,
      (req) => {
        describeTransport(req, tracker);
        return () => closeResource(tracker);
      }
  );

  return toRefs(tracker);
}

/**
 * Returns a ref containing the transport history for the given transport name for the given period.
 * Name must implement a TransportHistory trait aspect.
 *
 * @param {import('vue').MaybeRefOrGetter<string>} name - The name of the transport device
 * @param {import('vue').MaybeRefOrGetter<Period.AsObject>} period - The start, end period
 * @return {import('vue').UnwrapRef<Array<TransportRecord.AsObject|null>>}
 */
export function useTransportHistory(name, period) {
  const transportFromT = ref(/** @type {Array<null | TransportRecord.AsObject>} */ []);
  const fetching = ref(/**@type {null | {period: Period.AsObject, cancel: () => void}}*/ null);

  const periodsDifferent = (a, b) => {
    return a.startTime.seconds !== b.startTime.seconds &&
    a.endTime.seconds !== b.endTime.seconds;
  };
  watch([() => toValue(name), () => toValue(period)], async ([name, period]) => {
    const cancelled = () => {
      if (fetching.value === null) return true;
      return periodsDifferent(fetching.value.period, period);
    };

    if (fetching.value !== null) {
      if (!periodsDifferent(fetching.value.period, period)) {
        // already fetching this timestamp
        return;
      }
      // cancel the previous fetch
      fetching.value.cancel();
    }

    // if we don't have a name or timestamp, clear the history
    if (isNullOrUndef(name) || isNullOrUndef(period)) {
      fetching.value = null;
      return;
    }
    transportFromT.value = [];

    // note: currently no way to cancel unary RPCs,
    // see: https://github.com/grpc/grpc-web/issues/946
    fetching.value = {period, cancel: () => {}};
    try {
      const req = {
        name,
        pageSize: 1000,
        period: period,
      };

      do {
        const res = await listTransportHistory(req, {});
        if (res.transportRecordsList.length === 0) break; // just in case, no infinite loop
        req.pageToken = res.nextPageToken;
        transportFromT.value.push(...res.transportRecordsList);
        if (cancelled()) {
          return;
        }
      } while (req.pageToken && req.pageToken !== '');

    } catch (e) {
      if (!cancelled()) {
        console.warn('Failed to fetch transport history', period, e);
      }
    } finally {
      if (!cancelled()) {
        fetching.value = null;
      }
    }
  }, {immediate: true});

  return transportFromT;
}

/**
 * Convert a proto enum value to a display name. e.g. "DOOR_OPEN" -> "Door Open"
 * replace underscores with spaces, capitalize the first letter of each word
 *
 * @param {string} e
 * @return {string}
 */
function enumToDisplayName(e) {
  // replace underscores in with spaces, change the letters to lower case
  e = e.replace(/_/g, ' ').toLowerCase();
  // then capitalize the first letter eof each word
  e = e.replace(/\b\w/g, l => l.toUpperCase());
  return e;
}

/**
 * @param {MaybeRefOrGetter<Transport.AsObject|null>} value
 * @param {MaybeRefOrGetter<TransportSupport.AsObject|null>=} support
 * @return {{
 *   actualPosition: ComputedRef<string>,
 *   doorStatus: ComputedRef<Array<{label:string, value:string}>>,
 *   movingDirection: ComputedRef<string>,
 *   movingDirectionDisplay: ComputedRef<string>,
 *   nextDestination: ComputedRef<string>,
 *   load: ComputedRef<string>,
 *   table: ComputedRef<Array<{label:string, value:string}>>
 * }}
 */
export function useTransport(value, support = null) {
  const _v = computed(() => toValue(value));
  const _s = computed(() => toValue(support));

  const movingDirectionById = Object.entries(Transport.Direction).reduce((all, [name, id]) => {
    all[id] = name;
    return all;
  }, {});

  const operatingModeById = Object.entries(Transport.OperatingMode).reduce((all, [name, id]) => {
    all[id] = name;
    return all;
  }, {});

  const doorStatusById = Object.entries(Transport.Door.DoorStatus).reduce((all, [name, id]) => {
    all[id] = name;
    return all;
  }, {});

  const actualPosition = computed(() => {
    const v = _v.value;
    if (!v) return '';
    if (v.actualPosition?.floor && v.actualPosition?.floor !== '') {
      return v.actualPosition?.floor;
    }
    if (v.actualPosition?.title && v.actualPosition?.title !== '') {
      return v.actualPosition?.title;
    }
    if (v.actualPosition?.id && v.actualPosition?.id !== '') {
      return v.actualPosition?.id;
    }
    return '';
  });

  const doorStatus = computed(() => {
    const v = _v.value;
    if (!v) return [];
    let res = [];
    for (const door of v.doorsList) {
      res.push({
        label: door.title !== '' ? door.title : 'Door',
        value: enumToDisplayName(doorStatusById[door.status] ?? '')
      });
    }
    return res;
  });

  const movingDirection = computed(() => {
    const v = _v.value;
    if (!v) return 'DIRECTION_UNSPECIFIED';
    return movingDirectionById[v.movingDirection] ?? 'DIRECTION_UNSPECIFIED';
  });

  const movingDirectionDisplay = computed(() => {
    return enumToDisplayName(movingDirection.value);
  });

  const operatingMode = computed(() => {
    const v = _v.value;
    if (!v) return '';
    return enumToDisplayName(operatingModeById[v.operatingMode] ?? '');
  });

  const nextDestination = computed(() => {
    const v = _v.value;
    if (!v) return '';
    if (v.nextDestinationsList.length > 0) {
      return v.nextDestinationsList[0]?.floor ?? 'N/A';
    } else {
      return 'N/A';
    }
  });

  const loadUnit = computed(() => {
    return _s.value?.loadUnit ?? 'kg';
  });

  const load = computed(() => {
    const v = _v.value;
    if (!v) return '';
    return v.load?.toFixed(2);
  });

  const loadStr = computed(() => {
    if (load.value) {
      return `${load.value} ${loadUnit.value}`;
    }
    return '-';
  });

  const table = computed(() => {
    let t = [{
      label: 'Actual Position',
      value: actualPosition.value
    },
      {
        label: 'Moving Direction',
        value: movingDirectionDisplay.value
      },
      {
        label: 'Next Destination',
        value: nextDestination.value
      },
      {
        label: 'Operating Mode',
        value: operatingMode.value
      }
    ];

    for (const door of doorStatus.value) {
      t.push(door);
    }
    t.push({
      label: 'Load',
      value: loadStr.value
    });
    return t;
  });

  return {
    actualPosition,
    doorStatus,
    movingDirection,
    movingDirectionDisplay,
    nextDestination,
    operatingMode,
    loadStr,
    table
  };
}
