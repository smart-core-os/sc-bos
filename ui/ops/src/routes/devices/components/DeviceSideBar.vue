<template>
  <side-bar>
    <device-side-bar-content :device-id="deviceId" :traits="traits" :health-checks="healthChecks"/>
  </side-bar>
</template>

<script setup>
import SideBar from '@/components/SideBar.vue';
import DeviceSideBarContent from '@/routes/devices/components/DeviceSideBarContent.vue';
import {useSidebarStore} from '@/stores/sidebar';
import {computed} from 'vue';

const sidebar = useSidebarStore();

const deviceId = computed(() => {
  return sidebar.data?.name ?? '';
});

const traits = computed(() => {
  const traits = {};
  if (sidebar.data?.metadata?.traitsList) {
    // flatten array of trait objects (e.g. [{name: 'trait1', ...}, ...] into object (e.g. {trait1: true, ...})
    sidebar.data.metadata.traitsList.forEach((trait) => traits[trait.name] = true);
  }
  return traits;
});

const healthChecks = computed(() => {
  return sidebar.data?.healthChecksList ?? [];
});

</script>

<style scoped>
</style>
