import {closeResource, newActionTracker, newResourceValue} from '@/api/resource';
import {pullTemperature, temperatureToString, updateTemperature} from '@/api/sc/traits/temperature';
import {setRequestName, toQueryObject, watchResource} from '@/util/traits';
import {isNullOrUndef} from '@/util/types.js';
import {computed, onScopeDispose, reactive, ref, toRefs, toValue} from 'vue';

/**
 * @typedef {
 *   import('@smart-core-os/sc-bos-ui-gen/proto/temperature_pb').PullTemperatureRequest
 * } PullTemperatureRequest
 * @typedef {
 *   import('@smart-core-os/sc-bos-ui-gen/proto/temperature_pb').PullTemperatureResponse
 * } PullTemperatureResponse
 * @typedef {
 *   import('@smart-core-os/sc-bos-ui-gen/proto/temperature_pb').UpdateTemperatureRequest
 * } UpdateTemperatureRequest
 * @typedef {
 *  import('@smart-core-os/sc-bos-ui-gen/proto/temperature_pb').Temperature
 * } Temperature
 * @typedef {import('vue').Ref} Ref
 * @typedef {import('vue').UnwrapNestedRefs} UnwrapNestedRefs
 * @typedef {import('vue').ToRefs} ToRefs
 * @typedef {import('vue').ComputedRef} ComputedRef
 * @typedef {import('@/api/resource').ResourceValue} ResourceValue
 * @typedef {import('@/api/resource').ActionTracker} ActionTracker
 */

/**
 * @param {MaybeRefOrGetter<string|PullTemperatureRequest.AsObject>} query
 * @param {MaybeRefOrGetter<boolean>=} paused
 * @return {ToRefs<UnwrapNestedRefs<ResourceValue<Temperature.AsObject, PullTemperatureResponse>>>}
 */
export function usePullTemperature(query, paused = false) {
  const resource = reactive(
      /** @type {ResourceValue<Temperature.AsObject, PullTemperatureResponse>} */
      newResourceValue()
  );
  onScopeDispose(() => closeResource(resource));

  const queryObject = computed(() => toQueryObject(query));

  watchResource(
      () => toValue(queryObject),
      () => toValue(paused),
      (req) => {
        pullTemperature(req, resource);
        return () => closeResource(resource);
      }
  );

  return toRefs(resource);
}

/**
 * @typedef UpdateTemperatureRequestLike
 * @type {number|Partial<Temperature.AsObject>|Partial<UpdateTemperatureRequest.AsObject>}
 */
/**
 * @param {MaybeRefOrGetter<string>=} name The name of the device to update.
 *   If not provided request objects must include a name.
 * @return {ToRefs<ActionTracker<Temperature.AsObject>> & {
 *   updateTemperature: (req: MaybeRefOrGetter<UpdateTemperatureRequestLike>) => Promise<Temperature.AsObject>
 * }}
 */
export function useUpdateTemperature(name) {
  const tracker = reactive(
      /** @type {ActionTracker<Temperature.AsObject>} */
      newActionTracker()
  );

  /**
   * @param {MaybeRefOrGetter<UpdateTemperatureRequestLike>} req
   * @return {UpdateTemperatureRequest.AsObject}
   */
  const toRequestObject = (req) => {
    req = toValue(req);
    if (typeof req === 'number') {
      req = {
        temperature: {setPoint: {valueCelsius: /** @type {number} */ req}},
        updateMask: {pathsList: ['set_point']}
      };
    }
    if (!Object.hasOwn(req, 'temperature')) {
      req = {temperature: /** @type {Temperature.AsObject} */ req};
    }
    return setRequestName(req, name);
  };

  return {
    ...toRefs(tracker),
    updateTemperature: (req) => {
      return updateTemperature(toRequestObject(req), tracker);
    }
  };
}

const UNIT = '°C';

/**
 * @param {MaybeRefOrGetter<Temperature.AsObject>} value
 * @return {UseTemperatureValuesReturn & {
 *   measured: ComputedRef<number>,
 *   setPoint: ComputedRef<number>,
 *   temperatureData: ComputedRef<{}>,
 *   tempRange: Ref<{low: number, high: number}>,
 *   tempProgress: ComputedRef<number>
 * }}
 */
export function useTemperature(value) {
  const _v = computed(() => toValue(value));

  const setPoint = computed(() => {
    return _v.value?.setPoint?.valueCelsius;
  });
  const measured = computed(() => {
    return _v.value?.measured?.valueCelsius;
  });
  const {
    hasSetPoint,
    setPointStr,
    hasMeasured,
    measuredStr
  } = useTemperatureValues(measured, setPoint);

  const tempRange = ref({
    low: 18.0,
    high: 30.0
  });
  const tempProgress = computed(() => {
    let val = measured.value ?? 0;
    if (val > 0) {
      val -= tempRange.value.low;
      val = val / (tempRange.value.high - tempRange.value.low);
    }
    return val * 100;
  });
  const temperatureData = computed(() => {
    if (_v.value) {
      const data = {};
      Object.entries(_v.value).forEach(([key, value]) => {
        if (value !== undefined) {
          switch (key) {
            case 'measured': {
              data['measured'] = temperatureToString(value);
              break;
            }
            case 'setPoint': {
              data['setPoint'] = temperatureToString(value);
              break;
            }
            default: {
              data[key] = value;
            }
          }
        }
      });
      return data;
    }
    return {};
  });

  return {
    hasSetPoint,
    setPoint,
    setPointStr,
    hasMeasured,
    measured,
    measuredStr,
    tempRange,
    tempProgress,
    temperatureData
  };
}

/**
 * @typedef {Object} UseTemperatureValuesReturn
 * @property {Readonly<Ref<boolean>>} hasMeasured
 * @property {Readonly<Ref<boolean>>} hasSetPoint
 * @property {Readonly<Ref<string>>} setPointStr
 * @property {Readonly<Ref<string>>} measuredStr
 */
/**
 * @param {MaybeRefOrGetter<number | undefined>} measured
 * @param {MaybeRefOrGetter<number | undefined>} setPoint
 * @return {UseTemperatureValuesReturn}
 */
export function useTemperatureValues(measured, setPoint) {
  const has = (v) => !isNullOrUndef(v) && !isNaN(v);
  const hasSetPoint = computed(() => {
    return has(toValue(setPoint));
  });
  const setPointStr = computed(() => {
    return (toValue(setPoint) ?? 0).toFixed(1) + UNIT;
  });

  const hasMeasured = computed(() => {
    return has(toValue(measured));
  });
  const measuredStr = computed(() => {
    const numStr = (toValue(measured) ?? 0).toFixed(1);
    if (hasSetPoint.value) {
      return numStr;
    } else {
      return numStr + UNIT;
    }
  });

  return {
    hasSetPoint,
    setPointStr,
    hasMeasured,
    measuredStr
  };
}

