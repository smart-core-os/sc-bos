<template>
  <v-card class="d-flex flex-column">
    <v-toolbar v-if="!props.hideToolbar" color="transparent">
      <v-toolbar-title class="text-h4">{{ props.title }}</v-toolbar-title>
    </v-toolbar>
    <power-compare-chart class="content" :total-source="props.totalSource" :sources="props.sources" :metric="props.metric" :unit="props.unit"/>
  </v-card>
</template>

<script setup>
import PowerCompareChart from '@/dynamic/widgets/energy/PowerCompareChart.vue';

const props = defineProps({
  title: {
    type: String,
    default: 'Current Power'
  },
  hideToolbar: {
    type: Boolean,
    default: false
  },
  totalSource: {
    type: String,
    default: undefined,
  },
  metric: {
    type: String,
    default: 'realPower',
  },
  sources: {
    type: Array, // of String or {title: String, name: String}
    default: () => [],
  },
  unit: {
    type: String,
    default: undefined,
  }
});
</script>

<style scoped>
.content {
  margin: 0 8px 16px 8px;
  min-height: 0;
  min-width: 0;
  flex: 1 1 auto;
}
</style>