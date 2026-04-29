<template>
  <div>
    <slot :resource="openCloseValue" :info="info" :update="doUpdatePositions" :update-tracker="updateTracker"/>
  </div>
</template>

<script setup>
import {useDescribePositions, usePullOpenClosePositions, useUpdateOpenClosePositions} from '@/traits/openClose/openClose.js';
import {reactive} from 'vue';

const props = defineProps({
  // unique name of the device
  name: {
    type: String,
    default: ''
  },
  request: {
    type: Object, // of type PullOpenClosePositionsRequest.AsObject
    default: () => {}
  },
  paused: {
    type: Boolean,
    default: false
  }
});

const openCloseValue = reactive(usePullOpenClosePositions(() => props.request ?? props.name, () => props.paused));
const info = reactive(useDescribePositions(() => props.name));
const updateTracker = reactive(useUpdateOpenClosePositions(() => props.name));
const doUpdatePositions = updateTracker.updatePositions;
</script>
