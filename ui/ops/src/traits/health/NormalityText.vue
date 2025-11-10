<template>
  <span>{{ normalityStr }}</span>
</template>

<script setup>
import {HealthCheck} from '@smart-core-os/sc-bos-ui-gen/proto/health_pb';
import {computed} from 'vue';

const props = defineProps({
  modelValue: {
    /** @type {import('vue').PropType<import('@smart-core-os/sc-bos-ui-gen/proto/health_pb').HealthCheck.AsObject>} */
    type: Object,
    default: null
  }
})

const normalityStr = computed(() => {
  switch (props.modelValue?.normality ?? 0) {
    case HealthCheck.Normality.NORMAL:
      return 'Normal';
    case HealthCheck.Normality.ABNORMAL:
      return 'Abnormal';
    case HealthCheck.Normality.HIGH:
      return 'High';
    case HealthCheck.Normality.LOW:
      return 'Low';
    default:
      return 'Unknown';
  }
});
</script>

<style scoped>

</style>