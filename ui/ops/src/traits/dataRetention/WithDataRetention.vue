<template>
  <slot :resource="resource"/>
</template>

<script setup>
import {usePullDataRetention} from './data-retention.js';
import {reactive} from 'vue';

const props = defineProps({
  name: {
    type: String,
    default: ''
  },
  paused: {
    type: Boolean,
    default: false
  }
});

const resource = reactive(usePullDataRetention(() => props.name, () => props.paused));
</script>
