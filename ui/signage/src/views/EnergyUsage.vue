<template>
  <div class="card">
    <EnergyComp :demand="usageDiff" :generated="producedDiff" :period="period" :total-range="props.totalRange" id="EnergyComp" :show-period="false" :show-energy="false" :run-demo="false"/>
  </div>
</template>

<script setup>
import EnergyComp from '@/components/EnergyMeter.vue';
import {generatedBetween, useMeterDelta, useMeterWindow} from '@/composables/meterDelta.js';
import {isNullOrUndef} from '@/util/types.js';
import {computed} from 'vue';

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

// Both meters are measured over one shared window, so net consumption is a like-for-like
// subtraction.
const meterWindow = useMeterWindow({
  period: () => props.period,
  offset: () => props.offset,
  refreshInterval: () => props.refreshInterval
});

const {startReading, endReading} = useMeterDelta(() => props.name, {window: meterWindow});
const {startReading: generatedAtStart, endReading: generatedAtEnd} =
    useMeterDelta(() => props.generated, {window: meterWindow});

const usageDiff = computed(() => {
  const start = startReading.value;
  const end = endReading.value;
  if (isNullOrUndef(start) || isNullOrUndef(end)) {
    return null;
  }
  return end.usage - start.usage;
});

const producedDiff = computed(() => generatedBetween(generatedAtStart.value, generatedAtEnd.value));

</script>

<style lang="scss" scoped>
.card {
  @include card;
}
</style>
