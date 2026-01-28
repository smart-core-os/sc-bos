<template>
  <v-card class="d-flex flex-column">
    <v-toolbar class="chart-header" color="transparent" v-if="props.title !== ''">
      <v-toolbar-title class="text-h4">{{ props.title }}</v-toolbar-title>
    </v-toolbar>
    <div class="display text-h2 align-self-center">
      <div class="value">{{ densityDisplayStr }}</div>
      <div class="unit">{{ _unit }}</div>
    </div>
  </v-card>
</template>
<script setup>
import {usePeriod} from '@/composables/time.js';
import {useMeterReadingAt} from '@/traits/meter/meter.js';
import {usePullOccupancy} from '@/traits/occupancy/occupancy.js';
import {format} from '@/util/number.js';
import {isNull} from '@/util/types.js';
import {computed, onMounted, onUnmounted, reactive, ref, toRef, watch} from 'vue';


const props = defineProps({
  title: {
    type: String,
    default: 'Meter Density'
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
  period: {
    type: [String],
    default: 'day' // 'minute', 'hour', 'day', 'month', 'year'
  },
  offset: {
    type: [Number, String],
    default: 0 // Used via Math.abs, {period: 'day', offset: 1} means yesterday, and so on
  },
  refresh: {
    type: Number,
    default: 60000,
  }
});

const _unit = computed(() => `${props.meterUnit} per person`);
const _occupancy = reactive(usePullOccupancy(props.occupancy));
const density = ref(0);
const densityDisplayStr = computed(() => {
  return format(density.value);
});


const computeDensity = () => {
  if (props.name === '') return;
  const _offset = computed(() => -Math.abs(parseInt(props.offset)));
  const {start, end} = usePeriod(toRef(props, 'period'), toRef(props, 'period'), _offset);

  const after = useMeterReadingAt(() => props.name, end, true);
  const before = useMeterReadingAt(() => props.name, start, true);

  watch([before, after], () => {
    if (isNull(before?.value) || isNull(after?.value)) return;

    const netConsumed = after.value.usage - before.value.usage;
    const netGenerated = after.value.produced - before.value.produced;

    const net = netConsumed && netGenerated ? netConsumed - netGenerated : netConsumed;

    if (!net || isNaN(net)) return;

    const lookback = end.value.getTime() - start.value.getTime();
    // lookback is in ms, so scale back down to hours
    const hours = lookback / 1000 / 60 / 60;
    density.value = net / hours / (_occupancy.value.peopleCount === 0 ? 1 : _occupancy.value.peopleCount);
  });
};

let interval;
onMounted(() => {
  computeDensity();
  interval = setInterval(() => {
    computeDensity();
  }, props.refresh);
});

onUnmounted(() => {
  clearInterval(interval);
});
</script>

<style scoped>
.display {
  font-weight: lighter;
  text-align: center;
}

.value {
  font-size: 1.7em;
}

.unit {
  font-size: 1.1em;
}
</style>