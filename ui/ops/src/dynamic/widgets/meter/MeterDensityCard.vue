<template>
  <v-card class="d-flex flex-column">
    <v-toolbar color="transparent" v-if="props.title !== ''">
      <v-toolbar-title class="text-h4" style="overflow-wrap: break-word">{{ props.title }}</v-toolbar-title>
      <v-spacer/>
      <v-chip :color="densityThreshold.color" size="small" label class="flex-shrink-0">
        <v-icon start :icon="densityThreshold.icon"/>
        {{ densityThreshold.str }}
      </v-chip>
    </v-toolbar>
    <v-card-text class="flex-grow-1 d-flex flex-column align-center pt-0 position-relative">
      <!-- Dual gauges: outer = consumed, inner = generated -->
      <div v-if="consumedDensity !== null || generatedDensity !== null" class="gauge-container">
        <div class="gauge-wrapper">
          <!-- Outer gauge: consumed (no tooltip on the gauge itself) -->
          <circular-gauge
              :value="Math.min(Math.max(consumedDensity ?? 0, 0), densityGaugeMax)"
              :min="0"
              :max="densityGaugeMax"
              :color="densityThreshold.color"
              segments="30"
              width="160"
              class="outer-gauge">
            <!-- Inner gauge: generated with tooltip -->
            <div class="inner-gauge-wrapper">
              <v-tooltip location="top">
                <template #activator="{ props: generatedTooltipProps }">
                  <div v-bind="generatedTooltipProps" class="inner-gauge-activator">
                    <circular-gauge
                        v-if="generatedDensity !== null && !isNaN(generatedDensity)"
                        :value="Math.min(Math.max(generatedDensity, 0), densityGaugeMax)"
                        :min="0"
                        :max="densityGaugeMax"
                        :color="generatedColor"
                        segments="20"
                        width="110"
                        class="inner-gauge">
                      <span class="text-h2 font-weight-bold">{{ netDensityDisplayStr }}</span>
                    </circular-gauge>
                    <span v-else class="text-h2 font-weight-bold">{{ netDensityDisplayStr }}</span>
                  </div>
                </template>
                <span>Generated: {{ generatedDensityDisplayStr }} {{ _unit }}</span>
              </v-tooltip>
            </div>
          </circular-gauge>

          <!-- Consumed tooltip activator: donut-shaped overlay that excludes inner area -->
          <v-tooltip location="top">
            <template #activator="{ props: tooltipProps }">
              <div v-bind="tooltipProps" class="outer-gauge-activator"/>
            </template>
            <span>Consumed: {{ consumedDensityDisplayStr }} {{ _unit }}</span>
          </v-tooltip>
        </div>
      </div>
      <div v-else class="text-h2 opacity-40 my-2">—</div>

      <!-- Unit text - positioned absolutely between gauge and metrics -->
      <div class="unit-text text-caption opacity-70">{{ _unit }}</div>

      <!-- Metrics underneath -->
      <div class="metrics d-flex flex-row flex-wrap justify-center ga-8 align-self-stretch px-2 mt-2">
        <!-- Consumed -->
        <div class="d-flex flex-column ga-2 align-center" style="min-width: 70px; max-width: 90px;">
          <span class="text-caption font-weight-bold text-center" style="font-size: 0.7rem;">Consumed</span>
          <span
              class="text-caption font-weight-medium"
              :class="densityThreshold.color + '--text'"
              style="font-size: 0.75rem;">
            {{ consumedDensityDisplayStr }}
          </span>
          <v-chip
              v-if="props.showTrend"
              :color="getLocalTrendColor(consumedTrend, false)"
              size="x-small"
              style="height: 18px;">
            <v-icon :icon="getTrendIcon(consumedTrend)" size="x-small" start/>
            <span style="font-size: 0.65rem;">{{ formatTrend(consumedTrend) }}</span>
          </v-chip>
        </div>

        <!-- Generated -->
        <div class="d-flex flex-column ga-2 align-center" style="min-width: 70px; max-width: 90px;">
          <span class="text-caption font-weight-bold text-center" style="font-size: 0.7rem;">Generated</span>
          <span
              class="text-caption font-weight-medium"
              :class="generatedColor + '--text'"
              style="font-size: 0.75rem;">
            {{ generatedDensityDisplayStr }}
          </span>
          <v-chip
              v-if="props.showTrend"
              :color="getLocalTrendColor(generatedTrend, true)"
              size="x-small"
              style="height: 18px;">
            <v-icon :icon="getTrendIcon(generatedTrend)" size="x-small" start/>
            <span style="font-size: 0.65rem;">{{ formatTrend(generatedTrend) }}</span>
          </v-chip>
        </div>

        <!-- Net -->
        <div class="d-flex flex-column ga-2 align-center" style="min-width: 70px; max-width: 90px;">
          <span class="text-caption font-weight-bold text-center" style="font-size: 0.7rem;">Net</span>
          <span
              class="text-caption font-weight-medium"
              :class="netDensity !== null && netDensity < 0 ? 'success--text' : ''"
              style="font-size: 0.75rem;">
            {{ netDensityDisplayStr }}
          </span>
          <v-chip
              v-if="props.showTrend"
              :color="getLocalTrendColor(netTrend, false)"
              size="x-small"
              style="height: 18px;">
            <v-icon :icon="getTrendIcon(netTrend)" size="x-small" start/>
            <span style="font-size: 0.65rem;">{{ formatTrend(netTrend) }}</span>
          </v-chip>
        </div>
      </div>
    </v-card-text>
  </v-card>
