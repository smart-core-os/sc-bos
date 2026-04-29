import {closeResource, newActionTracker, newResourceValue} from '@/api/resource';
import {describeOpenClosePositions, pullOpenClosePositions, updateOpenClosePositions} from '@/api/sc/traits/open-close';
import {setRequestName, toQueryObject, watchResource} from '@/util/traits';
import {computed, onScopeDispose, reactive, toRefs, toValue} from 'vue';

/**
 * @typedef {import('@smart-core-os/sc-bos-ui-gen/proto/smartcore/bos/openclose/v1/open_close_pb').OpenClosePositions} OpenClosePositions
 * @typedef {import('@smart-core-os/sc-bos-ui-gen/proto/smartcore/bos/openclose/v1/open_close_pb').OpenClosePosition} OpenClosePosition
 * @typedef {import('@smart-core-os/sc-bos-ui-gen/proto/smartcore/bos/openclose/v1/open_close_pb').PositionsSupport} PositionsSupport
 * @typedef {
 *   import('@smart-core-os/sc-bos-ui-gen/proto/smartcore/bos/openclose/v1/open_close_pb').PullOpenClosePositionsRequest
 * } PullOpenClosePositionsRequest
 * @typedef {
 *   import('@smart-core-os/sc-bos-ui-gen/proto/smartcore/bos/openclose/v1/open_close_pb').PullOpenClosePositionsResponse
 * } PullOpenClosePositionsResponse
 * @typedef {
 *   import('@smart-core-os/sc-bos-ui-gen/proto/smartcore/bos/openclose/v1/open_close_pb').UpdateOpenClosePositionsRequest
 * } UpdateOpenClosePositionsRequest
 * @typedef {
 *   import('@smart-core-os/sc-bos-ui-gen/proto/smartcore/bos/openclose/v1/open_close_pb').DescribePositionsRequest
 * } DescribePositionsRequest
 * @typedef {import('vue').Ref} Ref
 * @typedef {import('vue').ToRefs} ToRefs
 * @typedef {import('vue').ComputedRef} ComputedRef
 * @typedef {import('@/api/resource').ResourceValue} ResourceValue
 * @typedef {import('@/api/resource').ActionTracker} ActionTracker
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
 * @param {MaybeRefOrGetter<string>} name
 * @return {{
 *   loading: Ref<boolean>,
 *   response: Ref<OpenClosePositions.AsObject|null>,
 *   error: Ref<*>,
 *   updatePositions: (req: Partial<UpdateOpenClosePositionsRequest.AsObject>|Partial<OpenClosePositions.AsObject>) => Promise<OpenClosePositions.AsObject>
 * }}
 */
export function useUpdateOpenClosePositions(name) {
  const tracker = reactive(
      /** @type {ActionTracker<OpenClosePositions.AsObject>} */
      newActionTracker()
  );

  /**
   * @param {Partial<UpdateOpenClosePositionsRequest.AsObject>|Partial<OpenClosePositions.AsObject>} req
   * @return {UpdateOpenClosePositionsRequest.AsObject}
   */
  const toRequestObject = (req) => {
    req = toValue(req);
    if (!Object.hasOwn(req, 'states')) {
      req = {states: /** @type {OpenClosePositions.AsObject} */ req};
    }
    return setRequestName(req, name);
  };

  return {
    ...toRefs(tracker),
    updatePositions: (req) => {
      return updateOpenClosePositions(toRequestObject(req), tracker);
    }
  };
}

/**
 * @param {MaybeRefOrGetter<string|DescribePositionsRequest.AsObject>} query
 * @return {ToRefs<ActionTracker<PositionsSupport.AsObject>>}
 */
export function useDescribePositions(query) {
  const tracker = reactive(
      /** @type {ActionTracker<PositionsSupport.AsObject>} */
      newActionTracker()
  );

  const queryObject = computed(() => toQueryObject(query));

  watchResource(
      () => toValue(queryObject),
      false,
      (req) => {
        describeOpenClosePositions(req, tracker)
            .catch(() => {}); // errors are tracked by tracker
        return () => closeResource(tracker);
      }
  );

  return toRefs(tracker);
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