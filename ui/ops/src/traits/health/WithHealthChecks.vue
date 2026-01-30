<template>
  <div>
    <slot :resource="healthChecksValue"/>
  </div>
</template>

<script setup>
import {usePullHealthChecks} from '@/traits/health/health.js';
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

const healthChecksValue = reactive(usePullHealthChecks(() => props.name, () => props.paused));
</script>

