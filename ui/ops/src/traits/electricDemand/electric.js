import {closeResource, newResourceValue} from '@/api/resource';
import {pullDemand} from '@/api/sc/traits/electric';
import {toQueryObject, watchResource} from '@/util/traits.js';
import {computed, onScopeDispose, reactive, toRefs, toValue} from 'vue';

/**
 * @typedef {
 *   import('@smart-core-os/sc-bos-ui-gen/proto/smartcore/bos/electric/v1/electric_pb').ElectricDemand
 * } ElectricDemand
 * @typedef {
 *   import('@smart-core-os/sc-bos-ui-gen/proto/smartcore/bos/electric/v1/electric_pb').PullDemandRequest
 * } PullDemandRequest
 * @typedef {
 *   import('@smart-core-os/sc-bos-ui-gen/proto/smartcore/bos/electric/v1/electric_pb').PullDemandResponse
 * } PullDemandResponse
 * @typedef {import('vue').ComputedRef} ComputedRef
 * @typedef {import('@/api/resource').ResourceValue} ResourceValue
 * @typedef {import('vue').Ref} Ref
 * @typedef {import('vue').ToRefs} ToRefs
 */

/**
 * @param {MaybeRefOrGetter<string|PullDemandRequest.AsObject>} query - The name of the device or a query object
 * @param {MaybeRefOrGetter<boolean>=} paused - Whether to pause the data stream
 * @return {ToRefs<ResourceValue<ElectricDemand.AsObject, PullDemandResponse>>}
 */
export function usePullElectricDemand(query, paused = false) {
  const resource = reactive(
      /** @type {ResourceValue<ElectricDemand.AsObject, PullDemandResponse>} */
      newResourceValue()
  );
  onScopeDispose(() => closeResource(resource));

  const queryObject = computed(() => toQueryObject(query));

  watchResource(
      () => toValue(queryObject),
      () => toValue(paused),
      (req) => {
        pullDemand(req, resource);
        return () => closeResource(resource);
      }
  );

  return toRefs(resource);
}

/**
 * @param {MaybeRefOrGetter<ElectricDemand.AsObject>} value
 * @return {{
 *   realPower: ComputedRef<number>,
 *   realPowerUnit: ComputedRef<string>,
 *   apparentPower: ComputedRef<number>,
 *   apparentPowerUnit: ComputedRef<string>,
 *   reactivePower: ComputedRef<number>,
 *   reactivePowerUnit: ComputedRef<string>,
 *   powerFactor: ComputedRef<number>
 * }}
 */
export function useElectricDemand(value) {
  const _v = computed(
      () => /** @type {ElectricDemand.AsObject} */ toValue(value));

  // return (v / d) or 0 if v is undefined
  const div = (v, d) => v ? v / d : 0;

  const realPower = computed(() => div(_v.value?.realPower, 1000));
  const realPowerUnit = computed(() => 'kW');
  const apparentPower = computed(() => div(_v.value?.apparentPower, 1000));
  const apparentPowerUnit = computed(() => 'kVA');
  const reactivePower = computed(() => div(_v.value?.reactivePower, 1000));
  const reactivePowerUnit = computed(() => 'kVAr');
  const powerFactor = computed(() => _v.value?.powerFactor ?? 0);
  const current = computed(() => _v.value?.current);
  const currentUnit = computed(() => 'A');

  const hasApparentPower = computed(() => apparentPower.value !== 0);
  const hasReactivePower = computed(() => reactivePower.value !== 0);
  const hasPowerFactor = computed(() => powerFactor.value !== 0);
  const hasCurrent = computed(() => Boolean(current.value));

  return {
    realPower, realPowerUnit,
    apparentPower, apparentPowerUnit, hasApparentPower,
    reactivePower, reactivePowerUnit, hasReactivePower,
    powerFactor, hasPowerFactor,
    current, currentUnit, hasCurrent
  };
}
