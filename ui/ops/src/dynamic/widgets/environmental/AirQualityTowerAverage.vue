<template>
  <v-card variant="tonal" class="pa-4">
    <div class="text-subtitle-1 font-weight-bold mb-3">{{ towerName }} Average</div>
    <div v-if="loading" class="d-flex justify-center py-4">
      <v-progress-circular indeterminate size="24" width="2"/>
    </div>
    <div v-else-if="hasData" class="d-flex flex-wrap ga-2">
      <v-chip
          v-for="metric in averageMetrics"
          :key="metric.key"
          :color="metric.color"
          variant="flat"
          size="small"
          class="font-weight-medium">
        {{ metric.label }}: {{ metric.displayValue }}
      </v-chip>
    </div>
    <div v-else class="text-disabled text-caption">No data</div>
  </v-card>
</template>

<script setup>
import {metrics, statusToColor, usePullAirQuality} from '@/traits/airQuality/airQuality.js';
import {computed, reactive, watch} from 'vue';

const props = defineProps({
  towerName: {
    type: String,
    required: true
  },
  sensorNames: {
    type: Array,
    required: true
  }
});

const metricKeys = ['carbonDioxideLevel', 'volatileOrganicCompounds', 'particulateMatter25', 'particulateMatter10'];
const metricLabels = {
  carbonDioxideLevel: 'CO₂',
  volatileOrganicCompounds: 'VOC',
  particulateMatter25: 'PM2.5',
  particulateMatter10: 'PM10'
};

// Store all sensor data reactively
const sensorData = reactive({});
const sensorLoading = reactive({});

// Set up watchers for each sensor
props.sensorNames.forEach(sensorName => {
  const {value, loading} = usePullAirQuality(() => sensorName);
  watch(value, (newVal) => {
    sensorData[sensorName] = newVal;
  }, {immediate: true});
  watch(loading, (newVal) => {
    sensorLoading[sensorName] = newVal;
  }, {immediate: true});
});

const loading = computed(() => {
  return props.sensorNames.some(name => sensorLoading[name]);
});

const hasData = computed(() => {
  return props.sensorNames.some(name => sensorData[name]);
});

const averageMetrics = computed(() => {
  const result = [];

  for (const key of metricKeys) {
    const values = [];
    for (const sensorName of props.sensorNames) {
      const data = sensorData[sensorName];
      if (data && data[key] !== undefined && data[key] !== null) {
        values.push(data[key]);
      }
    }

    if (values.length === 0) continue;

    const avg = values.reduce((a, b) => a + b, 0) / values.length;
    const metricInfo = metrics[key];
    const status = getStatusForValue(avg, metricInfo.levels);

    result.push({
      key,
      label: metricLabels[key],
      value: avg,
      displayValue: formatValue(avg, metricInfo.unit),
      color: statusToColor(status)
    });
  }

  return result;
});

function getStatusForValue(value, levels) {
  let status = '';
  for (const level of levels) {
    if (value >= level.value) {
      status = level.status;
    } else {
      break;
    }
  }
  return status;
}

function formatValue(value, unit) {
  if (value === null || value === undefined) return '-';
  const formatted = value < 10 ? value.toFixed(2) : Math.round(value);
  return `${formatted}${unit}`;
}
</script>
