<template>
  <slot :resource="resource"/>
</template>

<script setup>
import {usePullResourceUse} from '@/traits/resourceUse/resourceUse.js';
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

const resource = reactive(usePullResourceUse(() => props.name, () => props.paused));
</script>
