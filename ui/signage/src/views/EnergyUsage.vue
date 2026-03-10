<template>
  <div class="card">
    <EnergyComp :demand="usageDiff" :generated="producedDiff" :period="period" :total-range="props.totalRange" id="EnergyComp" :show-period="false" :show-energy="false" :run-demo="false"/>
  </div>
</template>

<script setup>
import EnergyComp from '@/components/EnergyMeter.vue';
import {useInterval} from '@/composables/time.js';
import {
  useMeterReadingAt,
  usePullMeterReading
} from '@/traits/meter/meter.js';
import {isNullOrUndef} from '@/util/types.js';
import {sub} from 'date-fns';
import {computed, effectScope, reactive, watch} from 'vue';

const props = defineProps({
  name: {
    type: String,
    required: true,
    default: ''
  },
  generated: {
    type: String,
    required: false,
    default: ''
  },
  totalRange: {
    type: Number,
    default: 300
  },
  period: {
    type: [String],
    default: 'hour' // 'minute', 'hour', 'day', 'month', 'year'
  },
  offset: {
    type: [Number, String],
    default: 0 // Used via Math.abs, {period: 'day', offset: 1} means yesterday, and so on
  },
  refreshInterval: {
    type: Number,
    default: 60 * 1000 // 1 minute in ms; set to 0 to disable
  }
});

const _offset = computed(() => -Math.abs(parseInt(props.offset)));

// Tick drives the rolling window — start and end update each interval
const tick = useInterval(() => props.refreshInterval);
const end = computed(() => { tick.value; return sub(new Date(), {[`${props.period}s`]: -_offset.value}); });
const start = computed(() => sub(end.value, {[`${props.period}s`]: 1}));

// const {response: meterReadingInfo} = useDescribeMeterReading(() => props.name);

const readingAtStart = useMeterReadingAt(() => props.name, start);
const generatedAtStart = useMeterReadingAt(() => props.generated, start);

const endIsLive = computed(() => _offset.value === 0);
let endCalcScope = null;
const readingAtEnd = reactive({value: null});
const generatedAtEnd = reactive({value: null});
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
      const {value: generatedReading} = usePullMeterReading(() => props.generated);
      generatedAtEnd.value = generatedReading;
    } else {
      readingAtEnd.value = useMeterReadingAt(() => props.name, end);
      generatedAtEnd.value = useMeterReadingAt(() => props.generated, end);
    }
  });
}, {immediate: true});

const usageDiff = computed(() => {
  const start = readingAtStart.value;
  const end = readingAtEnd.value;
  if (isNullOrUndef(start) || isNullOrUndef(end)) {
    return null;
  }
  return end.usage - start.usage;
});

const producedDiff = computed(() => {
  const start = generatedAtStart.value;
  const end = generatedAtEnd.value;
  if (isNullOrUndef(start) || isNullOrUndef(end)) {
    return null;
  }
  return end.produced - start.produced;
});

</script>

<style lang="scss" scoped>
.card {
  @include card;
}
</style>
