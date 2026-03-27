import {closeResource, newResourceValue} from '@/api/resource';
import {pullEnterLeaveEvents} from '@/api/sc/traits/enter-leave';
import {SECOND} from '@/components/now.js';
import {toQueryObject, watchResource} from '@/util/traits';
import {EnterLeaveEvent} from '@smart-core-os/sc-api-grpc-web/traits/enter_leave_sensor_pb';
import {computed, onScopeDispose, reactive, ref, toRefs, toValue, watch} from 'vue';

/**
 * @typedef {
 *   import('@smart-core-os/sc-api-grpc-web/traits/enter_leave_sensor_pb').PullEnterLeaveEventsRequest
 * } PullEnterLeaveEventsRequest
 * @typedef {
 *   import('@smart-core-os/sc-api-grpc-web/traits/enter_leave_sensor_pb').PullEnterLeaveEventsResponse
 * } PullEnterLeaveEventsResponse
 * @typedef {
 *   import('@smart-core-os/sc-api-grpc-web/traits/enter_leave_sensor_pb').EnterLeaveEvent
 * } EnterLeaveEvent
 * @typedef {import('vue').ComputedRef} ComputedRef
 * @typedef {import('@/api/resource').ResourceError} ResourceError
 * @typedef {import('@/api/resource').ResourceValue} ResourceValue
 * @typedef {import('vue').Ref} Ref
 * @typedef {import('vue').ToRefs} ToRefs
 */

/**
 * @param {MaybeRefOrGetter<string|PullEnterLeaveEventsRequest.AsObject>} query - The name of the device or a query
 * @param {MaybeRefOrGetter<boolean>=} paused - Whether to pause the data stream
 * @return {ToRefs<ResourceValue<EnterLeaveEvent.AsObject, PullEnterLeaveEventsResponse>>}
 */
export function usePullEnterLeaveEvents(query, paused = false) {
  const resource = reactive(
      /** @type {ResourceValue<EnterLeaveEvent.AsObject, PullEnterLeaveEventsResponse>} */
      newResourceValue()
  );
  onScopeDispose(() => closeResource(resource));

  const queryObject = computed(() => toQueryObject(query));

  watchResource(
      () => toValue(queryObject),
      () => toValue(paused),
      (req) => {
        pullEnterLeaveEvents(req, resource);
        return () => closeResource(resource);
      }
  );

  return toRefs(resource);
}

/**
 * @param {MaybeRefOrGetter<EnterLeaveEvent.AsObject>} value
 * @param {{
 *   showChangeDuration?: MaybeRefOrGetter<number>
 * }=} opts
 * @return {{
 *   hasTotals: ComputedRef<boolean>,
 *   enterTotal: ComputedRef<number>,
 *   leaveTotal: ComputedRef<number>,
 *   justEntered: Ref<boolean>,
 *   justLeft: Ref<boolean>
 * }}
 */
export function useEnterLeaveEvent(value, {showChangeDuration = 30 * SECOND} = {}) {
  const _v = computed(() => toValue(value));

  const hasTotals = computed(() => _v.value !== undefined && _v.value !== null);
  const enterTotal = computed(() => _v.value?.enterTotal || 0);
  const leaveTotal = computed(() => _v.value?.leaveTotal || 0);

  const enterTimeoutHandle = ref(0);
  const leaveTimeoutHandle = ref(0);
  const justEntered = ref(false);
  const justLeft = ref(false);
  watch(_v, (newVal, oldVal) => {
    if (!oldVal || !newVal) {
      justEntered.value = false;
      justLeft.value = false;
      clearTimeout(enterTimeoutHandle.value);
      clearTimeout(leaveTimeoutHandle.value);
    }

    if (newVal?.direction === EnterLeaveEvent.Direction.ENTER) {
      justEntered.value = true;
      clearTimeout(enterTimeoutHandle.value);
      enterTimeoutHandle.value = setTimeout(() => {
        justEntered.value = false;
      }, toValue(showChangeDuration));
    }
    if (newVal?.direction === EnterLeaveEvent.Direction.LEAVE) {
      justLeft.value = true;
      clearTimeout(leaveTimeoutHandle.value);
      leaveTimeoutHandle.value = setTimeout(() => {
        justLeft.value = false;
      }, toValue(showChangeDuration));
    }
  }, {deep: true});
  onScopeDispose(() => {
    clearTimeout(enterTimeoutHandle.value);
    clearTimeout(leaveTimeoutHandle.value);
  });

  return {
    hasTotals,
    enterTotal,
    leaveTotal,
    justEntered,
    justLeft
  };
}
