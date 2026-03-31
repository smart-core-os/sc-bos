<template>
  <div class="card">
    <BuildingComp :occupants="occupants" :max-value="maxValue" id="BuildingComp"/>
  </div>
</template>

<script setup>
import BuildingComp from '@/components/BuildingLights.vue';

import {useOccupancy, usePullOccupancy} from '@/traits/occupancy/occupancy.js';
import {onMounted, onUnmounted, ref, watch} from 'vue';


const props = defineProps({
  name: {
    type: String,
    required: true,
    default: ''
  },
  maxValue: {
    type: Number,
    default: 1250
  }
});

const paused = ref(false);
const occupants = ref(0);


const {value: occ} = usePullOccupancy(() => props.name, () => paused.value);

watch(occ, (occupancy) => {
  const {peopleCount} = useOccupancy(occupancy);
  occupants.value = peopleCount.value;
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
