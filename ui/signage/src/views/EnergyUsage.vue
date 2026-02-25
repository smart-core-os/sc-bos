<template>
  <div class="card">
    <EnergyComp :demand="usageDiff" :generated="producedDiff" :period="period" :total-range="props.totalRange" id="EnergyComp" :show-period="false" :show-energy="false" :run-demo="false"/>
  </div>
</template>

<script setup>
import EnergyComp from '@/components/EnergyMeter.vue';
import {usePeriod} from '@/composables/time.js';
import {
  useMeterReadingAt,
  usePullMeterReading
} from '@/traits/meter/meter.js';
import {isNullOrUndef} from '@/util/types.js';
import {computed, effectScope, reactive, toRef, watch} from 'vue';

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
});

const _offset = computed(() => -Math.abs(parseInt(props.offset)));
const {start, end} = usePeriod(toRef(props, 'period'), toRef(props, 'period'), _offset);

// const {response: meterReadingInfo} = useDescribeMeterReading(() => props.name);

// calculate the reading at the start date, which we assume is in the past.
const readingAtStart = useMeterReadingAt(() => props.name, start);
const generatedAtStart = useMeterReadingAt(() => props.generated, start);
const endIsLive = computed(() => _offset.value === 0);
let endCalcScope = null;
const readingAtEnd = reactive({value: null});
const generatedAtEnd = reactive({value: null});
watch(() => endIsLive, (endIsLive) => {
  if (endCalcScope) {
    endCalcScope();
  }
  const scope = effectScope();
  endCalcScope = () => scope.stop();
  scope.run(() => {
    if (endIsLive.value) {
      const {value: meterReading} = usePullMeterReading(() => props.name);
      readingAtEnd.value = meterReading;
      const {value: generatedReading} = usePullMeterReading(() => props.generated);
      generatedAtEnd.value = generatedReading;
    } else {
      readingAtEnd.value = useMeterReadingAt(() => props.name, end);
      generatedAtEnd.value = useMeterReadingAt(() => props.generated, end);
    }
  });
}, {deep: true, immediate: true});

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