</template>
<script setup>
import CircularGauge from '@/components/CircularGauge.vue';
import {useDateScale} from '@/components/charts/date.js';
import {useRollingValue} from '@/composables/rollingValue.js';
import {usePeriod} from '@/composables/time.js';
import {useMaxPeopleCount} from '@/dynamic/widgets/occupancy/occupancy.js';
import {useMeterReadingAt, usePullMeterReading} from '@/traits/meter/meter.js';
import {format} from '@/util/number.js';
import {formatTrend, getTrendColor, getTrendIcon} from '@/util/trend.js';
import {isNull} from '@/util/types.js';
import {useLocalProp} from '@/util/vue.js';
import {computed, ref, toRef, watch} from 'vue';


const props = defineProps({
  title: {
    type: String,
    default: 'Power Density'
  },
  name: {
    type: String, // name of the consumption meter device
    default: ''
  },
  generatedName: {
    type: String, // optional: name of the generation meter device (if separate from consumption meter)
    default: ''
  },
  meterUnit: {
    type: String,
    default: 'kWh' // TODO(get meter unit from DescribeMeterReading)
  },
  occupancy: {
    type: [
      String, // name of the device
      Object // Occupancy.AsObject
    ],
    default: null
  },
  showTrend: {
    type: Boolean,
    default: true
  },
  thresholds: {
    type: Array, // {density: number, str: string, icon: string} ordered by density (kW per day) in ascending order
    default: () => [
      {density: 0.3, str: 'Excellent', icon: 'mdi-leaf', color: 'success-lighten-1'},
      {density: 0.7, str: 'Acceptable', icon: 'mdi-check-circle-outline', color: 'success'},
      {density: 1.5, str: 'Poor', icon: 'mdi-alert-circle-outline', color: 'warning'},
      {density: Infinity, str: 'Inefficient', icon: 'mdi-fire-alert', color: 'error-lighten-1'},
    ]
  },
  period: {
    type: [String],
    default: 'day' // 'minute', 'hour', 'day', 'month', 'year'
  },
  offset: {
    type: [Number, String],
    default: 0 // Used via Math.abs, {period: 'day', offset: 1} means yesterday, and so on
  }
});

const _unit = computed(() => `${props.meterUnit} per person`);

// Gauge max: the last finite threshold, or 2 as a sensible default
const densityGaugeMax = computed(() => {
  const finite = props.thresholds.filter(t => isFinite(t.density));
  return finite.length ? finite[finite.length - 1].density + 0.5 : 2;
});

// Period — shared by occupancy and density calculations
const _offset = computed(() => -Math.abs(parseInt(props.offset)));
const {start, end} = usePeriod(toRef(props, 'period'), toRef(props, 'period'), _offset);

// Average occupancy over the period
const _occupancy = ref(0);
const {pastEdges} = useDateScale(start, end, useLocalProp(toRef(props, 'offset')));
const maxPeopleCounts = useMaxPeopleCount(toRef(props, 'occupancy'), pastEdges);
watch(maxPeopleCounts, (counts) => {
  if (!counts || counts.length === 0) {
    _occupancy.value = 0;
    return;
  }
  const sum = counts.reduce((acc, el) => {
    acc += el.y ?? 0;
    return acc;
  }, 0);
  _occupancy.value = sum / counts.length;
}, {deep: true, immediate: true});

// Live pull streams — density updates whenever the meter reports a new reading
const {value: liveConsumedReading} = usePullMeterReading(() => props.name || null);
const useGeneratedMeter = computed(() => props.generatedName !== '');
const {value: liveGeneratedReading} = usePullMeterReading(
    () => useGeneratedMeter.value ? props.generatedName : null
);

// Historical start-of-period readings (re-fetched when the period boundary changes)
const consumptionBefore = useMeterReadingAt(() => props.name || null, start, true);
const generationBefore = useMeterReadingAt(
    () => (useGeneratedMeter.value ? props.generatedName : props.name) || null,
    start,
    true
);

const consumedDensity = computed(() => {
  const after = liveConsumedReading.value;
  const before = consumptionBefore.value;
  if (isNull(after) || isNull(before)) return null;
  const consumed = Math.abs(after.usage - before.usage);
  const occupancy = _occupancy.value === 0 ? 1 : _occupancy.value;
  return consumed / occupancy;
});

