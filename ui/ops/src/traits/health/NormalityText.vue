<template>
  <span>{{ normalityStr }}</span>
</template>

<script setup>
import {normalityToString} from '@/api/sc/traits/health.js';
import {computed} from 'vue';

const props = defineProps({
  modelValue: {
    /** @type {import('vue').PropType<import('@smart-core-os/sc-bos-ui-gen/proto/smartcore/bos/health/v1/health_pb').HealthCheck.AsObject>} */
    type: Object,
    default: null
  }
})

const normalityStr = computed(() => normalityToString(props.modelValue?.normality ?? 0));
</script>

<style scoped>

</style>