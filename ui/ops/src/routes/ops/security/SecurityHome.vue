<template>
  <v-container fluid class="pa-0">
    <v-row class="px-3 pt-3 mb-5">
      <h3 class="text-h3 py-2">Security Overview</h3>
    </v-row>
    <content-card class="mb-8 d-flex flex-column py-0 px-0">
      <v-row
          class="d-flex flex-row align-center mt-0 px-6 mx-auto"
          style="position: absolute; width: 100%; z-index: 1; height: 0; top: 25px">
        <v-text-field
            v-show="!hiddenOnMap"
            v-model="search"
            append-inner-icon="mdi-magnify"
            variant="filled"
            hide-details
            label="Search devices"/>
        <v-spacer style="pointer-events: none"/>
        <v-btn-toggle v-if="hasMapView" v-model="viewType" mandatory variant="outlined">
          <v-btn value="list">List View</v-btn>
          <v-btn value="map">Map View</v-btn>
        </v-btn-toggle>
        <v-select
            v-if="formattedFloorList.length > 1"
            v-model="selectedFloor"
            class="ml-4"
            density="compact"
            hide-details
            :items="formattedFloorList"
            label="Floor"
            variant="outlined"
            style="min-width: 100px; width: 100%; max-width: 170px"/>
      </v-row>
      <list-view v-if="viewType === 'list'" :device-names="deviceQuery"/>
      <map-view v-else-if="hasMapView" :device-names="deviceNames" :floor="selectedFloor"/>
    </content-card>
  </v-container>
</template>

<script setup>
import ContentCard from '@/components/ContentCard.vue';

import {useDevices, useDeviceFloorList} from '@/composables/devices';
import ListView from '@/routes/ops/security/components/ListView.vue';
import MapView from '@/routes/ops/security/components/MapView.vue';
import {useUiConfigStore} from '@/stores/uiConfig.js';
import {storeToRefs} from 'pinia';
import {computed, ref, watch} from 'vue';

const props = defineProps({
  subsystem: {
    type: String,
    default: 'acs'
  },
  filter: {
    type: Function,
    default: () => true
  }
});

const {config} = storeToRefs(useUiConfigStore());

const viewType = ref('list');
const hiddenOnMap = ref(false);
const search = ref('');
const selectedFloor = ref('All');
const useDevicesOpts = computed(() => {
  return {
    subsystem: props.subsystem,
    floor: selectedFloor.value,
    filter: props.filter,
    wantCount: -1
  };
});
const {items: devicesData} = useDevices(useDevicesOpts);
const {floorList} = useDeviceFloorList();

const deviceNames = computed(() => {
  return devicesData.value.map((device) => {
    return {
      source: device.metadata.name,
      name: device.name,
      title: device.metadata?.appearance ? device.metadata?.appearance.title : device.metadata.name.split('/').at(-1),
      traits: device.metadata?.traitsList ? device.metadata?.traitsList.map((trait) => trait.name) : []
    };
  });
});

const hasMapView = computed(() =>
  Array.isArray(config.value.siteFloorPlans) &&
  config.value.siteFloorPlans.some((fp) => fp.svgPath || fp.bgSvgPath)
);

const formattedFloorList = computed(() => {
  if (viewType.value === 'list') return floorList.value;
  else return (config.value.siteFloorPlans ?? []).map((floor) => floor.name);
});

const deviceQuery = computed(() => {
  if (search.value.toLowerCase()) {
    return deviceNames.value.filter((device) => {
      return (
        device.name.toLowerCase().includes(search.value.toLowerCase()) ||
        device.title.toLowerCase().includes(search.value.toLowerCase()) ||
        device.source.toLowerCase().includes(search.value.toLowerCase())
      );
    });
  } else {
    return deviceNames.value;
  }
});

// Remove search when switching to map view
watch(
    viewType,
    (newVal) => {
      if (newVal === 'map') {
        selectedFloor.value = formattedFloorList.value[0] ?? 'Ground Floor';
        hiddenOnMap.value = true;
      } else {
        selectedFloor.value = 'All';
        hiddenOnMap.value = false;
      }
    },
    {immediate: true}
);

// If map view becomes unavailable, fall back to list view
watch(hasMapView, (available) => {
  if (!available) viewType.value = 'list';
});
</script>
