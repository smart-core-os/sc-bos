<template>
  <status-alert v-if="props.streamError" icon="mdi-elevator-passenger" :resource="props.streamError"/>

  <transport-chip
      v-else
      variant="text"
      size="30"
      layout="right"
      :current-floor="currentFloor"
      :next-floor="nextFloor"
      :moving-direction="movingDirection"/>
</template>

<script setup>
import StatusAlert from '@/components/StatusAlert.vue';
import {useTransport} from '@/traits/transport/transport.js';
import TransportChip from '@/traits/transport/TransportChip.vue';
import {computed} from 'vue';

const props = defineProps({
  value: {
    type: Object,
    default: () => ({})
  },
  loading: {
    type: Boolean,
    default: false
  },
  streamError: {
    type: Object,
    default: null
  }
});

const {
  actualPosition,
  nextDestination,
  movingDirection
} = useTransport(() => props.value);

const currentFloor = computed(() => actualPosition.value || '');
const nextFloor = computed(() => {
  const next = nextDestination.value;
  return (next && next !== 'N/A') ? next : '';
});
</script>

<style scoped>
</style>

