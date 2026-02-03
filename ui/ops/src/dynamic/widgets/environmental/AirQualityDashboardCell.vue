<template>
  <div class="d-flex flex-wrap ga-1 justify-center">
    <template v-if="loading">
      <v-progress-circular indeterminate size="16" width="2"/>
    </template>
    <template v-else-if="hasMetrics">
      <v-chip
          v-for="metric in displayMetrics"
          :key="metric.key"
          :color="metric.color"
          variant="flat"
          size="x-small"
          class="font-weight-medium">
        <span class="text-caption">{{ metric.label }}: {{ metric.displayValue }}</span>
      </v-chip>
    </template>
    <template v-else>
      <span class="text-disabled">-</span>
    </template>
  </div>
</template>

<script setup>
import {metrics, statusToColor, useAirQuality, usePullAirQuality} from '@/traits/airQuality/airQuality.js';
import {computed} from 'vue';

const props = defineProps({
  sensorName: {
    type: String,
    required: true
  }
});

const {value: airQuality, loading} = usePullAirQuality(() => props.sensorName);
const {presentMetrics} = useAirQuality(airQuality);

const metricKeys = ['carbonDioxideLevel', 'volatileOrganicCompounds', 'particulateMatter25', 'particulateMatter10'];
const metricLabels = {
  carbonDioxideLevel: 'CO₂',
  volatileOrganicCompounds: 'VOC',
  particulateMatter25: 'PM2.5',
  particulateMatter10: 'PM10'
};

const hasMetrics = computed(() => {
  return metricKeys.some(key => presentMetrics.value[key]);
});

const displayMetrics = computed(() => {
  const result = [];
  for (const key of metricKeys) {
    const metric = presentMetrics.value[key];
    if (!metric) continue;
    const metricInfo = metrics[key];
    result.push({
      key,
      label: metricLabels[key],
      value: metric.value,
      displayValue: formatValue(metric.value, metricInfo.unit),
      color: statusToColor(metric.status)
    });
  }
  return result;
});

function formatValue(value, unit) {
  if (value === null || value === undefined) return '-';
  const formatted = value < 10 ? value.toFixed(2) : Math.round(value);
  return `${formatted}${unit}`;
}
</script>
