<template>
  <v-card v-if="hasAirQualitySensor" :loading="loading">
    <v-card-title class="text-h4 font-weight-medium d-flex align-center pl-7 pr-6">
      <span class="mr-auto">Air Quality</span>
      <span v-if="hasScore" class="text-uppercase font-weight-light">{{ score.label }}</span>
    </v-card-title>
    <circular-gauge
        v-if="hasScore"
        class="gauge--primary mt-2"
        :value="score.value"
        :min="metrics.score.min" :max="metrics.score.max"
        arc-start="0" arc-end="240" size="80"
        width="20"
        fill-color="accent"/>
    <v-card-text class="metrics text-subtitle-2 pl-7 pb-6 pt-3 pr-12">
      <status-chip v-for="m of displayMetrics" :key="m.title" :status="m.status">
        <!-- eslint-disable-next-line vue/no-v-text-v-html-on-component vue/no-v-html -->
        <span v-html="m.title"/>
      </status-chip>
    </v-card-text>
  </v-card>
</template>

<script setup>
import CircularGauge from '@/components/CircularGauge.vue';
import StatusChip from '@/components/StatusChip.vue';
import {listServices} from '@/api/ui/services.js';
import {metrics, useAirQuality, usePullAirQuality} from '@/routes/components/airQuality.js';
import {computed, ref, watch} from 'vue';

const AIR_QUALITY_SENSOR_TRAIT = 'smartcore.traits.AirQualitySensor';

// null = unknown (checking), true = sensor present, false = not present
const hasAirQualitySensor = ref(null);

const props = defineProps({
  name: {
    type: String,
    default: ''
  }
});

watch(() => props.name, async (name) => {
  if (!name) {
    hasAirQualitySensor.value = false;
    return;
  }
  try {
    const res = await listServices({name});
    hasAirQualitySensor.value =
        res.servicesList?.some(s => s.id === AIR_QUALITY_SENSOR_TRAIT) ?? false;
  } catch {
    hasAirQualitySensor.value = false;
  }
}, {immediate: true});

const orderedMetrics = [
  'volatileOrganicCompounds', 'carbonDioxideLevel',
  'particulateMatter25', 'particulateMatter10', 'particulateMatter1',
  'infectionRisk', 'airChangePerHour'
];
const {value: airQuality, loading} = usePullAirQuality(() => props.name);
const {presentMetrics, score} = useAirQuality(airQuality);
const displayMetrics = computed(() => {
  const res = [];
  const src = presentMetrics.value;
  for (const orderedMetric of orderedMetrics) {
    if (!Object.hasOwn(src, orderedMetric)) continue;
    res.push({
      status: src[orderedMetric].status,
      title: metrics[orderedMetric].label,
    });
  }
  return res;
});
const hasScore = computed(() => !!score.value);

</script>

<style scoped>
.v-card {
  display: grid;
  grid-template-columns: 1fr 50px;
  grid-template-rows: repeat(2, auto);
}

.v-card-title {
  min-height: 72px;
}

.gauge--primary {
  grid-column: -2 / -1;
  grid-row: 1 / -1;
  justify-self: end;
  align-self: start;
  padding: 1.2em 1em 1em 1em;
}

.metrics {
  display: flex;
  column-gap: 1.5em;
  grid-column: 1/1; /* display metrics on new line if circular gauge is missing */
  /* all these prevent overflow metrics from showing */
  row-gap: 2em;
  flex-wrap: wrap;
  max-height: 3.7em;
}
</style>