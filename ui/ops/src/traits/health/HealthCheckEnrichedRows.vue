<template>
  <health-check-row v-for="check in enrichedChecks" :key="check.id" :model-value="check"/>
</template>

<script setup>
import {usePullHealthChecks} from '@/traits/health/health.js';
import HealthCheckRow from '@/traits/health/HealthCheckRow.vue';
import {computed} from 'vue';

const props = defineProps({
  deviceName: {
    type: String,
    required: true
  },
  healthChecks: {
    /** @type {import('vue').PropType<Array<import('@smart-core-os/sc-bos-ui-gen/proto/health_pb').HealthCheck.AsObject>>} */
    type: Array,
    default: () => []
  }
});

// Pull all health checks for this device to get measured values and live updates
const {value: healthChecksValue} = usePullHealthChecks(() => props.deviceName);

// Computed property that returns enriched checks from the pull stream, or falls back to original checks
const enrichedChecks = computed(() => {
  if (!props.healthChecks) return [];

  // If we have pulled data, use it
  if (healthChecksValue.value && Object.keys(healthChecksValue.value).length > 0) {
    // Convert the resource collection (object keyed by id) to an array
    // and maintain the same order as the original healthChecks
    return props.healthChecks.map(check => {
      if (!check.id) return check;
      return healthChecksValue.value[check.id] || check;
    }).filter(Boolean);
  }

  // Fallback to original checks while loading
  return props.healthChecks;
});
</script>

