<template>
  <v-chip
      v-if="deviation > 0"
      :color="color"
      size="x-small"
      variant="tonal"
      label>
    {{ label }}
  </v-chip>
</template>

<script setup>
import {computed} from 'vue';

const props = defineProps({
  modelValue: {
    /** @type {import('vue').PropType<import('@smart-core-os/sc-bos-ui-gen/proto/smartcore/bos/health/v1/health_pb').HealthCheck.AsObject>} */
    type: Object,
    default: null
  }
});

/** @type {import('vue').ComputedRef<number>} deviation as a fraction of the range width */
const deviation = computed(() => props.modelValue?.deviation ?? 0);

const label = computed(() => `${Math.round(deviation.value * 100)}% out`);

// Colour bands are a reading aid only. The wire format carries the measured
// fraction, so where these sit is a presentation choice, not a contract.
const color = computed(() => {
  if (deviation.value >= 0.25) return 'error';
  if (deviation.value >= 0.1) return 'warning';
  return 'info';
});
</script>

<style scoped>

</style>
