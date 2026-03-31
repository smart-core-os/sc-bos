<template>
  <v-list class="pa-0" density="compact" nav>
    <v-list-item
        v-for="(item, key) in enabledMenuItems"
        :to="item.link"
        :key="key"
        class="my-2"
        :disabled="hasNoAccess(item.link.path)"
        :prepend-icon="item.icon">
      <v-list-item-title class="text-truncate">{{ item.title }}</v-list-item-title>
    </v-list-item>
  </v-list>
</template>

<script setup>
import useAuthSetup from '@/composables/useAuthSetup';
import {useUiConfigStore} from '@/stores/uiConfig.js';
import {computed} from 'vue';

const {hasNoAccess} = useAuthSetup();

const menuItems = [
  {
    title: 'Zones',
    icon: 'mdi-select-all',
    link: {path: '/system/zones'}
  },
  {
    title: 'Drivers',
    icon: 'mdi-arrow-left-right-bold',
    link: {path: '/system/drivers'}
  },
  {
    title: 'Features',
    icon: 'mdi-tools',
    link: {path: '/system/features'}
  },
  {
    title: 'Components',
    icon: 'mdi-memory',
    link: {path: '/system/components'}
  },
  {
    title: 'Logs',
    icon: 'mdi-text-box-outline',
    link: {path: '/system/logs'}
  }
];

const uiConfig = useUiConfigStore();

const enabledMenuItems = computed(() => {
  return menuItems.filter((item) => uiConfig.pathEnabled(item.link.path));
});
</script>

<style scoped>
:deep(.v-list-item--active) {
  color: rgb(var(--v-theme-primary));
}
</style>
