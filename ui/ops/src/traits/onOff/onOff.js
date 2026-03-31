import {closeResource, newActionTracker, newResourceValue} from '@/api/resource';
import {pullOnOff, updateOnOff} from '@/api/sc/traits/on-off';
import {setRequestName, toQueryObject, watchResource} from '@/util/traits';
import {computed, onScopeDispose, reactive, toRefs, toValue} from 'vue';
import {OnOff} from '@smart-core-os/sc-bos-ui-gen/proto/smartcore/bos/onoff/v1/on_off_pb';

/**
 * @typedef {
 *   import('@smart-core-os/sc-bos-ui-gen/proto/smartcore/bos/onoff/v1/on_off_pb').PullOnOffRequest
 * } PullOnOffRequest
 * @typedef {
 *   import('@smart-core-os/sc-bos-ui-gen/proto/smartcore/bos/onoff/v1/on_off_pb').PullOnOffResponse
 * } PullOnOffResponse
 * @typedef {
 *   import('@smart-core-os/sc-bos-ui-gen/proto/smartcore/bos/onoff/v1/on_off_pb').UpdateOnOffRequest
 * } UpdateOnOffRequest
 * @typedef {
 *  import('@smart-core-os/sc-bos-ui-gen/proto/smartcore/bos/onoff/v1/on_off_pb').OnOff
 * } OnOff
 * @typedef {import('vue').Ref} Ref
 * @typedef {import('vue').UnwrapNestedRefs} UnwrapNestedRefs
 * @typedef {import('vue').ToRefs} ToRefs
 * @typedef {import('vue').ComputedRef} ComputedRef
 * @typedef {import('@/api/resource').ResourceValue} ResourceValue
 * @typedef {import('@/api/resource').ActionTracker} ActionTracker
 */

/**
 * @param {MaybeRefOrGetter<string|PullOnOffRequest.AsObject>} query
 * @param {MaybeRefOrGetter<boolean>=} paused
 * @return {ToRefs<UnwrapNestedRefs<ResourceValue<OnOff.AsObject, PullOnOffResponse>>>}
 */
export function usePullOnOff(query, paused = false) {
 const resource = reactive(
   /** @type {ResourceValue<OnOff.AsObject, PullOnOffResponse>} */
   newResourceValue()
 );
 onScopeDispose(() => closeResource(resource));
 
 const queryObject = computed(() => toQueryObject(query));
 
 watchResource(
   () => toValue(queryObject),
   () => toValue(paused),
   (req) => {
    pullOnOff(req, resource);
    return () => closeResource(resource);
   }
 );
 
 return toRefs(resource);
}

/**
 * @typedef UpdateOnOffRequest
 * @type {Partial<OnOff.AsObject>|Partial<UpdateOnOffRequest.AsObject>}
 */
export function useUpdateOnOff(name) {
 const tracker = reactive(
   /** @type {ActionTracker<AirTemperature.AsObject>} */
   newActionTracker()
 );
 
 /**
  * @param {MaybeRefOrGetter<UpdateOnOffRequest>} req
  * @return {UpdateOnOffRequest.AsObject}
  */
 const toRequestObject = (req) => {
  req = toValue(req);
  if (!Object.hasOwn(req, 'onOff')) {
   req = {onOff: /** @type {OnOff.AsObject} */ req};
  }
  return setRequestName(req, name);
 };
 
 return {
  ...toRefs(tracker),
   updateOnOff: (req) => {
   return updateOnOff(toRequestObject(req), tracker);
  }
 };
}

/**
 * Convert OnOff state to a human-readable string.
 *
 * @param {OnOff.State} onOff
 * @return {string}
 */
export function onOffToString(onOff) {
  switch (onOff) {
   case OnOff.State.OFF:
      return 'Off';
    case OnOff.State.ON:
      return 'On';
   default:
      return 'Unknown';
  }
}