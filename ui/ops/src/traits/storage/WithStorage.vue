<template>
  <slot :resource="resource"/>
</template>

<script setup>
import {usePullStorage} from './storage.js';
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

const resource = reactive(usePullStorage(() => props.name, () => props.paused));
</script>
