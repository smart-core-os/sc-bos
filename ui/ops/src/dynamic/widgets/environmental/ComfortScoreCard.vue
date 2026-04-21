<template>
  <v-card class="d-flex flex-column">
    <v-toolbar color="transparent">
      <v-toolbar-title class="text-h4" style="overflow-wrap: break-word">{{ props.title }}</v-toolbar-title>
      <v-spacer/>
      <v-chip :color="gaugeColor" size="small" label class="flex-shrink-0">
        <v-icon start :icon="scoreIcon"/>
        {{ scoreLabel }}
      </v-chip>
    </v-toolbar>
    <v-card-text class="flex-grow-1 d-flex flex-column align-center pt-0">
      <circular-gauge
          v-if="score !== null"
          :value="score"
          :min="0"
          :max="100"
          :color="gaugeColor"
          segments="30"
          width="160">
        <span class="text-h2 font-weight-bold">{{ score }}</span>
      </circular-gauge>
      <div v-else class="text-h2 opacity-40 my-2">—</div>

      <!-- Worst factors -->
      <div class="worst-factors d-flex flex-row flex-wrap justify-center ga-3 align-self-stretch px-2 mt-2">
        <template v-for="f in factors" :key="f.label">
          <!-- Metric column -->
          <div class="d-flex flex-column ga-2 align-center" style="min-width: 50px; max-width: 80px;">
            <span class="text-caption font-weight-bold text-center" style="font-size: 0.7rem;">{{ f.label }}</span>
            <span
                class="text-caption font-weight-medium"
                :class="factorColor(f.score) + '--text'"
                style="font-size: 0.75rem;">
              <template v-if="'rawValue' in f">{{ f.rawValue !== null ? f.rawValue.toFixed(1) + f.rawUnit : '—' }}</template>
              <template v-else>{{ f.score !== null ? f.score + '%' : '—' }}</template>
            </span>
            <v-chip
                v-if="props.showTrend"
                :color="getTrendColor(factorTrends.get(f.key), true)"
                size="x-small"
                style="height: 18px;">
              <v-icon :icon="getTrendIcon(factorTrends.get(f.key))" size="x-small" start/>
              <span style="font-size: 0.65rem;">{{ formatTrend(factorTrends.get(f.key)) }}</span>
            </v-chip>
          </div>
        </template>
      </div>
    </v-card-text>
  </v-card>
</template>

<script setup>
import CircularGauge from '@/components/CircularGauge.vue';
import {useRollingValue} from '@/composables/rollingValue.js';
import {useComfortScore} from '@/dynamic/widgets/environmental/comfort.js';
import {formatTrend, getTrendColor, getTrendIcon} from '@/util/trend.js';
import {computed, toRef, watch} from 'vue';

const props = defineProps({
  title: {type: String, default: 'Comfort Score'},
  airQuality: {type: String, default: ''},
  airTemperature: {type: String, default: ''},
  sound: {type: String, default: ''},
  tempSetpoint: {type: Number, default: 21},
  showTrend: {type: Boolean, default: true},
  thresholds: {
    type: Array,
    default: () => [
      {value: 40, label: 'Poor', color: 'error-lighten-1', icon: 'mdi-emoticon-sad-outline'},
      {value: 70, label: 'Acceptable', color: 'warning', icon: 'mdi-emoticon-neutral-outline'},
      {value: 101, label: 'Good', color: 'success-lighten-1', icon: 'mdi-emoticon-happy-outline'},
    ],
  },
});

const {score, factors} = useComfortScore(
  toRef(props, 'airQuality'),
  toRef(props, 'airTemperature'),
  toRef(props, 'sound'),
  {tempSetpoint: props.tempSetpoint}
);

// Track each factor score independently so that a snapshot captured before one sensor's data
// arrives (score=null) doesn't suppress the trend ticker for that factor once data loads.
const factorOldScores = Object.fromEntries(
  ['temp', 'co2', 'voc', 'pm25', 'sound'].map(key => [
    key,
    useRollingValue(() => props.showTrend ? (factors.value.find(f => f.key === key)?.score ?? null) : null)
  ])
);

// Reset trend baseline when the source sensors or setpoint changes
watch([toRef(props, 'airQuality'), toRef(props, 'airTemperature'), toRef(props, 'sound'), toRef(props, 'tempSetpoint')], () => {
  for (const key in factorOldScores) {
    factorOldScores[key].reset();
  }
});

// Compute trends for each factor
const factorTrends = computed(() => {
  const trends = new Map();
  for (const factor of factors.value) {
    const tracker = factorOldScores[factor.key];
    const percentChange = tracker?.percentChange?.value;

    // Use the percentChange directly from the rolling value tracker
    trends.set(factor.key, percentChange);
  }
  return trends;
});

const activeThreshold = computed(() => {
  if (score.value === null) return null;
  for (const t of props.thresholds) {
    if (score.value < t.value) return t;
  }
  return props.thresholds[props.thresholds.length - 1];
});

const gaugeColor = computed(() => activeThreshold.value?.color ?? 'primary');
const scoreLabel = computed(() => activeThreshold.value?.label ?? 'No data');
const scoreIcon = computed(() => activeThreshold.value?.icon ?? 'mdi-help-circle-outline');

/**
 * @param {number|null} s factor percentage (0-100)
 * @return {string} Vuetify color name
 */
function factorColor(s) {
  if (s === null) return '';
  for (const t of props.thresholds) {
    if (s < t.value) return t.color;
  }
  return props.thresholds[props.thresholds.length - 1].color;
}
</script>
