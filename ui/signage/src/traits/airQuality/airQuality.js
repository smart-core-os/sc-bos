import {closeResource, newResourceValue} from '@/api/resource.js';
import {pullAirQualitySensor} from '@/api/sc/traits/air-quality-sensor';
import {toQueryObject, watchResource} from '@/util/traits.js';
import {isNullOrUndef} from '@/util/types.js';
import {AirQuality} from '@smart-core-os/sc-api-grpc-web/traits/air_quality_sensor_pb';
import {computed, onScopeDispose, reactive, toRefs, toValue} from 'vue';

/**
 * @typedef {
 *   import('@smart-core-os/sc-api-grpc-web/traits/air_quality_sensor_pb').PullAirQualityRequest
 * } PullAirQualityRequest
 * @typedef {
 *   import('@smart-core-os/sc-api-grpc-web/traits/air_quality_sensor_pb').PullAirQualityResponse
 * } PullAirQualityResponse
 * @typedef {import('vue').ComputedRef} ComputedRef
 * @typedef {import('@/api/resource').ResourceError} ResourceError
 * @typedef {import('@/api/resource').ResourceValue} ResourceValue
 * @typedef {import('vue').Ref} Ref
 * @typedef {import('vue').ToRefs} ToRefs
 * @typedef {import('vue').MaybeRefOrGetter} MaybeRefOrGetter
 */

/**
 * @param {MaybeRefOrGetter<string|PullAirQualityRequest.AsObject>} query - The name of the device or a query object
 * @param {MaybeRefOrGetter<boolean>=} paused - Whether to pause the data stream
 * @return {ToRefs<ResourceValue<AirQuality.AsObject, PullAirQualityResponse>>}
 */
export function usePullAirQuality(query, paused = false) {
  const resource = reactive(
      /** @type {ResourceValue<AirQuality.AsObject, PullAirQualityResponse>} */
      newResourceValue()
  );
  onScopeDispose(() => closeResource(resource));

  const queryObject = computed(() => toQueryObject(query));

  watchResource(
      () => toValue(queryObject),
      () => toValue(paused),
      (req) => {
        pullAirQualitySensor(req, resource);
        return () => closeResource(resource);
      }
  );

  return toRefs(resource);
}

export const status = {
  ERROR: 'error',
  WARNING: 'warning',
  SUCCESS: 'success'
};

/**
 * @param {valueOf<status>} s
 * @return {string}
 */
export function statusToColor(s) {
  switch (s) {
    case status.ERROR:
      return 'error-lighten-1';
    case status.WARNING:
      return 'warning';
    case status.SUCCESS:
      return 'success-lighten-1';
    default:
      return undefined;
  }
}

/**
 * @param {import('vue').MaybeRefOrGetter<valueOf<status> | {status:valueOf<status>}>} item
 * @return {import('vue').ComputedRef<string>}
 */
export function useStatusColor(item) {
  return computed(() => {
    const v = toValue(item);
    const s = v?.status ?? v;
    return statusToColor(s);
  });
}

/**
 * @typedef {Object} AirQualityMetricDesc
 * @property {string} label
 * @property {string} unit
 * @property {number?} min
 * @property {number?} max
 * @property {Array<{value: number, status: valueOf<status>}>} levels
 */
/**
 * @type {Record<string, AirQualityMetricDesc>}
 */
