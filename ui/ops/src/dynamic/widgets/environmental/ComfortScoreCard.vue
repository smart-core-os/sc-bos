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
                :color="getTrendColor(factorTrends.get(f.key))"
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
import {useComfortScore} from '@/dynamic/widgets/environmental/comfort.js';
import {useRollingValue} from '@/composables/rollingValue.js';
import {computed, toRef} from 'vue';

const props = defineProps({
  title: {type: String, default: 'Comfort Score'},
  airQuality: {type: String, default: ''},
  airTemperature: {type: String, default: ''},
  sound: {type: String, default: ''},
  tempSetpoint: {type: Number, default: 21},
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
    useRollingValue(() => factors.value.find(f => f.key === key)?.score ?? null)
  ])
);

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

const gaugeColor = computed(() => {
  if (score.value === null) return 'primary';
  if (score.value >= 70) return 'success-lighten-1';
  if (score.value >= 40) return 'warning';
  return 'error-lighten-1';
});

const scoreLabel = computed(() => {
  if (score.value === null) return 'No data';
  if (score.value >= 70) return 'Good';
  if (score.value >= 40) return 'Acceptable';
  return 'Poor';
});

const scoreIcon = computed(() => {
  if (score.value === null) return 'mdi-help-circle-outline';
  if (score.value >= 70) return 'mdi-emoticon-happy-outline';
  if (score.value >= 40) return 'mdi-emoticon-neutral-outline';
  return 'mdi-emoticon-sad-outline';
});

/**
 * @param {number|null} s factor percentage (0-100)
 * @return {string} Vuetify color name
 */
function factorColor(s) {
  if (s === null) return '';
  if (s >= 70) return 'success-lighten-1';
  if (s >= 40) return 'warning';
  return 'error-lighten-1';
}

/**
 * Get trend icon based on change value
 *
 * @param {number|null} change
 * @return {string} Material Design icon name
 */
function getTrendIcon(change) {
  if (change === null || change === undefined) return 'mdi-minus';
  return change > 0 ? 'mdi-trending-up' : 'mdi-trending-down';
}

/**
 * Get trend color based on change value
 *
 * @param {number|null} change
 * @return {string} Vuetify color name
 */
function getTrendColor(change) {
  if (change === null || change === undefined || Math.abs(change) < 0.01) return 'grey-lighten-1';
  return change > 0 ? 'success' : 'error';
}

/**
 * Format trend value for display
 *
 * @param {number|null} change
 * @return {string}
 */
function formatTrend(change) {
  if (change === null || change === undefined) return '0%';
  const rounded = Math.round(change);
  if (rounded === 0) return '0%';
  return `${rounded > 0 ? '+' : ''}${rounded}%`;
}
</script>
