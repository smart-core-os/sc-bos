<template>
  <v-chip
      v-if="label"
      :color="color"
      size="x-small"
      variant="tonal"
      label>
    {{ label }}
  </v-chip>
</template>

<script setup>
import {HealthCheck} from '@smart-core-os/sc-bos-ui-gen/proto/smartcore/bos/health/v1/health_pb';
import {computed} from 'vue';

const props = defineProps({
  modelValue: {
    /** @type {import('vue').PropType<import('@smart-core-os/sc-bos-ui-gen/proto/smartcore/bos/health/v1/health_pb').HealthCheck.AsObject>} */
    type: Object,
    default: null
  }
});

const label = computed(() => {
  switch (props.modelValue?.deviation ?? 0) {
    case HealthCheck.Deviation.MINOR:
      return 'Minor';
    case HealthCheck.Deviation.MODERATE:
      return 'Moderate';
    case HealthCheck.Deviation.MAJOR:
      return 'Major';
    default:
      return '';
  }
});

const color = computed(() => {
  switch (props.modelValue?.deviation ?? 0) {
    case HealthCheck.Deviation.MINOR:
      return 'info';
    case HealthCheck.Deviation.MODERATE:
      return 'warning';
    case HealthCheck.Deviation.MAJOR:
      return 'error';
    default:
      return undefined;
  }
});
</script>

<style scoped>

</style>