export const metrics = {
  'score': {
    label: 'IAQ',
    labelText: 'IAQ',
    unit: '%',
    min: 0,
    max: 100,
    levels: [
      {value: 0, status: status.ERROR},
      {value: 10, status: status.WARNING},
      {value: 50, status: status.SUCCESS}
    ]
  },
  'carbonDioxideLevel': {
    label: 'CO<sub>2</sub>',
    labelText: 'CO₂',
    unit: 'ppm',
    min: 0,
    max: 5000,
    levels: [
      {value: 0, status: status.SUCCESS},
      {value: 1000, status: status.WARNING},
      {value: 2000, status: status.ERROR}
    ]
  },
  'volatileOrganicCompounds': {
    label: 'VOC',
    labelText: 'VOC',
    unit: 'ppm',
    min: 0,
    max: 1,
    levels: [
      {value: 0, status: status.SUCCESS},
      {value: 0.3, status: status.WARNING},
      {value: 0.5, status: status.ERROR}
    ]
  },
  'airPressure': {
    label: 'Air Pressure',
    labelText: 'Air Pressure',
    unit: 'hPa',
    min: 0,
    max: 1100,
    levels: [
      {value: 0, status: status.ERROR},
      {value: 1000, status: status.SUCCESS}
    ]
  },
  'infectionRisk': {
    label: 'Infection Risk',
    labelText: 'Infection Risk',
    unit: '%',
    min: 0,
    max: 100,
    levels: [
      {value: 0, status: status.SUCCESS},
      {value: 25, status: status.WARNING},
      {value: 50, status: status.ERROR}
    ]
  },
  'particulateMatter1': {
    label: 'PM1',
    labelText: 'PM1',
    unit: 'µg/m³',
    min: 0,
    max: 50,
    levels: [
      {value: 0, status: status.SUCCESS},
      {value: 10, status: status.WARNING},
      {value: 20, status: status.ERROR}
    ]
  },
  'particulateMatter25': {
    label: 'PM2.5',
    labelText: 'PM2.5',
    unit: 'µg/m³',
    min: 0,
    max: 50,
    levels: [
      {value: 0, status: status.SUCCESS},
      {value: 10, status: status.WARNING},
      {value: 20, status: status.ERROR}
    ]
  },
  'particulateMatter10': {
    label: 'PM10',
    labelText: 'PM10',
    unit: 'µg/m³',
    min: 0,
    max: 50,
    levels: [
      {value: 0, status: status.SUCCESS},
      {value: 10, status: status.WARNING},
      {value: 20, status: status.ERROR}
    ]
  },
  'airChangePerHour': {
    label: 'Air Exchange Rate',
    labelText: 'Air Exchange Rate',
    unit: '/h',
    min: 0,
    max: 10,
    levels: [
      {value: 0, status: status.ERROR},
      {value: 5, status: status.SUCCESS}
    ]
  },
  'comfort': {
    label: 'Comfort',
    labelText: 'Comfort',
    unit: '',
    levels: [
      {value: AirQuality.Comfort.COMFORTABLE, status: status.SUCCESS},
      {value: AirQuality.Comfort.UNCOMFORTABLE, status: status.ERROR}
    ]
  }
};

/**
 * @typedef {Object} AirQualityMetric
 * @property {number} value
 * @property {valueOf<status>} status
 */

/**
 * @typedef {Object} AirQualityScore
 * @property {number} value
 * @property {string} label
 * @property {valueOf<status>} status
 */

/**
 * @param {MaybeRefOrGetter<AirQuality.AsObject>} value
 * @return {{
 *   presentMetrics: import('vue').ComputedRef<Record<keyof metrics, AirQualityMetric>>,
 *   score: import('vue').ComputedRef<AirQualityScore>
 * }}
 */
export function useAirQuality(value) {
  const _v = computed(() => toValue(value));

  const presentMetrics = computed(() => {
    const result = /** @type {Record<keyof metrics, AirQualityMetric>} */ {};
    for (const [k, v] of Object.entries(_v.value ?? {})) {
      const m = metrics[k];
      if (!m || !v) {
        continue; // skip unknown metrics
      }
      let status = '';
      for (const level of m.levels) {
        if (v >= level.value) {
          status = level.status;
          continue;
        }
        break;
      }
      result[k] = {value: v, status};
    }
    return result;
  });

  const score = computed(() => {
    const presentScore = presentMetrics.value['score'];
    const statusToLabel = (status) => {
      switch (status) {
        case status.ERROR:
          return 'Poor';
        case status.WARNING:
          return 'Fair';
        case status.SUCCESS:
          return 'Good';
        default:
          return '';
      }
    };
    if (!isNullOrUndef(presentScore)) {
      return {
        value: presentScore.value,
        label: statusToLabel(presentScore.status),
        status: presentScore.status
      };
    }

    return null;
  });

  return {
    presentMetrics,
    score
  };
}
