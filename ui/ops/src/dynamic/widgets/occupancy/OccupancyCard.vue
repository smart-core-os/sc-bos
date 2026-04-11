<template>
  <v-card class="d-flex flex-column px-8 py-4" :class="props.class" :style="props.style">
    <v-toolbar v-if="!hideTitle" color="transparent" class="mb-4">
      <v-toolbar-title class="text-h4" style="overflow-wrap: break-word">{{ title }}</v-toolbar-title>
      <template v-if="!hideTotals">
        <occupancy-people-count :people-count="peopleCount" :error="streamError" class="text-h2 align-self-center"/>
      </template>
    </v-toolbar>
    <occupancy-graph class="flex-grow-1" :source="historyName" :span="span" v-bind="$attrs"/>
  </v-card>
</template>

<script setup>
import OccupancyGraph from '@/dynamic/widgets/occupancy/OccupancyGraph.vue';
import {useOccupancy, usePullOccupancy} from '@/traits/occupancy/occupancy.js';
import {computed} from 'vue';
import OccupancyPeopleCount from './OccupancyPeopleCount.vue';

const props = defineProps({
  class: {type: [String, Array, Object], default: null},
  style: {type: [String, Array, Object], default: null},
  source: {
    type: String,
    default: null
  },
  history: {
    type: String,
    default: null
  },
  title: {
    type: String,
    default: 'Occupancy'
  },
  hideTitle: {
    type: Boolean,
    default: false
  },
  hideTotals: {
    type: Boolean,
    default: false
  },
  span: {
    type: Number,
    default: undefined, // in ms
  }
});
defineOptions({inheritAttrs: false});

const sourceName = computed(() => props.source);
const historyName = computed(() => props.history ?? sourceName.value);

const {value, streamError} = usePullOccupancy(sourceName);
const {peopleCount} = useOccupancy(value);
</script>
