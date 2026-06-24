<template>
  <side-bar>
    <v-card-subtitle class="mt-4">Layers</v-card-subtitle>
    <v-list :items="layerList" v-model:selected="visibleLayers" selectable select-strategy="independent">
      <template #append="{isSelected}">
        <v-list-item-action>
          <v-switch :model-value="isSelected" readonly hide-details density="compact" style="pointer-events: none"/>
        </v-list-item-action>
      </template>
    </v-list>
  </side-bar>
</template>

<script setup>
import SideBar from '@/components/SideBar.vue';
import {useSidebarStore} from '@/stores/sidebar.js';
import {computed} from 'vue';

const sidebar = useSidebarStore();
const layerList = computed(() => sidebar.data.layerList);
const visibleLayers = computed({
  get: () => sidebar.data.visibleLayers,
  set: (v) => {
    if (sidebar.data.visibleLayers) {
      sidebar.data.visibleLayers = v;
    }
  }
});
</script>

<style scoped>

</style>
