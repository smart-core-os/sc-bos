<template>
  <v-card class="d-flex flex-column">
    <v-toolbar class="chart-header" color="transparent" v-if="props.title !== ''">
      <v-toolbar-title class="text-h4">{{ props.title }}</v-toolbar-title>
    </v-toolbar>
    <div class="display bold text-h5 align-self-center">
      <div class="value">{{ densityDisplayStr }}</div>
      <div class="unit">{{ _unit }}</div>
    </div>
    <div class="display bold pa-8 text-h5 align-self-center">
      <div class="value">{{ densityThreshold.str }}</div>
      <div class="icon">
        <v-icon>{{ densityThreshold.icon }}</v-icon>
      </div>
    </div>
  </v-card>
</template>
<script setup>
import {useDateScale} from '@/components/charts/date.js';
import {usePeriod} from '@/composables/time.js';
import {useMaxPeopleCount} from '@/dynamic/widgets/occupancy/occupancy.js';
import {useMeterReadingAt} from '@/traits/meter/meter.js';
import {format} from '@/util/number.js';
import {isNull} from '@/util/types.js';
import {useLocalProp} from '@/util/vue.js';
import {computed, onMounted, onUnmounted, ref, toRef, watch} from 'vue';


const props = defineProps({
  title: {
    type: String,
    default: 'Power Density'
  },
  name: {
    type: String, // name of the meter device
    default: ''
  },
  meterUnit: {
    type: String,
    default: 'kW' // TODO(get meter unit from DescribeMeterReading)
  },
  occupancy: {
    type: [
      String, // name of the device
      Object // Occupancy.AsObject
    ],
    default: null
  },
  thresholds: {
    type: Array, // {density: number, str: string, icon: string} ordered by density (kW per day) in ascending order
    default: () => [
      {density: 0.3, str: 'Excellent', icon: 'mdi-leaf'},
      {density: 0.7, str: 'Acceptable', icon: 'mdi-check-circle-outline'},
      {density: 1.5, str: 'Poor', icon: 'mdi-alert-circle-outline'},
      {density: Infinity, str: 'Inefficient', icon: 'mdi-fire-alert'},
    ]
  },
  period: {
    type: [String],
    default: 'day' // 'minute', 'hour', 'day', 'month', 'year'
  },
  offset: {
    type: [Number, String],
    default: 0 // Used via Math.abs, {period: 'day', offset: 1} means yesterday, and so on
  },
  refresh: { // refresh period in ms
    type: Number,
    default: 60000,
  }
});

const _unit = computed(() => `${props.meterUnit} per person`);

// Average occupancy over the period
const _occupancy = ref(0);

const computeOccupancy = () => {
  const _offset = computed(() => -Math.abs(parseInt(props.offset)));
  const {start, end} = usePeriod(toRef(props, 'period'), toRef(props, 'period'), _offset);
  const {pastEdges} = useDateScale(start, end, useLocalProp(toRef(props, 'offset')));

  const maxPeopleCounts = useMaxPeopleCount(toRef(props, 'occupancy'), pastEdges);

  return watch(maxPeopleCounts, (counts) => {
    let length = counts?.length ?? 1;

    _occupancy.value = counts.reduce((acc, el) => {
      acc += el.y;
      return acc;
    }, 0) / length;
  }, {deep: true});
};

const density = ref(0);
const densityDisplayStr = computed(() => {
  return format(density.value);
});

const densityThreshold = computed(() => {
  for (const threshold of props.thresholds) {
    if (density.value <= threshold.density) {
      return threshold;
    }
  }
  return {str: '', icon: 'mdi-check-circle-outline'};
});


const computeDensity = () => {
  if (props.name === '') return;
  const _offset = computed(() => -Math.abs(parseInt(props.offset)));
  const {start, end} = usePeriod(toRef(props, 'period'), toRef(props, 'period'), _offset);

  const after = useMeterReadingAt(() => props.name, end, true);
  const before = useMeterReadingAt(() => props.name, start, true);

  return watch([before, after], () => {
    if (isNull(before?.value) || isNull(after?.value)) return;

    const netConsumed = after.value.usage - before.value.usage;
    const netGenerated = after.value.produced - before.value.produced;

    const net = netConsumed && netGenerated ? netConsumed - netGenerated : netConsumed;

    if (net === null || net === undefined || isNaN(net)) return;

    const lookback = end.value - start.value;
    // lookback is in ms, so scale back down to hours
    const hours = lookback / 1000 / 60 / 60;
    density.value = net / hours / (_occupancy.value === 0 ? 1 : _occupancy.value);
  });
};

let interval;
let stopOccupancyWatch;
let stopDensityWatch;

onMounted(() => {
  stopOccupancyWatch = computeOccupancy();
  stopDensityWatch = computeDensity();
  interval = setInterval(() => {
    // Stop previous watches before creating new ones
    if (stopOccupancyWatch) stopOccupancyWatch();
    if (stopDensityWatch) stopDensityWatch();

    stopOccupancyWatch = computeOccupancy();
    stopDensityWatch = computeDensity();
  }, props.refresh);
});

onUnmounted(() => {
  if (interval) clearInterval(interval);
  if (stopOccupancyWatch) stopOccupancyWatch();
  if (stopDensityWatch) stopDensityWatch();
});

</script>

<style scoped>
.display {
  text-align: center;
}

.value {
  font-size: 1.7em;
}

.unit {
  font-size: 1.1em;
}
</style>