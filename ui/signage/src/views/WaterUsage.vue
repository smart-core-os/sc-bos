<template>
  <div class="card">
    <WaterComp :water-unit="net" :max-value="max" id="WaterComp" :show-max-value="false"/>
  </div>
</template>


<script setup>
import WaterComp from '@/components/WaterTank.vue';
import {useInterval} from '@/composables/time.js';
import {useMeterReadingAt, usePullMeterReading} from '@/traits/meter/meter.js';
import {isNullOrUndef} from '@/util/types.js';
import {startOfDay, startOfHour, startOfMinute, startOfMonth, startOfYear, sub} from 'date-fns';
import {computed, effectScope, reactive, ref, watch} from 'vue';


const props = defineProps({
  name: {
    type: String,
    required: true,
    default: ''
  },
  totalRange: {
    type: Number,
    default: 200
  },
  period: {
    type: [String],
    default: 'day' // 'minute', 'hour', 'day', 'month', 'year'
  },
  offset: {
    type: [Number, String],
    default: 0 // Used via Math.abs, {period: 'day', offset: 1} means yesterday, and so on
  },
  refreshInterval: {
    type: Number,
    default: 60 * 1000 // 1 minute in ms; set to 0 to disable
  },
  // When true the window snaps to calendar boundaries for `period` (e.g. from 12am
  // local time for 'day'); when false it is a rolling window of one `period` ending now.
  alignToPeriod: {
    type: Boolean,
    default: false
  }
});

const _offset = computed(() => -Math.abs(parseInt(props.offset)));

// Start of the calendar period, keyed by `period` (e.g. startOfDay -> 12am local time).
const startOfPeriod = {
  minute: startOfMinute,
  hour: startOfHour,
  day: startOfDay,
  month: startOfMonth,
  year: startOfYear,
};

// Tick drives the refresh — the window recomputes each interval.
const tick = useInterval(() => props.refreshInterval);
const endIsLive = computed(() => _offset.value === 0);

// Reference point: `offset` periods before now (offset 0 -> the current period).
const reference = computed(() => { tick.value; return sub(new Date(), {[`${props.period}s`]: -_offset.value}); });

// When aligned, the window is the calendar period containing the reference: it counts
// from that period's start (e.g. 12am today) up to a live "now" for the current period,
// or to the start of the next period for a past one. Otherwise it is a rolling window
// of one `period` ending at the reference.
const start = computed(() => props.alignToPeriod
    ? startOfPeriod[props.period](reference.value)
    : sub(reference.value, {[`${props.period}s`]: 1}));
const end = computed(() => {
  if (endIsLive.value) return reference.value;
  return props.alignToPeriod
      ? startOfPeriod[props.period](sub(reference.value, {[`${props.period}s`]: -1}))
      : reference.value;
});

// const {response: meterReadingInfo} = useDescribeMeterReading(() => props.name);

const readingAtStart = useMeterReadingAt(() => props.name, start);
let endCalcScope = null;
const readingAtEnd = reactive({value: null});
watch(endIsLive, (endIsLive) => {
  if (endCalcScope) {
    endCalcScope();
  }
  const scope = effectScope();
  endCalcScope = () => scope.stop();
  scope.run(() => {
    if (endIsLive) {
      const {value: meterReading} = usePullMeterReading(() => props.name);
      readingAtEnd.value = meterReading;
    } else {
      readingAtEnd.value = useMeterReadingAt(() => props.name, end);
    }
  });
}, {immediate: true});

const usageDiff = computed(() => {
  const start = readingAtStart.value;
  const end = readingAtEnd.value;
  if (isNullOrUndef(start) || isNullOrUndef(end)) {
    return null;
  }
  return {usage: end.usage - start.usage};
});

const producedDiff = computed(() => {
  const start = readingAtStart.value;
  const end = readingAtEnd.value;
  if (isNullOrUndef(start) || isNullOrUndef(end)) {
    return null;
  }
  return {production: end.produced - start.produced};
});

const previousHistory = ref([]);
const net = computed(() => {
  const usage = usageDiff.value ? usageDiff.value.usage : 0;
  const production = producedDiff.value ? producedDiff.value.production : 0;
  if (isNaN(production)) return usage;
  return usage - production;
});

watch(() => net.value, (v) => {
  if (isNaN(v)) return;
  previousHistory.value.push(v);

  if (previousHistory.value.length > 30) {
    previousHistory.value.shift();
  }
}, {immediate: true, deep: true});

const max = computed(() => {
  return Math.max(...previousHistory.value);
});

</script>


<style lang="scss" scoped>
.card {
  @include card;

  :deep(.svg svg .strokeWhite) {
    stroke: $island-card;
  }
}
</style>
