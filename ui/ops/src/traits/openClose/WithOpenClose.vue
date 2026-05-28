<template>
  <div>
    <slot :resource="openCloseValue" :info="info" :update="doUpdateOpenClose" :update-tracker="updateTracker"/>
  </div>
</template>

<script setup>
import {useDescribeOpenClose, usePullOpenClose, useUpdateOpenClose} from '@/traits/openClose/openClose.js';
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

const openCloseValue = reactive(usePullOpenClose(() => props.request ?? props.name, () => props.paused));
const info = reactive(useDescribeOpenClose(() => props.name));
const updateTracker = reactive(useUpdateOpenClose(() => props.name));
const doUpdateOpenClose = updateTracker.updateOpenClose;
</script>
