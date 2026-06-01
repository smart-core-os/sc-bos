<template>
  <div v-if="faults.length > 0" class="fault-list">
    <div v-for="(fault, i) in faults" :key="i" class="fault-item">
      <v-tooltip v-if="fault.detailsText" :text="fault.detailsText" location="top" max-width="400">
        <template #activator="{ props: tipProps }">
          <span v-bind="tipProps" class="text-error fault-summary">
            {{ fault.summaryText || fault.code?.code || 'Fault' }}
          </span>
        </template>
      </v-tooltip>
      <span v-else class="text-error">{{ fault.summaryText || fault.code?.code || 'Fault' }}</span>
    </div>
  </div>
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

const faults = computed(() => props.modelValue?.faults?.currentFaultsList ?? []);
</script>

<style scoped>
.fault-list {
  display: flex;
  flex-direction: column;
  gap: 0.1rem;
}

.fault-summary {
  cursor: help;
  text-decoration: underline dotted;
}
</style>
