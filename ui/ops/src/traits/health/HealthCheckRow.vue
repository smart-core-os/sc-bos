<template>
  <tr>
    <td class="check--name"><span>{{ name }}</span> <span class="opacity-60">{{ description }}</span></td>
    <td>
      <normality-time-text :model-value="props.modelValue"/>
    </td>
    <td>
      <reliability-time-text :model-value="props.modelValue"/>
    </td>
    <td>
      <template v-if="hasBounds">
        <bounds-text :model-value="props.modelValue" class="ml-1"/>
        <deviation-text :model-value="props.modelValue" class="ml-2"/>
      </template>
      <faults-text v-else-if="hasFaults" :model-value="props.modelValue"/>
      <template v-else-if="lastErrorSummary">
        <div class="text-caption text-medium-emphasis">{{ lastErrorSummary }}</div>
        <div v-if="lastErrorDetails" class="text-caption text-medium-emphasis mt-1">{{ lastErrorDetails }}</div>
      </template>
    </td>
    <td>
      <impacts-text :model-value="props.modelValue"/>
    </td>
  </tr>
</template>

<script setup>
import BoundsText from '@/traits/health/BoundsText.vue';
import DeviationText from '@/traits/health/DeviationText.vue';
import FaultsText from '@/traits/health/FaultsText.vue';
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
const hasFaults = computed(() => (props.modelValue?.faults?.currentFaultsList?.length ?? 0) > 0);

const lastErrorSummary = computed(() => {
  const rel = props.modelValue?.reliability;
  if (!rel || rel.state === HealthCheck.Reliability.State.RELIABLE) return null;
  return rel.lastError?.summaryText ?? null;
});

const lastErrorDetails = computed(() => {
  const rel = props.modelValue?.reliability;
  if (!rel || rel.state === HealthCheck.Reliability.State.RELIABLE) return null;
  return rel.lastError?.detailsText ?? null;
});
</script>

<style scoped>
.check--name {
  line-height: 1;
}
</style>