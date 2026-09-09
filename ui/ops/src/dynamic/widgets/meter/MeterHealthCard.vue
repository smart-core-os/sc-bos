<template>
  <v-card class="d-flex flex-column">
    <v-toolbar v-if="!props.hideToolbar" color="transparent">
      <v-toolbar-title class="text-h4" style="overflow-wrap: break-word">{{ props.title }}</v-toolbar-title>
      <v-spacer/>
      <v-chip v-if="hasData" :color="statusColor" size="small" label class="flex-shrink-0">
        <v-icon start :icon="statusIcon"/>
        {{ statusLabel }}
      </v-chip>
    </v-toolbar>
    <v-card-text class="flex-grow-1 d-flex flex-column justify-center pt-0">
      <template v-if="hasData">
        <div class="text-h2 font-weight-bold" :class="`text-${statusColor}`">{{ percentStr }}</div>
        <div class="text-h5 mt-1">
          <span :class="`text-${statusColor}`">{{ online }}</span>
          <span class="opacity-60"> / {{ total }} reporting</span>
        </div>
        <v-progress-linear class="mt-4" :model-value="percent" :color="statusColor" height="6" rounded/>
        <div class="text-caption opacity-60 mt-2">{{ offlineStr }}</div>
      </template>
      <template v-else>
        <div class="text-h2 opacity-40 my-2">—</div>
        <div class="text-caption opacity-60">{{ error ? 'Not connected' : 'Counting devices…' }}</div>
      </template>
    </v-card-text>
  </v-card>
</template>

<script setup>
import {useDeviceHealthCount} from '@/composables/devices.js';
import {computed} from 'vue';

const props = defineProps({
  title: {type: String, default: 'Meter Health'},
  hideToolbar: {type: Boolean, default: false},
  /**
   * Which devices to count. These are the same options useDevices takes, so a card can be
   * pointed at a trait, a subsystem, a floor, or any query the devices API understands.
   *
   * The trait is the primary selector: a meter is a device implementing smartcore.bos.Meter,
   * wherever it happens to be filed. The rest narrow that down - they AND together - which is
   * why subsystem no longer defaults to 'metering'.
   */
  trait: {type: [String, Array], default: 'smartcore.bos.Meter'},
  subsystem: {type: String, default: null},
  floor: {type: String, default: null},
  conditions: {type: Array, default: () => []},
  /**
   * Which health check ids count towards not reporting.
   *
   * Must match whatever a HealthCheckTable shown alongside is scoped to. Without it the card
   * counts a failure on a point no trait reads as a device not reporting while the scoped
   * table does not, giving "654 / 680 reporting" above an empty table - which reads as a bug
   * in the table.
   */
  checkId: {type: [String, Array], default: null},
  /**
   * The number of devices that are supposed to exist. Zero uses the live count instead.
   *
   * Worth setting wherever a device disappearing matters: a live count shrinks along with
   * the devices, so a controller dropping off the cohort takes its meters out of both the
   * numerator and the denominator and the outage hides itself.
   */
  expected: {type: Number, default: 0},
  /** How often to recount, in milliseconds. */
  pollPeriod: {type: Number, default: 30000},
  /**
   * Percentage bands, ascending, matched with `percent < value`. The last entry is the
   * fallback, so its value must exceed 100.
   */
  thresholds: {
    type: Array,
    default: () => [
      {value: 90, label: 'Critical', color: 'error-lighten-1', icon: 'mdi-alert-circle'},
      {value: 98, label: 'Degraded', color: 'warning', icon: 'mdi-alert'},
      {value: 100.01, label: 'Reporting', color: 'success-lighten-1', icon: 'mdi-check-circle'}
    ]
  }
});

const {total, online, percent, hasData, error} = useDeviceHealthCount(() => ({
  trait: props.trait,
  subsystem: props.subsystem,
  floor: props.floor,
  conditions: props.conditions,
  checkId: props.checkId,
  expected: props.expected,
  pollPeriod: props.pollPeriod
}));

const percentStr = computed(() => `${percent.value.toFixed(1)}%`);
const offlineStr = computed(() => {
  const n = total.value - online.value;
  return n === 1 ? '1 not reporting' : `${n} not reporting`;
});

const activeThreshold = computed(() => {
  for (const t of props.thresholds) {
    if (percent.value < t.value) return t;
  }
  return props.thresholds[props.thresholds.length - 1];
});

const statusColor = computed(() => activeThreshold.value?.color ?? 'primary');
const statusLabel = computed(() => activeThreshold.value?.label ?? 'Unknown');
const statusIcon = computed(() => activeThreshold.value?.icon ?? 'mdi-help-circle-outline');
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
