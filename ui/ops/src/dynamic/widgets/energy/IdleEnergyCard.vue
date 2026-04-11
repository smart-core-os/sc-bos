<template>
  <v-card class="d-flex flex-column">
    <v-toolbar color="transparent">
      <v-toolbar-title class="text-h4">{{ displayTitle }}</v-toolbar-title>
      <v-spacer/>
      <v-chip :color="badgeColor" size="small" label class="flex-shrink-0">
        <v-icon start :icon="badgeIcon"/>
        {{ badgeLabel }}
      </v-chip>
    </v-toolbar>
    <v-card-text class="flex-grow-1 d-flex flex-column align-center pt-0">
      <!-- No data -->
      <div v-if="!hasData" class="text-h2 opacity-40 my-2">—</div>

      <!-- Has energy data but no meaningful idle — all zones occupied -->
      <div v-else-if="totalIdle < 0.001" class="d-flex flex-column align-center my-2 ga-1">
        <span class="text-h2 font-weight-bold">{{ totalStr }}</span>
        <span class="text-caption text-medium-emphasis">{{ props.unit }}</span>
        <span class="text-body-2 text-success mt-1">All zones occupied</span>
      </div>

      <!-- Show gauge with idle data -->
      <template v-else>
        <div class="gauge-container">
          <circular-gauge
              :value="100 - Math.min(wastePercent, 100)"
              :min="0"
              :max="100"
              :color="badgeColor"
              segments="30"
              width="160">
            <span class="text-h2 font-weight-bold">{{ wasteStr }}</span>
          </circular-gauge>

          <!-- Zone label (only when multiple zones) -->
          <div v-if="worstZoneLabel" class="zone-label text-caption text-medium-emphasis mt-1">
            {{ worstZoneLabel }}
          </div>
        </div>
      </template>

      <!-- Metrics underneath - always in DOM so they align with other cards -->
      <div class="metrics d-flex flex-row flex-wrap justify-center ga-8 align-self-stretch px-2 mt-2">
        <!-- Idle -->
        <div class="d-flex flex-column ga-2 align-center" style="min-width: 70px; max-width: 90px;">
          <span class="text-caption font-weight-bold text-center" style="font-size: 0.7rem;">Idle</span>
          <span class="text-caption font-weight-medium" style="font-size: 0.75rem;">
            {{ idleStr }} {{ props.unit }}
          </span>
          <v-chip
              v-if="idleTrend !== null"
              :color="getTrendColor(idleTrend)"
              size="x-small"
              style="height: 18px;">
            <v-icon :icon="getTrendIcon(idleTrend)" size="x-small" start/>
            <span style="font-size: 0.65rem;">{{ formatTrend(idleTrend) }}</span>
          </v-chip>
          <span v-else class="text-caption" style="font-size: 0.65rem;">—</span>
        </div>

        <!-- Total -->
        <div class="d-flex flex-column ga-2 align-center" style="min-width: 70px; max-width: 90px;">
          <span class="text-caption font-weight-bold text-center" style="font-size: 0.7rem;">Total</span>
          <span class="text-caption font-weight-medium" style="font-size: 0.75rem;">
            {{ totalStr }} {{ props.unit }}
          </span>
          <v-chip
              v-if="totalTrend !== null"
              :color="getTrendColor(totalTrend)"
              size="x-small"
              style="height: 18px;">
            <v-icon :icon="getTrendIcon(totalTrend)" size="x-small" start/>
            <span style="font-size: 0.65rem;">{{ formatTrend(totalTrend) }}</span>
          </v-chip>
          <span v-else class="text-caption" style="font-size: 0.65rem;">—</span>
        </div>
      </div>
    </v-card-text>
  </v-card>
</template>

<script setup>
import {useDateScale} from '@/components/charts/date.js';
import CircularGauge from '@/components/CircularGauge.vue';
import {useRollingValue} from '@/composables/rollingValue.js';
import {useIdleConsumption, usePullOccupancyState} from '@/dynamic/widgets/energy/idle.js';
import {usePullMeterReading} from '@/traits/meter/meter.js';
import {format} from '@/util/number.js';
import {useLocalProp} from '@/util/vue.js';
import {computed, effectScope, onScopeDispose, reactive, ref, toRef, toValue, watch} from 'vue';

const props = defineProps({
  title: {type: String, default: 'Idle Energy'},
  zones: {type: Array, default: () => []},
  name: {type: String, default: ''},
  occupancy: {type: String, default: ''},
  unit: {type: String, default: 'kWh'},
  start: {type: [String, Number, Date], default: 'hour'},
  end: {type: [String, Number, Date], default: 'hour'},
  offset: {type: [Number, String], default: 0},
  // Pull-based updates configuration
  usePull: {type: Boolean, default: true},
  pullLookback: {type: Number, default: 5 * 60 * 1000}, // 5 minutes in milliseconds
  thresholds: {
    type: Array,
    default: () => [
      {percent: 0, label: 'Occupied', color: 'success', icon: 'mdi-check-circle-outline'},
      {percent: 10, label: 'Efficient', color: 'success', icon: 'mdi-leaf'},
      {percent: 30, label: 'Moderate', color: 'warning', icon: 'mdi-alert-circle-outline'},
      {percent: Infinity, label: 'Wasteful', color: 'error', icon: 'mdi-fire-alert'},
    ],
  },
});