const generatedDensity = computed(() => {
  const occupancy = _occupancy.value === 0 ? 1 : _occupancy.value;
  if (useGeneratedMeter.value) {
    const after = liveGeneratedReading.value;
    const before = generationBefore.value;
    if (isNull(after) || isNull(before)) return null;
    return Math.abs(after.usage - before.usage) / occupancy;
  }
  // Use produced field from consumption meter
  const after = liveConsumedReading.value;
  const before = consumptionBefore.value;
  if (isNull(after) || isNull(before)) return null;
  return Math.abs((after.produced ?? 0) - (before.produced ?? 0)) / occupancy;
});
const netDensity = computed(() => {
  if (consumedDensity.value === null && generatedDensity.value === null) return null;
  return (consumedDensity.value ?? 0) - (generatedDensity.value ?? 0);
});

const consumedDensityDisplayStr = computed(() => {
  if (consumedDensity.value === null || isNaN(consumedDensity.value)) return '—';
  return format(consumedDensity.value);
});

const generatedDensityDisplayStr = computed(() => {
  if (generatedDensity.value === null || isNaN(generatedDensity.value)) return '—';
  return format(generatedDensity.value);
});

const netDensityDisplayStr = computed(() => {
  if (netDensity.value === null || isNaN(netDensity.value)) return '—';
  return format(netDensity.value);
});

const densityThreshold = computed(() => {
  const densityValue = consumedDensity.value ?? 0;
  for (const threshold of props.thresholds) {
    if (densityValue <= threshold.density) {
      return {color: 'primary', ...threshold};
    }
  }
  return {str: '', icon: 'mdi-check-circle-outline', color: 'primary'};
});

const generatedColor = computed(() => {
  const generatedValue = generatedDensity.value ?? 0;
  // For generation, higher is better, so reverse the threshold logic
  // Use the same thresholds but interpret them inversely
  if (generatedValue >= 1.5) return 'success-lighten-1'; // Excellent generation
  if (generatedValue >= 0.7) return 'success'; // Good generation
  if (generatedValue >= 0.3) return 'warning'; // Moderate generation
  return 'error-lighten-1'; // Low/no generation
});

// Check if we have any actual meter data (not initial null state)
const hasData = computed(() => consumedDensity.value !== null || generatedDensity.value !== null);

// Track historical values for trend calculation
// Only track when we have actual data to avoid capturing initial null/zero as baseline
// This allows negative values and actual zeros once data is loaded
const consumedTracker = useRollingValue(() => props.showTrend && hasData.value && consumedDensity.value !== null ? consumedDensity.value : null);
const generatedTracker = useRollingValue(() => props.showTrend && hasData.value && generatedDensity.value !== null ? generatedDensity.value : null);
const netTracker = useRollingValue(() => props.showTrend && hasData.value && netDensity.value !== null ? netDensity.value : null);

// Compute trends using percentChange from rolling value trackers
const consumedTrend = computed(() => consumedTracker.percentChange.value);
const generatedTrend = computed(() => generatedTracker.percentChange.value);
const netTrend = computed(() => netTracker.percentChange.value);

// Reset trend baseline when sources or period changes
watch([toRef(props, 'name'), toRef(props, 'generatedName'), toRef(props, 'period'), toRef(props, 'offset')], () => {
  consumedTracker.reset();
  generatedTracker.reset();
  netTracker.reset();
});

/**
 * Get trend color based on change value
 * For consumed/net: down is good (green), up is bad (red)
 * For generated: up is good (green), down is bad (red)
 *
 * @param {number|null} change
 * @param {boolean} inverted - If true, positive change is good (green), negative is bad (red)
 * @return {string} Vuetify color name
 */
function getLocalTrendColor(change, inverted = false) {
  return getTrendColor(change, inverted);
}


</script>

<style scoped>
.unit-text {
  position: absolute;
  top: 120px;
  left: 50%;
  transform: translateX(-50%);
  z-index: 1;
  pointer-events: none;
}

.gauge-container {
  position: relative;
  display: inline-flex;
  align-items: center;
  justify-content: center;
}

.gauge-wrapper {
  position: relative;
  display: inline-flex;
  align-items: center;
  justify-content: center;
}

.outer-gauge {
  position: relative;
  pointer-events: none;
}

.outer-gauge-activator {
  position: absolute;
  top: 0;
  left: 0;
  width: 160px;
  height: 160px;
  border-radius: 50%;
  pointer-events: auto;
  /* Create a donut shape by masking out the center */
  -webkit-mask-image: radial-gradient(circle, transparent 0%, transparent 55px, black 55px, black 100%);
  mask-image: radial-gradient(circle, transparent 0%, transparent 55px, black 55px, black 100%);
  z-index: 1;
}

.inner-gauge-wrapper {
  position: absolute;
  top: 52%;
  left: 50%;
  transform: translate(-50%, -50%);
  display: flex;
  align-items: center;
  justify-content: center;
  pointer-events: none;
  z-index: 2;
}

.inner-gauge-activator {
  pointer-events: auto;
  cursor: pointer;
}

.inner-gauge {
  display: flex;
  align-items: center;
  justify-content: center;
}

:deep(.v-toolbar-title) {
  flex: 1 1 auto;
  overflow: visible;
  white-space: normal;
}

:deep(.v-toolbar-title__placeholder) {
  overflow: visible;
  white-space: normal;
}
</style>