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
});

const iconName = computed(() => {
  switch (props.modelValue?.reliability?.state ?? 0) {
    case HealthCheck.Reliability.State.RELIABLE:
      return 'mdi-check-circle';
    case HealthCheck.Reliability.State.UNRELIABLE:
      return 'mdi-alert-circle';
    case HealthCheck.Reliability.State.CONN_TRANSIENT_FAILURE:
      return 'mdi-wifi-off';
    case HealthCheck.Reliability.State.SEND_FAILURE:
      return 'mdi-send-lock';
    case HealthCheck.Reliability.State.NO_RESPONSE:
      return 'mdi-clock-alert';
    case HealthCheck.Reliability.State.BAD_RESPONSE:
      return 'mdi-message-alert';
    case HealthCheck.Reliability.State.NOT_FOUND:
      return 'mdi-file-question';
    case HealthCheck.Reliability.State.PERMISSION_DENIED:
      return 'mdi-lock-alert';
    default:
      return 'mdi-help-circle';
  }
});

const iconColor = computed(() => {
  const state = props.modelValue?.reliability?.state ?? 0;
  if (state === HealthCheck.Reliability.State.RELIABLE) {
    return 'success';
  } else if (state > HealthCheck.Reliability.State.RELIABLE) {
    return 'error';
  } else {
    return 'grey';
  }
});
</script>

<style scoped>

</style>