const _start = useLocalProp(toRef(props, 'start'));
const _end = useLocalProp(toRef(props, 'end'));
const _offset = useLocalProp(toRef(props, 'offset'));

const {edges, pastEdges} = useDateScale(_start, _end, _offset);

// Build list of zones from props
const allZones = computed(() => {
  const result = [];

  // Add zones from array prop
  if (props.zones?.length > 0) {
    for (const z of props.zones) {
      if (z?.name && z?.occupancy) {
        result.push(z);
      }
    }
  }

  // Add single zone from name/occupancy props
  if (props.name && props.occupancy) {
    result.push({name: props.name, occupancy: props.occupancy});
  }

  return result;
});

// Map to hold consumptions keyed by zone identifier (not reactive - just for storage)
const consumptionsByZone = new Map();

// Reactive array of consumptions for computations
const consumptions = reactive([]);

// Watch zones and manage consumption lifecycle using effectScope pattern
watch(allZones, (currentZones) => {
  // Build set of current zone keys
  const currentKeys = new Set();
  for (const zone of currentZones) {
    currentKeys.add(`${zone.name}:${zone.occupancy}`);
  }

  // Remove consumptions for zones that no longer exist
  for (const [key, consumption] of consumptionsByZone.entries()) {
    if (!currentKeys.has(key)) {
      consumption?._scope?.stop();
      consumptionsByZone.delete(key);
    }
  }

  // Add consumptions for new zones
  for (const zone of currentZones) {
    const key = `${zone.name}:${zone.occupancy}`;

    // Skip if already tracking this zone
    if (consumptionsByZone.has(key)) continue;

    // Create new consumption tracking in an effect scope
    const scope = effectScope();
    let consumptionData;

    scope.run(() => {
      const {state: realtimeState, stateHistory: realtimeHistory} = props.usePull
          ? usePullOccupancyState(() => zone.occupancy, () => props.pullLookback)
          : {state: computed(() => null), stateHistory: computed(() => [])};

      const {value: liveMeterReading} = props.usePull
          ? usePullMeterReading(() => zone.name)
          : {value: ref(null)};

      consumptionData = useIdleConsumption(
          () => zone.name,
          () => zone.occupancy,
          edges,
          pastEdges,
          realtimeState,
          liveMeterReading,
          realtimeHistory
      );
    });

    // Attach metadata
    consumptionData._scope = scope;
    consumptionData._zone = zone;
    consumptionsByZone.set(key, consumptionData);
  }

  // Update consumptions array (clear and rebuild to maintain reactivity)
  consumptions.splice(0, consumptions.length);
  currentZones.forEach(zone => {
    const consumption = consumptionsByZone.get(`${zone.name}:${zone.occupancy}`);
    if (consumption) {
      consumptions.push(consumption);
    }
  });
}, {immediate: true});

// Cleanup on unmount
onScopeDispose(() => {
  for (const [, consumption] of consumptionsByZone.entries()) {
    consumption?._scope?.stop();
  }
  consumptionsByZone.clear();
});


// Generate dynamic title based on props
const displayTitle = computed(() => {
  if (props.title) return props.title;

  // Determine period text
  let periodText = 'This Hour';
  const start = toValue(_start);
  const offset = toValue(_offset);

  if (start === 'hour' && offset === 0) {
    periodText = 'This Hour';
  } else if (start === 'hour' && offset === 1) {
    periodText = 'Last Hour';
  } else if (start === 'day' && offset === 0) {
    periodText = 'Today';
  } else if (start === 'day' && offset === 1) {
    periodText = 'Yesterday';
  } else if (start === 'week' && offset === 0) {
    periodText = 'This Week';
  } else if (start === 'week' && offset === 1) {
    periodText = 'Last Week';
  } else if (start === 'month' && offset === 0) {
    periodText = 'This Month';
  } else if (start === 'month' && offset === 1) {
    periodText = 'Last Month';
  } else if (start === 'year' && offset === 0) {
    periodText = 'This Year';
  } else if (start === 'year' && offset === 1) {
    periodText = 'Last Year';
  }

  return `Idle Energy ${periodText}`;
});

// Find zone with worst idle energy
const worstZoneIdx = computed(() => {
  const items = consumptions;
  if (items.length === 0) return -1;

  let maxIdx = -1;
  let maxIdle = 0;
  let firstWithData = -1;

  for (let i = 0; i < items.length; i++) {
    const consumption = items[i];
    const energy = toValue(consumption.totalEnergy);
    const idle = toValue(consumption.totalIdle);

    if (energy === null || idle === null) continue;

    // Track first zone with any energy data
    if (firstWithData === -1) {
      firstWithData = i;
    }

    // Track zone with most idle energy (positive values only)
    if (idle > maxIdle) {
      maxIdle = idle;
      maxIdx = i;
    }
  }

  // If no zone has idle energy, return first zone with data
  return maxIdx === -1 ? firstWithData : maxIdx;
});

