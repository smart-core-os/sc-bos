<template>
  <div class="card">
    <SolarArray :generated="generated" :unit="unit" :period="period" :total-range="props.totalRange"
                id="SolarArrayComp" :show-period="false" :run-demo="false"/>
  </div>
</template>

<script setup>
import SolarArray from '@/components/SolarArray.vue';
import {generatedBetween, useMeterDelta} from '@/composables/meterDelta.js';
import {useDescribeMeterReading} from '@/traits/meter/meter.js';
import {computed} from 'vue';

const props = defineProps({
  // Either a dedicated generation meter or a zone meterGroup, conventionally <zone>/generated.
  name: {
    type: String,
    required: true,
    default: ''
  },
  totalRange: {
    type: Number,
    default: 300
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
  // Defaults true so the figure reads as "generated today", counting from midnight.
  alignToPeriod: {
    type: Boolean,
    default: true
  }
});

const {startReading, endReading} = useMeterDelta(() => props.name, {
  period: () => props.period,
  offset: () => props.offset,
  refreshInterval: () => props.refreshInterval,
  alignToPeriod: () => props.alignToPeriod
});

const generated = computed(() => generatedBetween(startReading.value, endReading.value));

const {response: meterInfo} = useDescribeMeterReading(() => props.name || null);
const unit = computed(() => meterInfo.value?.usageUnit || 'kWh');
</script>

<style lang="scss" scoped>
.card {
  @include card;
}
</style>
