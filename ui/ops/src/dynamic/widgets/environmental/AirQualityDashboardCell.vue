<template>
  <v-chip
      :color="scoreColor"
      :variant="loading ? 'outlined' : 'flat'"
      size="small"
      class="font-weight-medium">
    <template v-if="loading">
      <v-progress-circular indeterminate size="12" width="2" class="mr-1"/>
    </template>
    <template v-else-if="score">
      {{ Math.round(score.value) }}%
    </template>
    <template v-else>
      -
    </template>
  </v-chip>
</template>

<script setup>
import {statusToColor, useAirQuality, usePullAirQuality} from '@/traits/airQuality/airQuality.js';
import {computed} from 'vue';

const props = defineProps({
  sensorName: {
    type: String,
    required: true
  }
});

const {value: airQuality, loading} = usePullAirQuality(() => props.sensorName);
const {score} = useAirQuality(airQuality);

const scoreColor = computed(() => {
  if (!score.value) return undefined;
  return statusToColor(score.value.status);
});
</script>