// Check if we have any data
const hasData = computed(() => {
  const items = consumptions;
  if (items.length === 0) return false;
  return items.some(c => {
    const energy = toValue(c?.totalEnergy);
    return energy !== null;
  });
});

// Label for the worst zone, shown only when there are multiple zones
const worstZoneLabel = computed(() => {
  const idx = worstZoneIdx.value;
  const zones = allZones.value;
  if (idx < 0 || zones.length <= 1) return '';
  return zones[idx].label ?? zones[idx].name ?? '';
});

// Aggregate metrics across all zones
const totalIdle = computed(() => {
  let sum = 0;
  let hasValue = false;
  for (const c of consumptions) {
    const val = toValue(c?.totalIdle);
    if (val !== null) {
      sum += val;
      hasValue = true;
    }
  }
  return hasValue ? sum : null;
});

const totalEnergy = computed(() => {
  let sum = 0;
  let hasValue = false;
  for (const c of consumptions) {
    const val = toValue(c?.totalEnergy);
    if (val !== null) {
      sum += val;
      hasValue = true;
    }
  }
  return hasValue ? sum : null;
});

const totalHistoricalIdle = computed(() => {
  let sum = 0;
  let hasValue = false;
  for (const c of consumptions) {
    const val = toValue(c?.historicalIdle);
    if (val !== null) {
      sum += val;
      hasValue = true;
    }
  }
  return hasValue ? sum : null;
});

const totalHistoricalEnergy = computed(() => {
  let sum = 0;
  let hasValue = false;
  for (const c of consumptions) {
    const val = toValue(c?.historicalEnergy);
    if (val !== null) {
      sum += val;
      hasValue = true;
    }
  }
  return hasValue ? sum : null;
});

const wastePercent = computed(() => {
  const total = totalEnergy.value;
  const idle = totalIdle.value;

  if (!total) return 0;
  return Math.abs((idle / total) * 100);
});

const idleStr = computed(() => format(totalIdle.value));
const totalStr = computed(() => format(totalEnergy.value));
const wasteStr = computed(() => `${wastePercent.value.toFixed(1)}%`);

const activeThreshold = computed(() => {
  for (const t of props.thresholds) {
    if (wastePercent.value <= t.percent) return t;
  }
  return props.thresholds[props.thresholds.length - 1];
});

const badgeColor = computed(() => activeThreshold.value.color);
const badgeLabel = computed(() => activeThreshold.value.label);
const badgeIcon = computed(() => activeThreshold.value.icon);

// Trend tracking for aggregated totals
const idleChange = useRollingValue(() => {
  if (!hasData.value) return null;
  // Use historical baseline if we have completed at least one bucket with non-zero data
  return totalHistoricalIdle.value > 0 ? totalIdle.value : null;
});

const energyChange = useRollingValue(() => {
  if (!hasData.value) return null;
  // Use historical baseline if we have completed at least one bucket with non-zero data
  return totalHistoricalEnergy.value > 0 ? totalEnergy.value : null;
});

// Reset trend baseline when the period changes
watch([_start, _end, _offset], () => {
  idleChange.reset();
  energyChange.reset();
});

const idleTrend = computed(() => idleChange.percentChange.value);
const totalTrend = computed(() => energyChange.percentChange.value);

/**
 * Get trend icon based on change value
 *
 * @param {number|null} change - The change value
 * @return {string} The icon name
 */
function getTrendIcon(change) {
  if (change === null || change === undefined || Math.abs(change) < 0.05) return 'mdi-minus';
  return change > 0 ? 'mdi-trending-up' : 'mdi-trending-down';
}

/**
 * Get trend color based on change value
 *
 * @param {number|null} change - The change value
 * @return {string} The color name
 */
function getTrendColor(change) {
  if (change === null || change === undefined || Math.abs(change) < 0.05) return 'grey-lighten-1';
  return change > 0 ? 'error' : 'success';
}

/**
 * Format trend value as percentage string
 *
 * @param {number|null} change - The change value
 * @return {string} Formatted percentage string
 */
function formatTrend(change) {
  if (change === null || change === undefined || Math.abs(change) < 0.05) return '0%';
  const absChange = Math.abs(change);
  if (absChange < 10) {
    const formatted = change.toFixed(1);
    if (formatted === '0.0' || formatted === '-0.0') return '0%';
    return `${change > 0 ? '+' : ''}${formatted}%`;
  }
  const rounded = Math.round(change);
  return `${rounded > 0 ? '+' : ''}${rounded}%`;
}

</script>
<style scoped>
.zone-label {
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
</style>