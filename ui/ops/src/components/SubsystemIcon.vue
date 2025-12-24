<template>
  <v-icon>{{ icon }}</v-icon>
</template>
<script setup>
import {computed} from 'vue';

const props = defineProps({
  subsystem: {
    type: String,
    default: ''
  },
  noDefault: {
    type: Boolean,
    default: false
  }
});
const navigationItemIcons = {
  acs: 'mdi-badge-account-horizontal',
  cctv: 'mdi-cctv',
  fire: 'mdi-fire',
  hvac: 'mdi-thermometer',
  lighting: 'mdi-lightbulb',
  lockers: 'mdi-locker-multiple',
  metering: 'mdi-meter-electric',
  network: 'mdi-router-network-wireless',
  sensors: 'mdi-leak', // we might need to change this icon
  smart: 'mdi-memory',
  vt: 'mdi-elevator-passenger',
  zones: 'mdi-select-all'
};
const defaultIcon = 'mdi-chevron-right';

const icon = computed(() => {
  const iconStr = navigationItemIcons[props.subsystem];
  if (iconStr) return iconStr;
  if (props.noDefault) return '';
  return defaultIcon;
});
</script>
