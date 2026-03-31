<template>
  <div>
    <slot :resource="allocationValue"/>
  </div>
</template>

<script setup>
import {usePullAllocation} from '@/traits/allocation/allocation.js';
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

const allocationValue = reactive(usePullAllocation(() => props.name, () => props.paused));
</script>