<template>
  <v-icon :color="iconColor" :icon="iconName"/>
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

const iconName = computed(() => {
  switch (props.modelValue?.normality ?? 0) {
    case HealthCheck.Normality.NORMAL:
      return 'mdi-check-circle';
    case HealthCheck.Normality.ABNORMAL:
      return 'mdi-alert-circle';
    case HealthCheck.Normality.HIGH:
      return 'mdi-arrow-up-circle';
    case HealthCheck.Normality.LOW:
      return 'mdi-arrow-down-circle';
    default:
      return 'mdi-help-circle';
  }
});

const iconColor = computed(() => {
  const normality = props.modelValue?.normality ?? 0;
  if (normality === HealthCheck.Normality.NORMAL) {
    return 'success';
  } else if (normality > HealthCheck.Normality.NORMAL) {
    return 'error';
  } else {
    return 'grey';
  }
});
</script>

<style scoped>

</style>
