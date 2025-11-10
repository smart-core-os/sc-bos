<template>
  <reliability-time-text :model-value="summaryCheck"/>
</template>

<script setup>
import {timestampToDate} from '@/api/convpb.js';
import ReliabilityTimeText from '@/traits/health/ReliabilityTimeText.vue';
import {HealthCheck} from '@smart-core-os/sc-bos-ui-gen/proto/health_pb';
import {computed} from 'vue';

const props = defineProps({
  modelValue: {
    /** @type {import('vue').PropType<import('@smart-core-os/sc-bos-ui-gen/proto/devices_pb').Device.AsObject>} */
    type: Object,
    default: null
  },
});

const newestReliableCheck = computed(() => {
  const checks = props.modelValue?.healthChecksList ?? [];
  return checks
    .filter(check => check.reliability?.state === HealthCheck.Reliability.State.RELIABLE)
    .reduce((newest, check) => {
      if (!newest) return check;
      const checkTime = timestampToDate(check.reliability?.reliableTime);
      const newestTime = timestampToDate(newest.reliability?.reliableTime);
      return checkTime > newestTime ? check : newest;
    }, null);
});

const oldestUnreliableCheck = computed(() => {
  const checks = props.modelValue?.healthChecksList ?? [];
  return checks
    .filter(check => check.reliability?.state !== HealthCheck.Reliability.State.RELIABLE)
    .reduce((oldest, check) => {
      if (!oldest) return check;
      const checkTime = timestampToDate(check.reliability?.unreliableTime);
      const oldestTime = timestampToDate(oldest.reliability?.unreliableTime);
      return checkTime < oldestTime ? check : oldest;
    }, null);
});

const summaryCheck = computed(() => {
  const unreliable = oldestUnreliableCheck.value;
  if (unreliable) {
    return unreliable;
  }
  return newestReliableCheck.value;
});
</script>

<style scoped>

</style>

