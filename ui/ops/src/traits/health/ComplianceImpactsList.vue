<template>
  <div v-if="complianceImpacts.length > 0">
    <div v-for="impact in complianceImpacts" :key="impact.standard?.title" class="compliance-impact">
      <span class="standard-name">{{ impact.standard?.displayName || impact.standard?.title }}</span>
      <span class="contribution" :class="contributionClass(impact.contribution)">
        {{ contributionStr(impact.contribution) }}
      </span>
    </div>
  </div>
  <span v-else class="text-medium-emphasis">None</span>
</template>

<script setup>
import {HealthCheck} from '@vanti-dev/sc-bos-ui-gen/proto/health_pb';
import {computed} from 'vue';

const props = defineProps({
  modelValue: {
    /** @type {import('vue').PropType<import('@vanti-dev/sc-bos-ui-gen/proto/health_pb').HealthCheck.AsObject>} */
    type: Object,
    default: null
  }
});

const complianceImpacts = computed(() => props.modelValue?.complianceImpactsList ?? []);

const contributionStr = (contribution) => {
  switch (contribution) {
    case HealthCheck.ComplianceImpact.Contribution.NOTE:
      return 'Note';
    case HealthCheck.ComplianceImpact.Contribution.RATING:
      return 'Rating';
    case HealthCheck.ComplianceImpact.Contribution.WARNING:
      return 'Warning';
    case HealthCheck.ComplianceImpact.Contribution.FAIL:
      return 'Fail';
    default:
      return 'Unknown';
  }
};

const contributionClass = (contribution) => {
  switch (contribution) {
    case HealthCheck.ComplianceImpact.Contribution.NOTE:
      return 'text-info';
    case HealthCheck.ComplianceImpact.Contribution.RATING:
      return 'text-primary';
    case HealthCheck.ComplianceImpact.Contribution.WARNING:
      return 'text-warning';
    case HealthCheck.ComplianceImpact.Contribution.FAIL:
      return 'text-error';
    default:
      return 'text-medium-emphasis';
  }
};
</script>

<style scoped>
.compliance-impact {
  display: flex;
  justify-content: space-between;
  gap: 0.5rem;
  padding: 0.125rem 0;
}

.standard-name {
  flex: 1;
}

.contribution {
  font-weight: 500;
  font-size: 0.875rem;
}
</style>

