<template>
  <v-card>
    <v-toolbar v-if="!props.hideToolbar" color="transparent">
      <v-toolbar-title class="text-h4">{{ props.title }}</v-toolbar-title>
    </v-toolbar>
    <v-card-text class="text-h1 px-5 d-flex align-center">
      <occupancy-people-count :people-count="peopleCount"
                              :max-occupancy="props.maxOccupancy"
                              :thresholds="props.thresholds"
                              class="justify-space-between flex-grow-1"/>
      <donut-gauge class="gauge" fill-color="primary" size="3em" width="15px" min="0" :max="props.maxOccupancy" :value="peopleCount" arc-start="0" arc-end="230"/>
    </v-card-text>
  </v-card>
</template>

<script setup>
import DonutGauge from '@/components/DonutGauge.vue';
import OccupancyPeopleCount from '@/dynamic/widgets/occupancy/OccupancyPeopleCount.vue';
import {useOccupancy, usePullOccupancy} from '@/traits/occupancy/occupancy.js';
import {toRef} from 'vue';

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
    default: null
  }
});

const {value} = usePullOccupancy(toRef(props, 'source'));
const {peopleCount} = useOccupancy(value);
</script>

<style scoped>
.v-toolbar-title {
  padding-right: 4em;
}
.v-card-text {
  /* The toolbar has a height of 64px, this aligns with that */
  margin-top: -53px;
}
.gauge {
  margin-left: calc(-1.5em - 15px);
}
</style>