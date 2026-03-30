<template>
  <div class="ml-3">
    <v-row class="mt-0 ml-0 pl-0">
      <h3 class="text-h3 pt-2 pb-6">Air Quality</h3>
    </v-row>

    <v-row class="mx-0 mb-4">
      <v-col cols="12" sm="6" class="pl-0">
        <v-select
            v-model="airDevice"
            hide-details
            :items="deviceOptions"
            item-title="label"
            item-value="value"
            label="Data source"
            :loading="isFetching"
            variant="outlined"/>
      </v-col>
      <v-col cols="12" sm="6">
        <v-select
            v-model="selectedMetric"
            hide-details
            :items="metricOptions"
            item-title="label"
            item-value="value"
            label="Metric"
            variant="outlined"/>
      </v-col>
    </v-row>

    <air-quality-history-card
        :source="airDevice"
        :metric="selectedMetric"
        :unit="metricUnit"
        :title="metricTitle"
        style="min-height: 480px"/>

    <!-- Most recent values -->
    <content-card class="mt-4 px-4 pt-4 pb-6">
      <v-row class="d-flex flex-row justify-space-between ml-15 mr-4">
        <v-col
            v-for="(recentValue, key) in filteredValues"
            :key="key"
            cols="auto"
            class="text-h1 align-self-auto"
            :class="{'selected-metric': key === selectedMetric}"
            style="line-height: 0.35em; cursor: pointer;"
            @click="selectedMetric = key">
          {{ recentValue.value }}
          <span style="font-size: 0.5em;">{{ recentValue.unit }}</span><br>
          <span
              class="text-h6"
              :style="{lineHeight: '0.4em', color: getColorForKey(key)}">
            {{ recentValue.label }}
          </span>
        </v-col>
      </v-row>
    </content-card>
  </div>
</template>

<script setup>
import AirQualityHistoryCard from '@/dynamic/widgets/environmental/AirQualityHistoryCard.vue';
import {HOUR, MINUTE} from '@/components/now';
import ContentCard from '@/components/ContentCard.vue';
import useAirQuality from '@/routes/ops/air-quality/useAirQualityHistory';
import {computed, reactive, ref} from 'vue';

const airQualityProps = reactive({
  filter: () => true,
  name: '',
  pollDelay: 15 * MINUTE,
  span: 15 * MINUTE,
  subsystem: 'zones',
  timeFrame: 24 * HOUR
});

const {
  acronyms,
  airQualitySensorHistoryValues,
  airDevice,
  deviceOptions,
  isFetching,
  readComfortValue
} = useAirQuality(airQualityProps);

// Metric selector
const selectedMetric = ref('score');
const metricOptions = computed(() => Object.entries(acronyms).map(([value, {label}]) => ({value, label})));
const metricTitle = computed(() => acronyms[selectedMetric.value]?.label ?? selectedMetric.value);
const metricUnit = computed(() => acronyms[selectedMetric.value]?.unit ?? '');

// Color mapping for recent values display
const colorMapping = {
  carbonDioxideLevel: 'rgba(255, 179, 186, 0.9)',
  volatileOrganicCompounds: 'rgba(255, 223, 186, 0.9)',
  airPressure: 'rgba(255, 255, 186, 0.9)',
  comfort: 'rgba(150, 255, 201, 0.9)',
  infectionRisk: 'rgba(100, 225, 255, 1)',
  score: 'rgba(100, 100, 255, 1)'
};
const getColorForKey = (key) => colorMapping[key] || 'rgba(0, 0, 0, 0.5)';

// Most recent values
const mostRecentValues = computed(() => {
  const airDeviceData = airQualitySensorHistoryValues[airDevice.value]?.data;
  if (!airDeviceData) return {};
  const mostRecent = airDeviceData[airDeviceData.length - 1];
  if (!mostRecent) return {};
  const result = {};
  for (const [key, value] of Object.entries(mostRecent.y)) {
    result[key] = value;
  }
  return result;
});

const filteredValues = computed(() => {
  return Object.entries(mostRecentValues.value).reduce((acc, [key, value]) => {
    const showValue = showHideValue(value, key);
    if (showValue.value) {
      acc[key] = showValue;
    }
    return acc;
  }, {});
});

const showHideValue = (value, key) => {
  const unit = acronyms[key]?.unit;
  const label = acronyms[key]?.label;
  if (key === 'comfort') {
    return {
      label,
      unit,
      value: readComfortValue(value) !== 'Unspecified' ? readComfortValue(value) : null
    };
  } else {
    return {
      label,
      unit,
      value: value > 0 ? value.toFixed(2) : null
    };
  }
};
</script>

<style scoped>
.selected-metric {
  opacity: 1;
}

.text-h1:not(.selected-metric) {
  opacity: 0.45;
}
</style>
