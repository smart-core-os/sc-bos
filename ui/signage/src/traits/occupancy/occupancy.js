import {timestampToDate} from '@/api/convpb.js';
import {closeResource, newResourceValue} from '@/api/resource';
import {occupancyStateToString, pullOccupancy} from '@/api/sc/traits/occupancy';
import {toQueryObject, watchResource} from '@/util/traits';
import {Occupancy} from '@smart-core-os/sc-api-grpc-web/traits/occupancy_sensor_pb';
import {computed, onScopeDispose, reactive, toRefs, toValue} from 'vue';

/**
 * @typedef {
 *   import('@smart-core-os/sc-api-grpc-web/traits/occupancy_sensor_pb').PullOccupancyRequest
 * } PullOccupancyRequest
 * @typedef {
 *   import('@smart-core-os/sc-api-grpc-web/traits/occupancy_sensor_pb').PullOccupancyResponse
 * } PullOccupancyResponse
 * @typedef {import('@smart-core-os/sc-api-grpc-web/traits/occupancy_sensor_pb').Occupancy} Occupancy
 * @typedef {import('vue').ComputedRef} ComputedRef
 * @typedef {import('vue').Ref} Ref
 * @typedef {import('vue').ToRefs} ToRefs
 * @typedef {import('@/api/resource').ResourceValue} ResourceValue
 * @typedef {import('@/api/resource').ResourceError} ResourceError
 */

/**
 * @param {MaybeRefOrGetter<string|PullOccupancyRequest.AsObject>} query - The name of the device or a query object
 * @param {MaybeRefOrGetter<boolean>=} paused - Whether to pause the data stream
 * @return {ToRefs<ResourceValue<Occupancy.AsObject, PullOccupancyResponse>>}
 */
export function usePullOccupancy(query, paused = false) {
  const resource = reactive(
      /** @type {ResourceValue<Occupancy.AsObject, PullOccupancyResponse>} */
      newResourceValue()
  );
  onScopeDispose(() => closeResource(resource));

  const queryObject = computed(() => toQueryObject(query));

  watchResource(
      () => toValue(queryObject),
      () => toValue(paused),
      (req) => {
        pullOccupancy(req, resource);
        return () => closeResource(resource);
      }
  );

  return toRefs(resource);
}

/**
 * @param {MaybeRefOrGetter<Occupancy.AsObject>} value
 * @return {{
 *   stateColor: ComputedRef<string>,
 *   stateStr: ComputedRef<string>,
 *   lastUpdate: ComputedRef<Date>,
 *   icon: ComputedRef<string>,
 *   iconColor: ComputedRef<string>,
 *   peopleCount: ComputedRef<number>,
 *   state: ComputedRef<Occupancy.State>
 * }}
 */
export function useOccupancy(value) {
  const _v = computed(() => toValue(value));

  const state = computed(() => _v.value?.state);
  const stateStr = computed(() => occupancyStateToString(state.value) ?? '');
  const stateColor = computed(() => stateStr.value.toLowerCase());
  const icon = computed(() => {
    if (state.value === Occupancy.State.OCCUPIED) {
      return 'mdi-crosshairs-gps';
    } else if (state.value === Occupancy.State.UNOCCUPIED) {
      return 'mdi-crosshairs';
    } else if (state.value === Occupancy.State.IDLE) {
      return 'mdi-crosshairs-gps';
    } else {
      return '';
    }
  });
  const iconColor = computed(() => {
    if (state.value === Occupancy.State.OCCUPIED) {
      return 'success-lighten-1';
    } else if (state.value === Occupancy.State.UNOCCUPIED) {
      return 'warning';
    } else if (state.value === Occupancy.State.IDLE) {
      return 'info';
    } else {
      return undefined;
    }
  });

  const peopleCount = computed(() => _v.value?.peopleCount ?? 0);

  const lastUpdate = computed(() => timestampToDate(_v.value?.stateChangeTime));

  return {
    state,
    stateStr,
    stateColor,
    icon,
    iconColor,
    peopleCount,
    lastUpdate
  };
}


