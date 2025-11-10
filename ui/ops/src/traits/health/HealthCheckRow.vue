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
      <bounds-text v-if="hasBounds" :model-value="props.modelValue" class="ml-1"/>
    </td>
    <td>
      <impacts-text :model-value="props.modelValue"/>
    </td>
  </tr>
</template>

<script setup>
import BoundsText from '@/traits/health/BoundsText.vue';
import ImpactsText from '@/traits/health/ImpactsText.vue';
import NormalityTimeText from '@/traits/health/NormalityTimeText.vue';
import ReliabilityTimeText from '@/traits/health/ReliabilityTimeText.vue';
import {computed} from 'vue';

const props = defineProps({
  modelValue: {
    /** @type {import('vue').PropType<import('@smart-core-os/sc-bos-ui-gen/proto/health_pb').HealthCheck.AsObject>} */
    type: Object,
    default: null
  }
});

const name = computed(() => props.modelValue?.displayName ?? props.modelValue?.id ?? '-');
const description = computed(() => props.modelValue?.description);

const hasBounds = computed(() => Boolean(props.modelValue?.bounds));
</script>

<style scoped>
.check--name {
  line-height: 1;
}
</style>