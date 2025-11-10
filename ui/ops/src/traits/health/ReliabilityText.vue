<template>
  <span>{{ reliabilityStr }}</span>
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

const reliabilityStr = computed(() => {
  switch (props.modelValue?.reliability?.state ?? 0) {
    case HealthCheck.Reliability.State.RELIABLE:
      return 'Reliable';
    case HealthCheck.Reliability.State.UNRELIABLE:
      return 'Unreliable';
    case HealthCheck.Reliability.State.CONN_TRANSIENT_FAILURE:
      return 'Connection Failure';
    case HealthCheck.Reliability.State.SEND_FAILURE:
      return 'Send Failure';
    case HealthCheck.Reliability.State.NO_RESPONSE:
      return 'No Response';
    case HealthCheck.Reliability.State.BAD_RESPONSE:
      return 'Bad Response';
    case HealthCheck.Reliability.State.NOT_FOUND:
      return 'Not Found';
    case HealthCheck.Reliability.State.PERMISSION_DENIED:
      return 'Permission Denied';
    default:
      return 'Unknown';
  }
});
</script>

<style scoped>

</style>