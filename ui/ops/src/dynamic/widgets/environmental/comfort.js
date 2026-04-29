import {usePullAirQuality} from '@/traits/airQuality/airQuality.js';
import {usePullAirTemperature} from '@/traits/airTemperature/airTemperature.js';
import {usePullSoundLevel} from '@/traits/sound/sound.js';
import {cap, scale} from '@/util/number.js';
import {computed, toValue} from 'vue';

/**
 * @typedef {Object} ComfortFactor
 * @property {string} label - display name
 * @property {number|null} score - 0-100, higher is better; null when not available
 */

/**
 * Combines AirQualitySensor, AirTemperature and SoundSensor into a single comfort score (0–100).
 * Weights should sum to 1; factors with no data are excluded and remaining weights are renormalised.
 *
 * @param {import('vue').MaybeRefOrGetter<string>} airQualityName
 * @param {import('vue').MaybeRefOrGetter<string>} airTempName
 * @param {import('vue').MaybeRefOrGetter<string|string[]>} soundName
 * @param {{
 *   tempSetpoint?: number,
 *   weights?: {temp: number, co2: number, voc: number, pm25: number, sound: number}
 * }} [options]
 * @return {{
 *   score: import('vue').ComputedRef<number|null>,
 *   factors: import('vue').ComputedRef<ComfortFactor[]>,
 *   worstFactors: import('vue').ComputedRef<ComfortFactor[]>
 * }}
 */
export function useComfortScore(airQualityName, airTempName, soundName, options = {}) {
  const tempSetpoint = options.tempSetpoint ?? 21;
  const weights = options.weights ?? {temp: 0.30, co2: 0.30, voc: 0.20, pm25: 0.10, sound: 0.10};

  const {value: aq} = usePullAirQuality(() => toValue(airQualityName));
  const {value: at} = usePullAirTemperature(() => toValue(airTempName));
  const {value: sl} = usePullSoundLevel(() => toValue(soundName));

  // Individual 0–100 scores (higher = better)
  const factors = computed(() => {
    const _aq = aq.value;
    const _at = at.value;
    const _sl = sl.value;

    const co2 = _aq?.carbonDioxideLevel ?? null;
    const voc = _aq?.volatileOrganicCompounds ?? null;
    const pm25 = _aq?.particulateMatter25 ?? null;
    const tempC = _at?.ambientTemperature?.valueCelsius ?? null;
    const sound = _sl?.soundPressureLevel ?? null;

    return [
      {
        label: 'Temperature',
        key: 'temp',
        score: tempC !== null
          ? Math.round(cap(scale(Math.abs(tempC - tempSetpoint), 0, 5, 100, 0), 0, 100))
          : null,
        rawValue: tempC,
        rawUnit: '°C',
      },
      {
        label: 'CO₂',
        key: 'co2',
        score: co2 !== null ? Math.round(cap(scale(co2, 400, 1500, 100, 0), 0, 100)) : null,
      },
      {
        label: 'VOC',
        key: 'voc',
        score: voc !== null ? Math.round(cap(scale(voc, 0, 0.5, 100, 0), 0, 100)) : null,
      },
      {
        label: 'PM2.5',
        key: 'pm25',
        score: pm25 !== null ? Math.round(cap(scale(pm25, 0, 35, 100, 0), 0, 100)) : null,
      },
      {
        label: 'Sound',
        key: 'sound',
        score: sound !== null ? Math.round(cap(scale(sound, 30, 70, 100, 0), 0, 100)) : null,
      },
    ];
  });

  const score = computed(() => {
    let weightedSum = 0;
    let totalWeight = 0;
    for (const f of factors.value) {
      if (f.score !== null) {
        weightedSum += f.score * (weights[f.key] ?? 0);
        totalWeight += weights[f.key] ?? 0;
      }
    }
    if (totalWeight === 0) return null;
    return Math.round(weightedSum / totalWeight);
  });

  const worstFactors = computed(() =>
    factors.value
      .filter(f => f.score !== null)
      .sort((a, b) => a.score - b.score)
      .slice(0, 3)
  );

  return {score, factors, worstFactors};
}
