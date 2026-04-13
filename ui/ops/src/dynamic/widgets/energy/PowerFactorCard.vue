<template>
  <v-card class="d-flex flex-column">
    <v-toolbar color="transparent">
      <v-toolbar-title class="text-h4" style="overflow-wrap: break-word">{{ props.title }}</v-toolbar-title>
      <v-spacer/>
      <v-chip :color="gaugeColor" size="small" label class="flex-shrink-0">
        <v-icon start :icon="pfIcon"/>
        {{ pfLabel }}
      </v-chip>
    </v-toolbar>
    <v-card-text class="flex-grow-1 d-flex flex-column align-center pt-0">
      <circular-gauge
          v-if="hasEffectivePF"
          :value="effectivePowerFactor"
          :min="0"
          :max="1"
          :color="gaugeColor"
          segments="30"
          width="160">
        <span class="text-h2 font-weight-bold">{{ pfStr }}</span>
      </circular-gauge>
      <div v-else class="text-h2 opacity-40 my-2">—</div>

      <!-- Power metrics -->
      <div class="power-metrics d-flex flex-row flex-wrap justify-center ga-8 align-self-stretch px-2 mt-2">
        <template v-for="m in metrics" :key="m.key">
          <!-- Metric column -->
          <div v-if="m.hasValue" class="d-flex flex-column ga-2 align-center" style="min-width: 60px; max-width: 80px;">
            <span class="text-caption font-weight-bold text-center" style="font-size: 0.7rem;">{{ m.label }}</span>
            <span class="text-caption font-weight-medium" style="font-size: 0.75rem;">
              {{ format(m.value) }} {{ m.unit }}
            </span>
            <v-chip
                v-if="props.showTrend"
                :color="getTrendColor(metricTrends.get(m.key))"
                size="x-small"
                style="height: 18px;">
              <v-icon :icon="getTrendIcon(metricTrends.get(m.key))" size="x-small" start/>
              <span style="font-size: 0.65rem;">{{ formatTrend(metricTrends.get(m.key)) }}</span>
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
import {useElectricDemand, usePullElectricDemand} from '@/traits/electricDemand/electric.js';
import {format} from '@/util/number.js';
import {formatTrend, getTrendColor, getTrendIcon} from '@/util/trend.js';
import {computed, toRef, watch} from 'vue';

const props = defineProps({
  title: {type: String, default: 'Power Factor'},
  name: {type: String, default: ''},
  showTrend: {type: Boolean, default: true},
  thresholds: {
    type: Array,
    default: () => [
      {value: 0.90, label: 'Poor', color: 'error-lighten-1', icon: 'mdi-alert-circle'},
      {value: 0.95, label: 'Acceptable', color: 'warning', icon: 'mdi-check-circle-outline'},
      {value: 1.01, label: 'Excellent', color: 'success-lighten-1', icon: 'mdi-check-circle'},
    ],
  },
});

const {value} = usePullElectricDemand(toRef(props, 'name'));
const {
  powerFactor, hasPowerFactor,
  realPower,
  apparentPower, hasApparentPower,
  reactivePower, hasReactivePower,
} = useElectricDemand(value);

// Derive PF from realPower/apparentPower when not reported directly
const effectivePowerFactor = computed(() => {
  if (hasPowerFactor.value) return powerFactor.value;
  if (hasApparentPower.value && apparentPower.value > 0) {
    return Math.min(1, realPower.value / apparentPower.value);
  }
  return null;
});
const hasEffectivePF = computed(() => effectivePowerFactor.value !== null);

const pfStr = computed(() => hasEffectivePF.value ? effectivePowerFactor.value.toFixed(2) : '—');

// Create metrics array for consistent display
const metrics = computed(() => {
  return [
    {
      label: 'Real',
      key: 'real',
      value: realPower.value,
      unit: 'kW',
      hasValue: realPower.value !== 0,
    },
    {
      label: 'Apparent',
      key: 'apparent',
      value: apparentPower.value,
      unit: 'kVA',
      hasValue: hasApparentPower.value,
    },
    {
      label: 'Reactive',
      key: 'reactive',
      value: reactivePower.value,
      unit: 'kVAr',
      hasValue: hasReactivePower.value,
    },
  ];
});

// Track each metric individually
// Filter out zero values to avoid capturing 0 as baseline (which causes null percentChange)
// But allow negative values (reverse power flow / generation)
const realChange = useRollingValue(() => props.showTrend && realPower.value !== 0 ? realPower.value : null);
const apparentChange = useRollingValue(() => props.showTrend && apparentPower.value !== 0 ? apparentPower.value : null);
const reactiveChange = useRollingValue(() => props.showTrend && reactivePower.value !== 0 ? reactivePower.value : null);

// Reset trend baseline when the device name changes
watch(toRef(props, 'name'), () => {
  realChange.reset();
  apparentChange.reset();
  reactiveChange.reset();
});

// Compute trends for each metric
const metricTrends = computed(() => {
  const trends = new Map();
  trends.set('real', realChange.percentChange.value);
  trends.set('apparent', apparentChange.percentChange.value);
  trends.set('reactive', reactiveChange.percentChange.value);
  return trends;
});

const activeThreshold = computed(() => {
  if (!hasEffectivePF.value) return null;
  for (const t of props.thresholds) {
    if (effectivePowerFactor.value < t.value) return t;
  }
  return props.thresholds[props.thresholds.length - 1];
});

const gaugeColor = computed(() => activeThreshold.value?.color ?? 'primary');
const pfLabel = computed(() => activeThreshold.value?.label ?? 'No data');
const pfIcon = computed(() => activeThreshold.value?.icon ?? 'mdi-help-circle-outline');

</script>

<style scoped>
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
