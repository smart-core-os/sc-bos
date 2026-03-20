import {closeResource, newResourceValue} from '@/api/resource';
import {pullOpenClosePositions} from '@/api/sc/traits/open-close';
import {toQueryObject, watchResource} from '@/util/traits';
import {computed, onScopeDispose, reactive, toRefs, toValue} from 'vue';

/**
 * @typedef {import('@smart-core-os/sc-bos-ui-gen/proto/smartcore/bos/openclose/v1/open_close_pb').OpenClosePositions} OpenClosePositions
 * @typedef {import('@smart-core-os/sc-bos-ui-gen/proto/smartcore/bos/openclose/v1/open_close_pb').OpenClosePosition} OpenClosePosition
 * @typedef {
 *   import('@smart-core-os/sc-bos-ui-gen/proto/smartcore/bos/openclose/v1/open_close_pb').PullOpenClosePositionsRequest
 * } PullOpenClosePositionsRequest
 * @typedef {
 *   import('@smart-core-os/sc-bos-ui-gen/proto/smartcore/bos/openclose/v1/open_close_pb').PullOpenClosePositionsResponse
 * } PullOpenClosePositionsResponse
 * @typedef {import('vue').Ref} Ref
 * @typedef {import('vue').ToRefs} ToRefs
 * @typedef {import('vue').ComputedRef} ComputedRef
 */

/**
 * @param {MaybeRefOrGetter<string|PullOpenClosePositionsRequest.AsObject>} query
 * @param {MaybeRefOrGetter<boolean>=} paused
 * @return {ToRefs<ResourceValue<OpenClosePositions.AsObject, PullOpenClosePositionsResponse>>}
 */
export function usePullOpenClosePositions(query, paused = false) {
  const openCloseValue = reactive(
      /** @type {ResourceValue<OpenClosePositions.AsObject, PullOpenClosePositionsResponse>} */
      newResourceValue()
  );
  onScopeDispose(() => closeResource(openCloseValue));

  const queryObject = computed(() => toQueryObject(query));

  watchResource(
      () => toValue(queryObject),
      () => toValue(paused),
      (req) => {
        pullOpenClosePositions(req, openCloseValue);
        return () => closeResource(openCloseValue);
      }
  );

  return toRefs(openCloseValue);
}

/**
 * @param {MaybeRefOrGetter<OpenClosePositions.AsObject>} value
 * @return {{
 *   openStr: ComputedRef<string>,
 *   openClass: ComputedRef<string>,
 *   openIcon: ComputedRef<string>,
 *   openPercent: ComputedRef<number|undefined>,
 *   state: ComputedRef<OpenClosePosition.AsObject>}}
 */
export function useOpenClosePositions(value) {
  const _v = computed(() => toValue(value));

  const state = computed(() => _v.value?.statesList[0]);
  const openPercent = computed(() => state.value?.openPercent);
  const openStr = computed(() => {
    const p = openPercent.value;
    if (p === undefined || p === null) return '';
    if (p === 0) {
      return 'Closed';
    } else if (p === 100) {
      return 'Open';
    } else {
      return p + '%';
    }
  });
  const openIcon = computed(() => {
    const p = openPercent.value;
    if (p === 0) {
      return 'mdi-door-closed';
    } else if (p === 100) {
      return 'mdi-door-open';
    } else {
      // also accounts for undefined or null
      return 'mdi-door';
    }
  });
  const openClass = computed(() => {
    const p = openPercent.value;
    if (p === undefined || p === null) return 'unknown';
    if (p === 0) {
      return 'closed';
    } else if (p === 100) {
      return 'open';
    } else {
      return 'moving';
    }
  });

  return {
    state,
    openPercent,
    openStr,
    openIcon,
    openClass
  };
}
