<template>
  <v-icon :color="iconInfo.color" v-tooltip:bottom="tooltipStr">{{ iconInfo.icon }}</v-icon>
</template>

<script setup>
import {HealthCheck} from '@smart-core-os/sc-bos-ui-gen/proto/health_pb';
import {computed} from 'vue';

const props = defineProps({
  modelValue: {
    /** @type {import('vue').PropType<Array<import('@smart-core-os/sc-bos-ui-gen/proto/health_pb').HealthCheck.AsObject>>} */
    type: Array,
    default: null
  }
});

const totalCount = computed(() => props.modelValue?.length ?? 0);
const abnormalCount = computed(() => props.modelValue?.filter(check => check.normality !== HealthCheck.Normality.NORMAL).length ?? 0);
const hasChecks = computed(() => totalCount.value > 0);
const hasAbnormal = computed(() => abnormalCount.value > 0);

const iconInfo = computed(() => {
  if (!hasChecks.value) {
    return {icon: 'mdi-heart-outline', color: 'grey'};
  }
  const total = totalCount.value;
  const abnormal = abnormalCount.value;
  if (abnormal === 0) {
    return {icon: 'mdi-heart', color: 'success'};
  }
  if (abnormal < total) {
    return {icon: 'mdi-heart-half-full', color: 'error'};
  }
  return {icon: 'mdi-heart-broken', color: 'error'};
});
const tooltipStr = computed(() => {
  if (!hasChecks.value) {
    return 'No health checks are set up for this device';
  }
  const checkPlural = totalCount.value === 1 ? 'check is' : 'checks are';
  if (hasAbnormal.value) {
    return `${abnormalCount.value} of ${totalCount.value} health ${checkPlural} failing`;
  }
  return `All ${totalCount.value} health ${checkPlural} normal`;
});
</script>

<style scoped>

</style>