<template>
  <div class="impact-breakdowns">
    <health-impact-breakdown
        v-for="table in impactBreakdowns"
        :key="table.title"
        v-bind="table"/>
  </div>
</template>

<script setup>
import HealthImpactBreakdown from '@/traits/health/HealthImpactBreakdown.vue';
import {
  useComplianceImpactBreakdown,
  useEquipmentImpactBreakdown,
  useOccupantImpactBreakdown
} from '@/traits/health/impactBreakdown.js';
import {computed} from 'vue';

const {table: occupantBreakdown} = useOccupantImpactBreakdown();
const {table: equipmentBreakdown} = useEquipmentImpactBreakdown();
const {table: complianceBreakdown} = useComplianceImpactBreakdown();

const impactBreakdowns = computed(() => {
  return [
    occupantBreakdown.value,
    equipmentBreakdown.value,
    complianceBreakdown.value,
  ].filter(t => t.totalCount > 0);
})
</script>

<style scoped lang="scss">
.impact-breakdowns {
  display: flex;
  justify-content: stretch;
  flex-wrap: wrap;
  gap: 40px;

  > * {
    flex: 1;
    min-width: 30em;
    max-width: 40em;
  }
}
</style>