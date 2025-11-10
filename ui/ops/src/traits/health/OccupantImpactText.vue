<template>
  <span>{{ occupantImpactStr }}</span>
</template>

<script setup>
import {HealthCheck} from '@smart-core-os/sc-bos-ui-gen/proto/health_pb';
import {computed} from 'vue';

const props = defineProps({
  modelValue: {
    /** @type {import('vue').PropType<import('@smart-core-os/sc-bos-ui-gen/proto/health_pb').HealthCheck.AsObject>} */
    type: Object,
    default: null
  }
});

const occupantImpactStr = computed(() => {
  switch (props.modelValue?.occupantImpact ?? 0) {
    case HealthCheck.OccupantImpact.NO_OCCUPANT_IMPACT:
      return 'None';
    case HealthCheck.OccupantImpact.COMFORT:
      return 'Comfort';
    case HealthCheck.OccupantImpact.HEALTH:
      return 'Health';
    case HealthCheck.OccupantImpact.LIFE:
      return 'Life Safety';
    default:
      return 'Unknown';
  }
});
</script>

<style scoped>

</style>

