import {closeResource, newResourceValue} from '@/api/resource';
import {pullEmergency} from '@/api/sc/traits/emergency';
import {toQueryObject, watchResource} from '@/util/traits.js';
import {Emergency} from '@smart-core-os/sc-bos-ui-gen/proto/smartcore/bos/emergency/v1/emergency_pb';
import {computed, onScopeDispose, reactive, toRefs, toValue} from 'vue';

/**
 * @typedef {
 *   import('@smart-core-os/sc-bos-ui-gen/proto/smartcore/bos/emergency/v1/emergency_pb').Emergency
 * } Emergency
 * @typedef {
 *   import('@smart-core-os/sc-bos-ui-gen/proto/smartcore/bos/emergency/v1/emergency_pb').PullEmergencyRequest
 * } PullEmergencyRequest
 * @typedef {
 *   import('@smart-core-os/sc-bos-ui-gen/proto/smartcore/bos/emergency/v1/emergency_pb').PullEmergencyResponse
 * } PullEmergencyResponse
 * @typedef {import('vue').ComputedRef} ComputedRef
 * @typedef {import('@/api/resource').ResourceValue} ResourceValue
 * @typedef {import('vue').Ref} Ref
 * @typedef {import('vue').ToRefs} ToRefs
 */
/**
 * @param {MaybeRefOrGetter<string|PullEmergencyRequest.AsObject>} query - The name of the device or a query object
 * @param {MaybeRefOrGetter<boolean>=} paused - Whether to pause the data stream
 * @return {ToRefs<ResourceValue<Emergency.AsObject, PullEmergencyResponse>>}
 */
export function usePullEmergency(query, paused = false) {
  const resource = reactive(
      /** @type {ResourceValue<Emergency.AsObject, PullEmergencyResponse>} */
      newResourceValue()
  );
  onScopeDispose(() => closeResource(resource));

  const queryObject = computed(() => toQueryObject(query));

  watchResource(
      () => toValue(queryObject),
      () => toValue(paused),
      (req) => {
        pullEmergency(req, resource);
        return () => closeResource(resource);
      });

  return toRefs(resource);
}

/**
 * @param {MaybeRefOrGetter<Emergency.AsObject>} value
 * @return {{
 *   level: ComputedRef<Emergency.Level>,
 *   colorClass: ComputedRef<string>,
 *   iconStr: ComputedRef<string>,
 *   tooltipStr: ComputedRef<string>,
 *   drill: ComputedRef<boolean>
 * }}
 */
export function useEmergency(value) {
  const _v = computed(() => toValue(value));

  const level = computed(() => _v.value?.level ?? 0);
  const drill = computed(() => _v.value?.drill ?? false);
  const colorClass = computed(() => {
    switch (level.value) {
      default:
      case Emergency.Level.OK:
        return '';
      case Emergency.Level.WARNING:
        return 'text-warning';
      case Emergency.Level.EMERGENCY:
        if (drill.value) {
          return 'text-info';
        }
        return 'text-error';
    }
  });
  const iconStr = computed(() => {
    switch (level.value) {
      default:
      case Emergency.Level.OK:
        return 'mdi-smoke-detector-outline';
      case Emergency.Level.WARNING:
        return 'mdi-smoke-detector';
      case Emergency.Level.EMERGENCY:
        return 'mdi-smoke-detector-alert';
    }
  });
  const tooltipStr = computed(() => {
    // todo: work out a better message based on current state
    return 'Emergency status';
  });

  return {
    level,
    drill,
    colorClass,
    iconStr,
    tooltipStr
  };
}

/**
 * @param {MaybeRefOrGetter<string|PullEmergencyRequest.AsObject>} query
 * @param {MaybeRefOrGetter<boolean>} paused
 * @return {{
 *  emergencyValue: ResourceValue<Emergency.AsObject, PullEmergencyResponse>,
 *  emergencyColorClass: import('vue').ComputedRef<string>,
 *  emergencyIconString: import('vue').ComputedRef<string>,
 *  emergencyTooltipString: import('vue').ComputedRef<string>,
 *  error: import('vue').ComputedRef<ResourceError>,
 *  loading: import('vue').ComputedRef<boolean>
 * }}
 */
export default function(query, paused) {
  const emergencyValue = reactive(
      /** @type {ResourceValue<Emergency.AsObject, PullEmergencyResponse>} */
      newResourceValue()
  );

  const queryObject = computed(() => toQueryObject(query));

  watchResource(
      () => toValue(queryObject),
      () => toValue(paused),
      (req) => {
        pullEmergency(req, emergencyValue);
        return () => closeResource(emergencyValue);
      });


  /** @type {import('vue').ComputedRef<string>} emergencyColorClass */
  const emergencyColorClass = computed(() => {
    const val = emergencyValue.value?.level;
    const drill = emergencyValue.value?.drill;
    switch (val) {
      default:
      case Emergency.Level.OK:
        return '';
      case Emergency.Level.WARNING:
        return 'text-warning';
      case Emergency.Level.EMERGENCY:
        if (drill) {
          return 'text-info';
        }
        return 'text-error';
    }
  });

  /** @type {import('vue').ComputedRef<string>} emergencyIconString */
  const emergencyIconString = computed(() => {
    const val = emergencyValue.value?.level;
    switch (val) {
      default:
      case Emergency.Level.OK:
        return 'mdi-smoke-detector-outline';
      case Emergency.Level.WARNING:
        return 'mdi-smoke-detector';
      case Emergency.Level.EMERGENCY:
        return 'mdi-smoke-detector-alert';
    }
  });

  /** @type {import('vue').ComputedRef<string>} emergencyTooltipString */
  const emergencyTooltipString = computed(() => {
    // todo: work out a better message based on current state
    return 'Emergency status';
  });


  /** @type {import('vue').ComputedRef<ResourceError>} */
  const error = computed(() => emergencyValue.streamError);


  /** @type {import('vue').ComputedRef<boolean>} */
  const loading = computed(() => emergencyValue.loading);

  return {
    emergencyValue,

    emergencyColorClass,
    emergencyIconString,
    emergencyTooltipString,

    error,
    loading
  };
}
