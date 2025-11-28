<template>
  <span>
    <reliability-icon :model-value="props.modelValue" class="mr-1"/>
    <reliability-text :model-value="props.modelValue"/> since <relative-time :time="time" short/>
  </span>
</template>

<script setup>
import {timestampToDate} from '@/api/convpb.js';
import RelativeTime from '@/components/RelativeTime.vue';
import ReliabilityIcon from '@/traits/health/ReliabilityIcon.vue';
import ReliabilityText from '@/traits/health/ReliabilityText.vue';
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
  switch (props.modelValue?.reliability?.state ?? 0) {
    case HealthCheck.Reliability.State.RELIABLE:
      return timestampToDate(props.modelValue?.reliability?.reliableTime);
    default:
      return timestampToDate(props.modelValue?.reliability?.unreliableTime ?? new Date());
  }
});
</script>

<style scoped>

</style>