/**
 *
 * @param {MaybeRefOrGetter<string|PullOccupancyRequest.AsObject>} query - The name of the device or a query object
 * @param {MaybeRefOrGetter<boolean>} paused - Whether to pause the data stream
 * @return {{
 *  occupancyValue: ResourceValue<Occupancy.AsObject, Occupancy>,
 *  occupancyPeopleCount: import('vue').ComputedRef<number>,
 *  occupancyStateNumber: import('vue').ComputedRef<Occupancy.State>,
 *  occupancyStateString: import('vue').ComputedRef<string>,
 *  occupancyLastUpdateDate: import('vue').ComputedRef<Date>,
 *  occupancyColorString: import('vue').ComputedRef<string>,
 *  occupancyIconString: import('vue').ComputedRef<string>,
 *  occupancyIconColor: import('vue').ComputedRef<string|undefined>,
 *  error: import('vue').ComputedRef<ResourceError>,
 *  loading: import('vue').ComputedRef<boolean>
 * }}
 */
export default function(query, paused) {
  const occupancyValue = reactive(
      /** @type {ResourceValue<Occupancy.AsObject, Occupancy>} */
      newResourceValue()
  );

  const queryObject = computed(() => toQueryObject(query));

  watchResource(
      () => toValue(queryObject),
      () => toValue(paused),
      (req) => {
        pullOccupancy(req, occupancyValue);
        return () => closeResource(occupancyValue);
      }
  );


  /** @type {import('vue').ComputedRef<number>} */
  const occupancyPeopleCount = computed(() => {
    return occupancyValue.value?.peopleCount || 0;
  });

  /** @type {import('vue').ComputedRef<number>} */
  const occupancyStateNumber = computed(() => {
    return occupancyValue.value?.state;
  });

  const occupancyStateString = computed(() => {
    return occupancyStateToString(occupancyValue.value?.state) || '';
  });

  /** @type {import('vue').ComputedRef<Date>} */
  const occupancyLastUpdateDate = computed(() => {
    return timestampToDate(occupancyValue.value?.stateChangeTime);
  });

  /** @type {import('vue').ComputedRef<string>} */
  const occupancyColorString = computed(() => {
    return occupancyStateString.value.toLowerCase();
  });

  /** @type {import('vue').ComputedRef<string>} */
  const occupancyIconString = computed(() => {
    if (occupancyStateNumber.value === Occupancy.State.OCCUPIED) {
      return 'mdi-crosshairs-gps';
    } else if (occupancyStateNumber.value === Occupancy.State.UNOCCUPIED) {
      return 'mdi-crosshairs';
    } else if (occupancyStateNumber.value === Occupancy.State.IDLE) {
      return 'mdi-crosshairs-gps';
    } else {
      return '';
    }
  });

  /** @type {import('vue').ComputedRef<string|undefined>} */
  const occupancyIconColor = computed(() => {
    if (occupancyStateNumber.value === Occupancy.State.OCCUPIED) {
      return 'success-lighten-1';
    } else if (occupancyStateNumber.value === Occupancy.State.UNOCCUPIED) {
      return 'warning';
    } else if (occupancyStateNumber.value === Occupancy.State.IDLE) {
      return 'info';
    } else {
      return undefined;
    }
  });

  /** @type {import('vue').ComputedRef<ResourceError>} */
  const error = computed(() => {
    return occupancyValue.streamError;
  });

  /** @type {import('vue').ComputedRef<boolean>} */
  const loading = computed(() => {
    return occupancyValue.loading;
  });


  return {
    occupancyValue,

    occupancyPeopleCount,
    occupancyStateNumber,
    occupancyStateString,
    occupancyLastUpdateDate,

    occupancyColorString,
    occupancyIconString,
    occupancyIconColor,

    error,
    loading
  };
}
