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
.v-card-text {
  padding-top: 0;
  padding-bottom: 0;
  margin-top: 8px;
  margin-bottom: 8px;
}
</style>