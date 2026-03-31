<template>
  <RouterView/>
</template>
<script setup>
import {useUiConfigStore} from '@/stores/ui-config.js';
import {onBeforeMount} from 'vue';
import {RouterView} from 'vue-router';

const uiConfig = useUiConfigStore();

onBeforeMount(async () => {
  await uiConfig.loadConfig();
});
</script>

<style lang="scss" scoped>
</style>
