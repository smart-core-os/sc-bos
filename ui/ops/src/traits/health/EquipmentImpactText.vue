<template>
  <span>{{ equipmentImpactStr }}</span>
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

const equipmentImpactStr = computed(() => {
  switch (props.modelValue?.equipmentImpact ?? 0) {
    case HealthCheck.EquipmentImpact.NO_EQUIPMENT_IMPACT:
      return 'None';
    case HealthCheck.EquipmentImpact.WARRANTY:
      return 'Warranty';
    case HealthCheck.EquipmentImpact.LIFESPAN:
      return 'Lifespan';
    case HealthCheck.EquipmentImpact.FUNCTION:
      return 'Function';
    default:
      return 'Unknown';
  }
});
</script>

<style scoped>

</style>

