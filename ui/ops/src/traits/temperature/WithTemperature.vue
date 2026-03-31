<template>
  <div>
    <slot :resource="temperatureResource" :update="doUpdateTemperature" :update-tracker="updateTracker"/>
  </div>
</template>

<script setup>
import {usePullTemperature, useUpdateTemperature} from '@/traits/temperature/temperature.js';
import {reactive} from 'vue';

const props = defineProps({
  // unique name of the device
  name: {
    type: String,
    default: ''
  },
  paused: {
    type: Boolean,
    default: false
  }
});

const temperatureResource = reactive(usePullTemperature(() => props.name, () => props.paused));
const updateTracker = reactive(useUpdateTemperature(() => props.name));
const doUpdateTemperature = updateTracker.updateTemperature;
</script>

