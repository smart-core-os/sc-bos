import {closeResource, newResourceValue} from '@/api/resource.js';
import {pullAirQualitySensor} from '@/api/sc/traits/air-quality-sensor.js';
import {pullAirTemperature} from '@/api/sc/traits/air-temperature.js';
import {pullBrightnessSensor} from '@/api/sc/traits/brightness-sensor.js';
import {pullSoundSensor} from '@/api/sc/traits/sound-sensor.js';
import {toQueryObject, watchResource} from '@/util/traits.js';
import {isNullOrUndef} from '@/util/types.js';
import {AirQuality} from '@smart-core-os/sc-bos-ui-gen/proto/smartcore/bos/airqualitysensor/v1/air_quality_sensor_pb';
import {computed, onScopeDispose, reactive, toRefs, toValue} from 'vue';

/**
 * @param {MaybeRefOrGetter<string|PullAirQualityRequest.AsObject>} query
 * @param {MaybeRefOrGetter<boolean>=} paused
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

/**
 * @param {MaybeRefOrGetter<string|PullAirTemperatureRequest.AsObject>} query
 * @param {MaybeRefOrGetter<boolean>=} paused
 * @return {ToRefs<ResourceValue<AirTemperature.AsObject, PullAirTemperatureResponse>>}
 */
export function usePullAirTemperature(query, paused = false) {
  const resource = reactive(newResourceValue());
  onScopeDispose(() => closeResource(resource));
  const queryObject = computed(() => toQueryObject(query));
  watchResource(
      () => toValue(queryObject),
      () => toValue(paused),
      (req) => {
        pullAirTemperature(req, resource);
        return () => closeResource(resource);
      }
  );
  return toRefs(resource);
}

/**
 * @param {MaybeRefOrGetter<string|PullAmbientBrightnessRequest.AsObject>} query
 * @param {MaybeRefOrGetter<boolean>=} paused
 * @return {ToRefs<ResourceValue<AmbientBrightness.AsObject, PullAmbientBrightnessResponse>>}
 */
export function usePullBrightness(query, paused = false) {
  const resource = reactive(newResourceValue());
  onScopeDispose(() => closeResource(resource));
  const queryObject = computed(() => toQueryObject(query));
  watchResource(
      () => toValue(queryObject),
      () => toValue(paused),
      (req) => {
        pullBrightnessSensor(req, resource);
        return () => closeResource(resource);
      }
  );
  return toRefs(resource);
}

/**
 * @param {MaybeRefOrGetter<string|PullSoundLevelRequest.AsObject>} query
 * @param {MaybeRefOrGetter<boolean>=} paused
 * @return {ToRefs<ResourceValue<SoundLevel.AsObject, PullSoundLevelResponse>>}
 */
export function usePullSoundLevel(query, paused = false) {
  const resource = reactive(newResourceValue());
  onScopeDispose(() => closeResource(resource));
  const queryObject = computed(() => toQueryObject(query));
  watchResource(
      () => toValue(queryObject),
      () => toValue(paused),
      (req) => {
        pullSoundSensor(req, resource);
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
    labelText: 'CO2',
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
    unit: 'ug/m3',
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
    unit: 'ug/m3',
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
    unit: 'ug/m3',
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
  },
  'ambientTemperature': {
    label: 'Temperature',
    labelText: 'Temperature',
    unit: '°C',
    min: 0,
    max: 40,
    levels: [
      {value: 0, status: status.ERROR},
      {value: 18, status: status.SUCCESS},
      {value: 27, status: status.WARNING},
      {value: 30, status: status.ERROR}
    ]
  },
  'ambientHumidity': {
    label: 'Humidity',
    labelText: 'Humidity',
    unit: '%',
    min: 0,
    max: 100,
    levels: [
      {value: 0, status: status.ERROR},
      {value: 30, status: status.SUCCESS},
      {value: 60, status: status.WARNING},
      {value: 80, status: status.ERROR}
    ]
  },
  'brightnessLux': {
    label: 'Light',
    labelText: 'Light',
    unit: 'lux',
    min: 0,
    max: 2000,
    levels: [
      {value: 0, status: status.ERROR},
      {value: 300, status: status.WARNING},
      {value: 500, status: status.SUCCESS}
    ]
  },
  'soundPressureLevel': {
    label: 'Noise',
    labelText: 'Noise',
    unit: 'dB',
    min: 0,
    max: 100,
    levels: [
      {value: 0, status: status.SUCCESS},
      {value: 55, status: status.WARNING},
      {value: 70, status: status.ERROR}
    ]
  }
};

/**
 * Returns a comparison status based on indoor vs outdoor values.
 *
 * @param {number} indoorValue
 * @param {number|null} outdoorValue
 * @return {{status: valueOf<status>, diffPercent: number, arrow: string}}
 */
export function comparisonStatus(indoorValue, outdoorValue) {
  if (outdoorValue === null || outdoorValue === undefined || outdoorValue === 0) {
    return {status: status.SUCCESS, diffPercent: 0, arrow: '\u2248'};
  }
  const diffPercent = ((indoorValue - outdoorValue) / outdoorValue) * 100;
  if (diffPercent > 10) {
    return {status: status.ERROR, diffPercent, arrow: '\u2191'};
  } else if (diffPercent < -10) {
    return {status: status.SUCCESS, diffPercent, arrow: '\u2193'};
  } else {
    return {status: status.WARNING, diffPercent, arrow: '\u2248'};
  }
}

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
        continue;
      }
      let metricStatus = '';
      for (const level of m.levels) {
        if (v >= level.value) {
          metricStatus = level.status;
          continue;
        }
        break;
      }
      result[k] = {value: v, status: metricStatus};
    }
    return result;
  });

  const score = computed(() => {
    const presentScore = presentMetrics.value['score'];
    const statusToLabel = (s) => {
      switch (s) {
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
