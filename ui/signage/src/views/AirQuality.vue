<template>
  <div class="card">
    <FlowerPetals :air-quality="co2level"
                  :min-value="props.minValue"
                  :max-value="props.maxValue"
                  id="FlowerComp"/>
  </div>
</template>

<script setup>
import FlowerPetals from '@/components/FlowerPetals.vue';
import {useAirQuality, usePullAirQuality} from '@/traits/airQuality/airQuality.js';
import {onMounted, onUnmounted, ref, watch} from 'vue';

const props = defineProps({
  name: {
    type: String,
    required: true,
    default: ''
  },
  minValue: {
    type: Number,
    default: 400
  },
  maxValue: {
    type: Number,
    default: 700
  }
});

const paused = ref(false);

const {value: airQuality} = usePullAirQuality(() => props.name);
const {presentMetrics} = useAirQuality(airQuality);
const co2level = ref(400);

watch(presentMetrics, (metrics) => {
  if (Object.hasOwn(metrics, 'carbonDioxideLevel')) {
    co2level.value = metrics['carbonDioxideLevel'].value;
  }
}, {immediate: true, deep: true});

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
