<template>
  <v-card>
    <v-toolbar v-if="!props.hideToolbar" color="transparent">
      <v-toolbar-title class="text-h4">{{ props.title }}</v-toolbar-title>
      <v-spacer/>
      <v-chip
          v-if="activeThreshold"
          :color="activeThreshold.color"
          size="small"
          label
          class="flex-shrink-0">
        <v-icon start :icon="activeThreshold.icon"/>
        {{ activeThreshold.label }}
      </v-chip>
    </v-toolbar>
    <v-card-text class="text-h2 px-5 d-flex align-center">
      <occupancy-people-count
          :people-count="peopleCount"
          :max-occupancy="props.maxOccupancy"
          class="justify-space-between flex-grow-1"/>
      <donut-gauge class="gauge" :fill-color="gaugeColor" size="3em" width="15px" min="0" :max="props.maxOccupancy" :value="peopleCount" arc-start="0" arc-end="230"/>
    </v-card-text>
  </v-card>
</template>

<script setup>
import DonutGauge from '@/components/DonutGauge.vue';
import OccupancyPeopleCount from '@/dynamic/widgets/occupancy/OccupancyPeopleCount.vue';
import {resolveOccupancyThreshold} from '@/dynamic/widgets/occupancy/occupancy.js';
import {useOccupancy, usePullOccupancy} from '@/traits/occupancy/occupancy.js';
import {computed, toRef} from 'vue';

const props = defineProps({
  title: {
    type: String,
    default: 'People Count'
  },
  hideToolbar: {
    type: Boolean,
    default: false
  },
  source: {
    type: String,
    default: null
  },
  maxOccupancy: {
    type: Number,
    default: 0
  },
  thresholds: {
    type: Array, // {percentage: number, str: string} ordered by percentage in ascending order
    default: () => [
      {percentage: 10, str: "Quiet"},
      {percentage: 50, str: "Comfortable"},
      {percentage: 70, str: "Busy"},
      {percentage: 90, str: "Full"}
    ]
  }
});

const {value} = usePullOccupancy(toRef(props, 'source'));
const {peopleCount} = useOccupancy(value);

const occupancyPercentage = computed(() => {
  if (!props.maxOccupancy) return 0;
  return (peopleCount.value / props.maxOccupancy) * 100;
});

const activeThreshold = computed(() => resolveOccupancyThreshold(occupancyPercentage.value, props.thresholds));

const gaugeColor = computed(() => {
  if (activeThreshold.value) {
    return activeThreshold.value.color;
  }
  // Default mapping if no thresholds are provided
  if (occupancyPercentage.value >= 90) return 'error-lighten-1';
  if (occupancyPercentage.value >= 70) return 'warning';
  return 'success-lighten-1';
});
</script>

<style scoped>
.v-card-text {
  /* The toolbar has a height of 64px, this aligns with that */
  margin-top: -70px;
}
.gauge {
  margin-left: calc(-1.5em - 15px);
  margin-top: 38px;
}
</style>