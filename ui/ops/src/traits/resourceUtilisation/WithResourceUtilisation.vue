<template>
  <slot :resource="resource"/>
</template>

<script setup>
import {usePullResourceUtilisation} from '@/traits/resourceUtilisation/resourceUtilisation.js';
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

const resource = reactive(usePullResourceUtilisation(() => props.name, () => props.paused));
</script>
