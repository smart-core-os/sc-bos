<template>
  <normality-time-text :model-value="summaryCheck"/>
</template>

<script setup>
import {timestampToDate} from '@/api/convpb.js';
import NormalityTimeText from '@/traits/health/NormalityTimeText.vue';
import {HealthCheck} from '@smart-core-os/sc-bos-ui-gen/proto/health_pb';
import {computed} from 'vue';

const props = defineProps({
  modelValue: {
    /** @type {import('vue').PropType<import('@smart-core-os/sc-bos-ui-gen/proto/devices_pb').Device.AsObject>} */
    type: Object,
    default: null
  },
});

const newestNormalCheck = computed(() => {
  const checks = props.modelValue?.healthChecksList ?? [];
  return checks
    .filter(check => check.normality === HealthCheck.Normality.NORMAL)
    .reduce((newest, check) => {
      if (!newest) return check;
      const checkTime = timestampToDate(check.normalTime);
      const newestTime = timestampToDate(newest.normalTime);
      return checkTime > newestTime ? check : newest;
    }, null);
});
const oldestAbnormalCheck = computed(() => {
  const checks = props.modelValue?.healthChecksList ?? [];
  return checks
    .filter(check => check.normality !== HealthCheck.Normality.NORMAL)
    .reduce((oldest, check) => {
      if (!oldest) return check;
      const checkTime = timestampToDate(check.abnormalTime);
      const oldestTime = timestampToDate(oldest.abnormalTime);
      return checkTime < oldestTime ? check : oldest;
    }, null);
});

const summaryCheck = computed(() => {
  const abnormal = oldestAbnormalCheck.value;
  if (abnormal) {
    return abnormal;
  }
  return newestNormalCheck.value;
});
</script>

<style scoped>

</style>