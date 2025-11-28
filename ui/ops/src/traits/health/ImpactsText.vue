<template>
  <div class="impacts">
    <div v-if="hasOccupantImpact" class="impact-item">
      <span class="impact-label">Occupant:</span>
      <occupant-impact-text :model-value="props.modelValue"/>
    </div>
    <div v-if="hasEquipmentImpact" class="impact-item">
      <span class="impact-label">Equipment:</span>
      <equipment-impact-text :model-value="props.modelValue"/>
    </div>
    <div v-if="hasComplianceImpact" class="impact-item">
      <span class="impact-label">Compliance:</span>
      <compliance-impacts-list :model-value="props.modelValue"/>
    </div>
    <span v-if="!hasAnyImpact" class="text-medium-emphasis">No impacts</span>
  </div>
</template>

<script setup>
import ComplianceImpactsList from '@/traits/health/ComplianceImpactsList.vue';
import EquipmentImpactText from '@/traits/health/EquipmentImpactText.vue';
import OccupantImpactText from '@/traits/health/OccupantImpactText.vue';
import {HealthCheck} from '@smart-core-os/sc-bos-ui-gen/proto/health_pb';
import {computed} from 'vue';

const props = defineProps({
  modelValue: {
    /** @type {import('vue').PropType<import('@smart-core-os/sc-bos-ui-gen/proto/health_pb').HealthCheck.AsObject>} */
    type: Object,
    default: null
  }
});

const hasOccupantImpact = computed(() => {
  return props.modelValue?.occupantImpact &&
         props.modelValue.occupantImpact !== HealthCheck.OccupantImpact.NO_OCCUPANT_IMPACT;
});

const hasEquipmentImpact = computed(() => {
  return props.modelValue?.equipmentImpact &&
         props.modelValue.equipmentImpact !== HealthCheck.EquipmentImpact.NO_EQUIPMENT_IMPACT;
});

const hasComplianceImpact = computed(() => {
  return props.modelValue?.complianceImpactsList?.length > 0;
});

const hasAnyImpact = computed(() => {
  return hasOccupantImpact.value || hasEquipmentImpact.value || hasComplianceImpact.value;
});
</script>

<style scoped>
.impacts {
  display: flex;
  flex-direction: column;
  gap: 0.25rem;
}

.impact-item {
  display: flex;
  gap: 0.5rem;
  align-items: flex-start;
}

.impact-label {
  font-weight: 500;
  min-width: 5rem;
  opacity: 60%;
}
</style>

