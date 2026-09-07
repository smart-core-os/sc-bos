<template>
  <v-card elevation="0">
    <v-card-text class="text-body-1 py-0 my-2">{{ name }}</v-card-text>
    <v-card-text class="text-body-2 py-0 my-2">{{ description }}</v-card-text>
    <v-card-text class="py-0">
      <normality-time-text :model-value="props.modelValue"/>
    </v-card-text>
    <v-card-text>
      <reliability-time-text :model-value="props.modelValue"/>
    </v-card-text>
    <v-card-text v-if="currentFaults.length > 0" class="py-0">
      <h4 class="text-caption">{{ currentFaults.length === 1 ? 'Fault:' : 'Faults:' }}</h4>
      <div v-for="fault in currentFaults" :key="fault.code?.code ?? fault.summaryText" class="ml-1">
        <span>{{ fault.summaryText }}</span>
        <div v-if="fault.detailsText" class="ml-1 text-caption text-medium-emphasis">{{ fault.detailsText }}</div>
      </div>
    </v-card-text>
    <v-card-text v-if="errorSummary" class="py-0">
      <h4 class="text-caption">Error:</h4>
      <span class="ml-1">{{ errorSummary }}</span>
      <div v-if="errorDetails" class="ml-1 text-caption text-medium-emphasis">{{ errorDetails }}</div>
    </v-card-text>
    <v-card-text v-if="hasBounds">
      <h4 class="text-caption">Measured value:</h4>
      <bounds-text :model-value="props.modelValue" class="ml-1"/>
    </v-card-text>
    <v-card-text>
      <h4 class="text-caption">Potential impact:</h4>
      <impacts-text :model-value="props.modelValue" class="ml-1"/>
    </v-card-text>
  </v-card>
</template>

<script setup>
import BoundsText from '@/traits/health/BoundsText.vue';
import ImpactsText from '@/traits/health/ImpactsText.vue';
import NormalityTimeText from '@/traits/health/NormalityTimeText.vue';
import ReliabilityTimeText from '@/traits/health/ReliabilityTimeText.vue';
import {HealthCheck} from '@smart-core-os/sc-bos-ui-gen/proto/smartcore/bos/health/v1/health_pb';
import {computed} from 'vue';

const props = defineProps({
  modelValue: {
    /** @type {import('vue').PropType<import('@smart-core-os/sc-bos-ui-gen/proto/smartcore/bos/health/v1/health_pb').HealthCheck.AsObject>} */
    type: Object,
    default: null
  }
});

const name = computed(() => props.modelValue?.displayName ?? props.modelValue?.id ?? '-');
const description = computed(() => props.modelValue?.description);

const hasBounds = computed(() => Boolean(props.modelValue?.bounds));
const currentFaults = computed(() => props.modelValue?.faults?.currentFaultsList ?? []);
// lastError is a record of the most recent error rather than a current one: healthpb keeps it
// on recovery, deliberately, so it can sit alongside reliableTime. Reading it unguarded shows
// an error banner on a check that is reporting fine. HealthCheckRow and SubsystemHealthCard
// both suppress it once the state is RELIABLE; do the same here.
const lastError = computed(() => {
  const rel = props.modelValue?.reliability;
  if (!rel || rel.state === HealthCheck.Reliability.State.RELIABLE) return null;
  return rel.lastError ?? null;
});
const errorSummary = computed(() => lastError.value?.summaryText ?? null);
const errorDetails = computed(() => lastError.value?.detailsText ?? null);
</script>

<style scoped>
.v-card-text {
  padding-top: 0;
  padding-bottom: 0;
  margin-top: 8px;
  margin-bottom: 8px;
}
</style>