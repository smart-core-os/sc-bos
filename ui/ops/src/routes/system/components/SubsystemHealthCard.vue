<template>
  <v-card width="300px" class="subsystem-health-card">
    <div class="subsystem-health-card__accent" :style="{ background: accentColor }"/>
    <v-card-text class="pa-3" style="position: relative; overflow: hidden;">
      <!-- Watermark icon -->
      <v-icon class="subsystem-health-card__watermark">{{ icon }}</v-icon>

      <!-- Header -->
      <div class="d-flex align-start mb-1" style="position: relative;">
        <v-icon size="20" class="mr-2 mt-1" :color="statusColor">{{ icon }}</v-icon>
        <div class="flex-grow-1 min-width-0">
          <div class="text-subtitle-2 font-weight-bold text-truncate" :title="title">{{ title }}</div>
          <div v-if="description" class="text-caption text-medium-emphasis text-truncate">{{ description }}</div>
        </div>
      </div>

      <v-divider class="my-2"/>

      <!-- Aggregate status row -->
      <div class="d-flex align-center mb-2" style="position: relative;">
        <v-icon size="16" :color="statusColor" class="mr-1">{{ statusIcon }}</v-icon>
        <span class="text-caption font-weight-medium" :class="`text-${statusColor}`">{{ statusLabel }}</span>
      </div>

      <!-- Individual check rows -->
      <template v-if="checkEntries.length > 0">
        <div
            v-for="entry in checkEntries"
            :key="entry.check.name"
            class="d-flex align-center check-row py-1"
            style="position: relative;">
          <normality-icon :model-value="{ normality: worstNormality(entry.items) }" size="14" class="mr-1"/>
          <reliability-icon :model-value="{ reliability: { state: worstReliabilityState(entry.items) } }" size="14" class="mr-2"/>
          <span class="text-caption text-truncate flex-grow-1" :title="entry.check.displayName ?? entry.check.name">
            {{ entry.check.displayName ?? entry.check.name }}
          </span>
          <span class="text-caption text-medium-emphasis flex-shrink-0 ml-1">
            {{ checkCountLabel(entry.items) }}
          </span>
        </div>
      </template>
      <div v-else class="text-caption text-medium-emphasis" style="position: relative;">
        No checks configured
      </div>
    </v-card-text>
  </v-card>
</template>

<script setup>
import {usePullHealthChecks} from '@/traits/health/health.js';
import NormalityIcon from '@/traits/health/NormalityIcon.vue';
import ReliabilityIcon from '@/traits/health/ReliabilityIcon.vue';
import {HealthCheck} from '@smart-core-os/sc-bos-ui-gen/proto/smartcore/bos/health/v1/health_pb';
import {computed} from 'vue';

const props = defineProps({
  title: {type: String, required: true},
  icon: {type: String, default: 'mdi-cube-outline'},
  description: {type: String, default: ''},
  checks: {
    /** @type {import('vue').PropType<Array<{name: string, displayName?: string}>>} */
    type: Array,
    default: () => []
  }
});

// One stream per configured device name. props.checks is static config — safe to call in a loop.
const checkResources = props.checks.map(check => {
  const {value: checksRef, streamError} = usePullHealthChecks(() => check.name);
  return {check, checksRef, streamError};
});

const checkEntries = computed(() => {
  return checkResources.map(r => ({
    check: r.check,
    items: Object.values(r.checksRef.value ?? {})
  }));
});

function worstNormality(items) {
  if (!items.length) return 0;
  return Math.max(...items.map(c => c.normality ?? 0));
}

function worstReliabilityState(items) {
  if (!items.length) return 0;
  return Math.max(...items.map(c => c.reliability?.state ?? 0));
}

function checkCountLabel(items) {
  if (!items.length) return '';
  const normal = items.filter(c => (c.normality ?? 0) === HealthCheck.Normality.NORMAL).length;
  return `${normal}/${items.length}`;
}

const aggregateStatus = computed(() => {
  const allItems = checkEntries.value.flatMap(e => e.items);
  const hasStreamError = checkResources.some(r => r.streamError.value !== null);
  if (allItems.length === 0) return hasStreamError ? 'error' : 'unknown';
  const anyUnreliable = allItems.some(c => (c.reliability?.state ?? 0) > HealthCheck.Reliability.State.RELIABLE);
  if (anyUnreliable) return 'error';
  const anyAbnormal = allItems.some(c => (c.normality ?? 0) > HealthCheck.Normality.NORMAL);
  if (anyAbnormal) return 'warning';
  return 'ok';
});

const statusColor = computed(() => {
  switch (aggregateStatus.value) {
    case 'ok': return 'success';
    case 'warning': return 'warning';
    case 'error': return 'error';
    default: return 'medium-emphasis';
  }
});

const statusIcon = computed(() => {
  switch (aggregateStatus.value) {
    case 'ok': return 'mdi-check-circle';
    case 'warning': return 'mdi-alert';
    case 'error': return 'mdi-alert-circle';
    default: return 'mdi-help-circle';
  }
});

const statusLabel = computed(() => {
  switch (aggregateStatus.value) {
    case 'ok': return 'Healthy';
    case 'warning': return 'Degraded';
    case 'error': return 'Fault';
    default: return 'Unknown';
  }
});

const accentColor = computed(() => {
  switch (aggregateStatus.value) {
    case 'ok': return 'rgb(var(--v-theme-success))';
    case 'warning': return 'rgb(var(--v-theme-warning))';
    case 'error': return 'rgb(var(--v-theme-error))';
    default: return 'rgba(var(--v-theme-on-surface), 0.12)';
  }
});
</script>

<style scoped>
.subsystem-health-card__accent {
  height: 3px;
  width: 100%;
  transition: background 0.3s ease;
}

.subsystem-health-card__watermark {
  position: absolute;
  bottom: -12px;
  right: -8px;
  font-size: 96px !important;
  opacity: 0.04;
  pointer-events: none;
  user-select: none;
}

.min-width-0 {
  min-width: 0;
}

.check-row:not(:last-child) {
  border-bottom: 1px solid rgba(var(--v-theme-on-surface), 0.04);
}
</style>
