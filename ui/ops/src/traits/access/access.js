import {closeResource, newResourceValue} from '@/api/resource';
import {pullAccessAttempts} from '@/api/sc/traits/access';
import {toQueryObject, watchResource} from '@/util/traits';
import {AccessAttempt} from '@smart-core-os/sc-bos-ui-gen/proto/smartcore/bos/access/v1/access_pb';
import {computed, onScopeDispose, reactive, toRefs, toValue} from 'vue';

/**
 * @typedef {import('@smart-core-os/sc-bos-ui-gen/proto/smartcore/bos/access/v1/access_pb').PullAccessAttemptsRequest} PullAccessAttemptsRequest
 * @typedef {import('@smart-core-os/sc-bos-ui-gen/proto/smartcore/bos/access/v1/access_pb').PullAccessAttemptsResponse} PullAccessAttemptsResponse
 * @typedef {import('@smart-core-os/sc-bos-ui-gen/proto/smartcore/bos/access/v1/access_pb').AccessAttempt} AccessAttempt
 * @typedef {import('vue').UnwrapNestedRefs} UnwrapNestedRefs
 * @typedef {import('vue').ToRefs} ToRefs
 * @typedef {import('vue').ComputedRef} ComputedRef
 * @typedef {import('vue').MaybeRefOrGetter} MaybeRefOrGetter
 * @typedef {import('@/api/resource').ResourceValue} ResourceValue
 */

/**
 * @param {MaybeRefOrGetter<string|PullAccessAttemptsRequest.AsObject>} query
 * @param {MaybeRefOrGetter<boolean>=} paused
 * @return {ToRefs<ResourceValue<AccessAttempt.AsObject, PullAccessAttemptsResponse>>}
 */
export function usePullAccessAttempts(query, paused = false) {
  const resource = reactive(
      /** @type {ResourceValue<AccessAttempt.AsObject, PullAccessAttemptsResponse>} */
      newResourceValue()
  );
  onScopeDispose(() => closeResource(resource));

  const queryObject = computed(() => toQueryObject(query));

  watchResource(
      () => toValue(queryObject),
      () => toValue(paused),
      (req) => {
        pullAccessAttempts(req, resource);
        return () => closeResource(resource);
      }
  );

  return toRefs(resource);
}

/**
 * @param {MaybeRefOrGetter<AccessAttempt.AsObject>} value
 * @return {{
 *   accessAttemptInfo: ComputedRef<[{},{}]>,
 *   grantId: ComputedRef<string>,
 *   grantClass: ComputedRef<string>,
 *   grantState: ComputedRef<string>,
 *   grantNamesByID: {}
 * }}
 */
export function useAccessAttempt(value) {
  const grantId = computed(() => toValue(value)?.grant);
  const grantNamesByID = Object.entries(AccessAttempt.Grant).reduce((all, [name, id]) => {
    all[id] = name;
    return all;
  }, {});
  const grantState = computed(() => grantNamesByID[grantId.value || 0]);
  const grantClass = computed(() => grantState.value.toLowerCase());
  const accessAttemptInfo = computed(() => {
    // Initialize variables for info and subInfo
    const info = {};
    const subInfo = {};

    // Check if value has metadata property
    const v = toValue(value);
    if (v) {
      // Get all properties of metadata as an array of [key, value] pairs
      const data = Object.entries(v);

      // Flatten out the data
      data.forEach(([key, value]) => {
        if (value) {
          if (typeof value !== 'object') {
            if (key === 'grant') {
              const state = grantNamesByID[value].split('_').join(' ');
              info[key] = state.charAt(0).toUpperCase() + state.slice(1);
            } else info[key] = value;
          } else {
            for (const subValue in value) {
              if (subValue && value[subValue]) {
                if (!subInfo[key]) {
                  subInfo[key] = {};
                }

                // If subValue is an array, map it to an object
                if (value[subValue].length) {
                  if (Array.isArray(value[subValue])) {
                    subInfo[key][subValue] = value[subValue].map(([key, value]) => {
                      return {[key]: value};
                    });
                  } else {
                    subInfo[key][subValue] = value[subValue];
                  }
                }
              }
            }
          }
        }
      });
    }

    // Return info and subInfo objects within an array
    return [info, subInfo];
  });
  return {
    grantId,
    grantNamesByID,
    grantState,
    grantClass,
    accessAttemptInfo
  };
}
