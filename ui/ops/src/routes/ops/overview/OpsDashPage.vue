<template>
  <component :is="pageConfig.layout" v-bind="pageConfig.config" :key="pageKey"/>
</template>

<script setup>
import usePageConfig from '@/routes/ops/overview/pageConfig.js';
import {computed, reactive} from 'vue';

const props = defineProps({
  pathSegments: {
    type: Array,
    required: true
  }
});

const pageConfig = reactive(usePageConfig(() => props.pathSegments));
const pageKey = computed(() => props.pathSegments.join('/'));
</script>
