import {closeResource, newResourceValue} from '@/api/resource';
import {pullEnergyLevel} from '@/api/sc/traits/energy-storage';
import {toQueryObject, watchResource} from '@/util/traits.js';
import {computed, onScopeDispose, reactive, toRefs, toValue} from 'vue';

/**
 * @typedef {
 *   import('@smart-core-os/sc-bos-ui-gen/proto/smartcore/bos/energystorage/v1/energy_storage_pb').EnergyLevel
 * } EnergyLevel
 * @typedef {
 *   import('@smart-core-os/sc-bos-ui-gen/proto/smartcore/bos/energystorage/v1/energy_storage_pb').PullEnergyLevelRequest
 * } PullEnergyLevelRequest
 * @typedef {
 *   import('@smart-core-os/sc-bos-ui-gen/proto/smartcore/bos/energystorage/v1/energy_storage_pb').PullEnergyLevelResponse
 * } PullEnergyLevelResponse
 * @typedef {import('vue').ComputedRef} ComputedRef
 * @typedef {import('@/api/resource').ResourceValue} ResourceValue
 * @typedef {import('vue').Ref} Ref
 * @typedef {import('vue').ToRefs} ToRefs
 */

/**
 * @param {MaybeRefOrGetter<string|PullEnergyLevelRequest.AsObject>} query - The name of the device or a query object
 * @param {MaybeRefOrGetter<boolean>=} paused - Whether to pause the data stream
 * @return {ToRefs<ResourceValue<EnergyLevel.AsObject, PullEnergyLevelResponse>>}
 */
export function usePullEnergyLevel(query, paused = false) {
  const resource = reactive(
      /** @type {ResourceValue<EnergyLevel.AsObject, PullEnergyLevelResponse>} */
      newResourceValue()
  );
  onScopeDispose(() => closeResource(resource));

  const queryObject = computed(() => toQueryObject(query));

  watchResource(
      () => toValue(queryObject),
      () => toValue(paused),
      (req) => {
        pullEnergyLevel(req, resource);
        return () => closeResource(resource);
      }
  );

  return toRefs(resource);
}

/**
 * @param {MaybeRefOrGetter<EnergyLevel.AsObject>} value
 * @return {{
 *   percentage: ComputedRef<number>,
 *   energyKwh: ComputedRef<number>,
 *   distanceKm: ComputedRef<number>,
 *   voltage: ComputedRef<number>,
 *   hasVoltage: ComputedRef<boolean>,
 *   descriptive: ComputedRef<string>,
 *   isCharging: ComputedRef<boolean>,
 *   isDischarging: ComputedRef<boolean>,
 *   isIdle: ComputedRef<boolean>,
 *   pluggedIn: ComputedRef<boolean>,
 *   flowStatus: ComputedRef<string>,
 *   speed: ComputedRef<string>,
 *   targetPercentage: ComputedRef<number>,
 *   hasTargetPercentage: ComputedRef<boolean>
 * }}
 */
export function useEnergyStorage(value) {
  const _v = computed(
      () => /** @type {EnergyLevel.AsObject} */ toValue(value));

  const quantity = computed(() => _v.value?.quantity);

  const percentage = computed(() => quantity.value?.percentage ?? 0);
  const energyKwh = computed(() => quantity.value?.energyKwh ?? 0);
  const distanceKm = computed(() => quantity.value?.distanceKm ?? 0);
  const voltage = computed(() => quantity.value?.voltage ?? 0);
  const hasVoltage = computed(() => voltage.value > 0);

  const descriptive = computed(() => {
    const desc = quantity.value?.descriptive;
    switch (desc) {
      case 1: return 'Critically Low';
      case 2: return 'Empty';
      case 3: return 'Low';
      case 4: return 'Medium';
      case 5: return 'High';
      case 7: return 'Full';
      case 8: return 'Critically High';
      default: return '';
    }
  });

  const isCharging = computed(() => !!_v.value?.charge);
  const isDischarging = computed(() => !!_v.value?.discharge);
  const isIdle = computed(() => !!_v.value?.idle);
  const pluggedIn = computed(() => _v.value?.pluggedIn ?? false);

  const flowStatus = computed(() => {
    if (isCharging.value) return 'Charging';
    if (isDischarging.value) return 'Discharging';
    if (isIdle.value) return 'Idle';
    return 'Unknown';
  });

  const speed = computed(() => {
    const transfer = _v.value?.charge || _v.value?.discharge;
    const speedEnum = transfer?.speed;
    switch (speedEnum) {
      case 1: return 'Extra Slow';
      case 2: return 'Slow';
      case 3: return 'Normal';
      case 4: return 'Fast';
      case 5: return 'Extra Fast';
      default: return '';
    }
  });

  const targetPercentage = computed(() => {
    const transfer = _v.value?.charge || _v.value?.discharge;
    return transfer?.target?.percentage ?? 0;
  });

  const hasTargetPercentage = computed(() => targetPercentage.value > 0);

  return {
    percentage, energyKwh, distanceKm, voltage, hasVoltage, descriptive,
    isCharging, isDischarging, isIdle, pluggedIn,
    flowStatus, speed, targetPercentage, hasTargetPercentage
  };
}
