<template>
  <span>
    <normality-icon :model-value="props.modelValue" class="mr-1"/>
    <normality-text :model-value="props.modelValue"/> since <relative-time :time="time" short/>
  </span>
</template>

<script setup>
import {timestampToDate} from '@/api/convpb.js';
import RelativeTime from '@/components/RelativeTime.vue';
import NormalityIcon from '@/traits/health/NormalityIcon.vue';
import NormalityText from '@/traits/health/NormalityText.vue';
import {HealthCheck} from '@smart-core-os/sc-bos-ui-gen/proto/health_pb';
import {computed} from 'vue';

const props = defineProps({
  modelValue: {
    /** @type {import('vue').PropType<import('@smart-core-os/sc-bos-ui-gen/proto/health_pb').HealthCheck.AsObject>} */
    type: Object,
    default: null
  },
});

const time = computed(() => {
  switch (props.modelValue?.normality ?? 0) {
    case HealthCheck.Normality.NORMAL:
      return timestampToDate(props.modelValue?.normalTime);
    default:
      return timestampToDate(props.modelValue?.abnormalTime ?? new Date());
  }
});
</script>

<style scoped>

</style>