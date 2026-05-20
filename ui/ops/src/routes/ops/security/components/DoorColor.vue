<template>
  <div/>
</template>
<script setup>
import {useStatus} from '@/routes/ops/security/components/access-point-card/useStatus';
import {computed, watch} from 'vue';

const props = defineProps({
  name: {
    type: String,
    default: ''
  },
  accessAttempt: {
    type: Object,
    default: () => ({})
  },
  openClose: {
    type: Object,
    default: () => ({})
  }
});

const emit = defineEmits(['updateFill', 'updateStroke']);

const {color} = useStatus(() => props.accessAttempt);

const setDoorColor = computed(() => {
  if (props.openClose?.statesList?.[0]) {
    const openPercent = props.openClose.statesList[0].openPercent;
    if (openPercent === 0) {
      return 'closed';
    }
    if (openPercent === 100) {
      return 'open';
    }
    if (openPercent > 0 && openPercent < 100) {
      return 'moving';
    }
  }
  return 'unknown';
});

watch(color, (newColor) => {
  if (props.accessAttempt && Object.keys(props.accessAttempt).length) {
    emit('updateFill', {name: props.name, color: newColor});
  }
}, {immediate: true});

watch(setDoorColor, (newColor) => {
  if (props.openClose && Object.keys(props.openClose).length) {
    emit('updateStroke', {name: props.name, color: newColor});
  }
}, {immediate: true});
</script>

