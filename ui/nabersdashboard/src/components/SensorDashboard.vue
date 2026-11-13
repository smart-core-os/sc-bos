<template>
  <div class="sensor-dashboard">
    <div class="sensor-content">
      <div v-if="anyLoading" class="d-flex justify-center py-1">
        <v-progress-circular indeterminate size="20" width="2" class="mr-2"/>
        <span class="text-body-2" style="opacity: 0.5">Updating sensor data...</span>
      </div>
      <div class="metric-grid">
        <SensorMetricCard
            v-for="card in metricCards"
            :key="card.key"
            :title="card.title"
            :unit="card.unit"
            :gauges="card.gauges"
            :min="card.min"
            :max="card.max"/>
      </div>
    </div>
    <div class="nabers-wrapper">
      <NabersSection/>
    </div>
  </div>
</template>

<script setup>
import {computed, reactive, watch} from 'vue';
import {
  comparisonStatus,
  usePullAirQuality,
  usePullAirTemperature,
  usePullSoundLevel
} from '@/traits/airQuality/airQuality.js';
import {useOutdoorAirQualityStore} from '@/stores/outdoorAirQuality.js';
import SensorMetricCard from './SensorMetricCard.vue';
import NabersSection from './nabers/NabersSection.vue';

const outdoorStore = useOutdoorAirQualityStore();

const props = defineProps({
  sensors: {
    type: Array,
    required: true,
    validator: (value) => value.every(s =>
        typeof s.label === 'string' && typeof s.sensorName === 'string'
    )
  }
});

// Per-sensor reactive data store
const sensorData = reactive({});
const sensorLoading = reactive({});

for (const sensor of props.sensors) {
  const name = sensor.sensorName;

  const {value: aqVal, loading: aqLoading} = usePullAirQuality(() => name);
  const {value: tempVal, loading: tempLoading} = usePullAirTemperature(() => name);
  const {value: soundVal, loading: soundLoading} = usePullSoundLevel(() => name);

  if (!sensorData[name]) sensorData[name] = {};

  watch(aqVal, v => {
    if (v) Object.assign(sensorData[name], {
      score: v.score,
      carbonDioxideLevel: v.carbonDioxideLevel,
      volatileOrganicCompounds: v.volatileOrganicCompounds,
      particulateMatter25: v.particulateMatter25
    });
  }, {immediate: true});

  watch(tempVal, v => {
    if (v) Object.assign(sensorData[name], {
      ambientTemperature: v.ambientTemperature?.valueCelsius,
      ambientHumidity: v.ambientHumidity
    });
  }, {immediate: true});

  watch(soundVal, v => {
    if (v) Object.assign(sensorData[name], {soundPressureLevel: v.soundPressureLevel});
  }, {immediate: true});

  watch([aqLoading, tempLoading, soundLoading], ([a, t, s]) => {
    sensorLoading[name] = a || t || s;
  }, {immediate: true});
}

const anyLoading = computed(() => props.sensors.some(s => sensorLoading[s.sensorName]));

const colorGreen   = '#4caf50';
const colorAmber   = '#f7a12c';
const colorRed     = '#f89c9b';
const colorOutdoor = '#7F00FF'; // SC Violet

/**
 * @param {string} status
 * @return {string}
 */
function statusToBarColor(status) {
  switch (status) {
    case 'error': return colorRed;
    case 'warning': return colorAmber;
    case 'success': return colorGreen;
    default: return colorGreen;
  }
}

// Keys whose outdoor baseline is a live reading from OpenWeatherMap
const liveOutdoorKeys = new Set(['ambientTemperature', 'ambientHumidity', 'particulateMatter25', 'particulateMatter10']);

const metricDefs = [
  {key: 'ambientTemperature',       title: 'Temperature',    unit: '°C',    min: 0, max: 40},
  {key: 'ambientHumidity',          title: 'Humidity',       unit: '%',     min: 0, max: 100},
  {key: 'carbonDioxideLevel',       title: 'CO<sub>2</sub>', unit: 'ppm',   min: 0, max: 5000},
  {key: 'volatileOrganicCompounds', title: 'VOC',            unit: 'ppm',   min: 0, max: 1},
  {key: 'particulateMatter25',      title: 'PM2.5',          unit: '\u00B5g/m\u00B3', min: 0, max: 50},
  {key: 'soundPressureLevel',       title: 'Noise',          unit: 'dB',    min: 0, max: 100}
];

const metricCards = computed(() => {
  const baselines = outdoorStore.outdoorBaselines;

  return metricDefs.map(md => {
    const gauges = [];
    const outdoorValue = baselines[md.key];

    for (const sensor of props.sensors) {
      const value = sensorData[sensor.sensorName]?.[md.key] ?? null;
      const comparison = comparisonStatus(value ?? 0, outdoorValue);
      gauges.push({
        label: sensor.label,
        value: value,
        color: value != null ? statusToBarColor(comparison.status) : 'rgba(255, 255, 255, 0.15)'
      });
    }

    if (outdoorValue !== null && outdoorValue !== undefined) {
      const outdoorLabel = liveOutdoorKeys.has(md.key) ? 'Outdoor' : 'Typical';
      gauges.push({label: outdoorLabel, value: outdoorValue, color: colorOutdoor});
    }

    return {key: md.key, title: md.title, unit: md.unit, min: md.min, max: md.max, gauges};
  });
});
</script>

<style scoped>
.iaq-row {
  flex: 0 0 auto;
  display: flex;
  align-items: center;
  padding: 8px 24px;
  border-bottom: 1px solid var(--sc-navy);
}

.iaq-label {
  width: 200px;
  flex-shrink: 0;
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.iaq-title {
  font-size: 22px;
  font-weight: 600;
  line-height: 1.2;
}

.iaq-unit {
  font-size: 14px;
  opacity: 0.45;
  font-weight: 400;
}

.iaq-gauges {
  display: flex;
  flex: 1;
  justify-content: space-around;
  align-items: center;
}

.iaq-cell {
  flex: 1;
  display: flex;
  justify-content: center;
  align-items: center;
}


.sensor-dashboard {
  display: flex;
  flex-direction: column;
  height: 100%;
  overflow: hidden;
}

.sensor-content {
  flex-shrink: 0;
}

.nabers-wrapper {
  flex: 1;
  min-height: 0;
  overflow-y: auto;
}

.metric-grid {
  display: grid;
  grid-template-columns: 1fr 1fr;
}
</style>
