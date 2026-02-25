<template>
  <div class="card">
    <WaterComp :water-unit="net" :max-value="max" id="WaterComp" :show-max-value="false"/>
  </div>
</template>


<script setup>
import WaterComp from '@/components/WaterTank.vue';
import {usePeriod} from '@/composables/time.js';
import {useMeterReadingAt, usePullMeterReading} from '@/traits/meter/meter.js';
import {isNullOrUndef} from '@/util/types.js';
import {computed, effectScope, reactive, ref, toRef, watch} from 'vue';


const props = defineProps({
  name: {
    type: String,
    required: true,
    default: ''
  },
  totalRange: {
    type: Number,
    default: 1000
  },
  period: {
    type: [String],
    default: 'day' // 'minute', 'hour', 'day', 'month', 'year'
  },
  offset: {
    type: [Number, String],
    default: 0 // Used via Math.abs, {period: 'day', offset: 1} means yesterday, and so on
  },
});

const _offset = computed(() => -Math.abs(parseInt(props.offset)));
const {start, end} = usePeriod(toRef(props, 'period'), toRef(props, 'period'), _offset);

// const {response: meterReadingInfo} = useDescribeMeterReading(() => props.name);

// calculate the reading at the start date, which we assume is in the past.
const readingAtStart = useMeterReadingAt(() => props.name, start);

const endIsLive = computed(() => _offset.value === 0);
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
