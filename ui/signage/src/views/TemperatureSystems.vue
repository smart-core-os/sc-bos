<template>
  <div class="card">
    <TemperatureComp
        :temperature="temperature"
        :target-temperature="targetTemperature"
        :energy="energy"
        :show-energy="showEnergy"
        :total-range="props.totalRange"
        id="TemperatureComp"/>
  </div>
</template>

<script setup>
import TemperatureComp from '@/components/TemperatureMeter.vue';
import {useAirTemperature, usePullAirTemperature} from '@/traits/airTemperature/airTemperature.js';
import {usePullMeterReading} from '@/traits/meter/meter.js';
import {onMounted, onUnmounted, ref, watch} from 'vue';

const props = defineProps({
  name: {
    type: String,
    required: true,
    default: ''
  },
  totalRange: {
    type: Number,
    default: 25
  }
});

const paused = ref(false);
const temperature = ref(0);
const targetTemperature = ref(0);
const energy = ref(0);

const {value: temperatureReading} = usePullAirTemperature(() => props.name, () => paused.value);
const {value: meterReading} = usePullMeterReading(() => props.name, () => paused.value);

const showEnergy = false;

watch(() => temperatureReading, (reading) => {
  const {temp, setPoint} = useAirTemperature(reading);
  temperature.value = temp?.value ?? 0;
  targetTemperature.value = setPoint?.value ?? 22;
}, {deep: true, immediate: true});

watch(() => meterReading, (reading) => {
  energy.value = reading?.value?.usage ?? 0;
}, {deep: true, immediate: true});

onMounted(() => {
  paused.value = false;
});

onUnmounted(() => {
  paused.value = true;
});

</script>

<style lang="scss" scoped>
.card {
  @include card;
}
</style>
