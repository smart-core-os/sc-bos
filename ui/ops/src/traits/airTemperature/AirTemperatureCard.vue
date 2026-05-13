<template>
  <v-card elevation="0" tile>
    <v-list tile class="ma-0 pa-0">
      <v-list-subheader class="text-title-caps-large text-neutral-lighten-3">Air Temperature</v-list-subheader>
      <v-list-item v-for="(val, key) of airTempData" :key="key" class="py-1">
        <v-list-item-title class="text-body-small text-capitalize">{{ camelToSentence(key) }}</v-list-item-title>
        <template #append>
          <v-list-item-subtitle class="text-capitalize font-weight-medium text-body-1">{{ val }}</v-list-item-subtitle>
        </template>
      </v-list-item>
    </v-list>
    <v-slider
        class="mx-4 my-2"
        color="accent"
        track-color="neutral-lighten-1"
        :disabled="blockActions || setPoint === undefined"
        hide-details
        :min="tempRange.low"
        :max="tempRange.high"
        :step="0.5"
        v-model="sliderSetPoint">
      <template #prepend>
        <v-icon>mdi-thermometer</v-icon>
      </template>
      <template #append>
        <span class="text-body-1 slider-label">{{ sliderSetPointStr }}</span>
      </template>
    </v-slider>
    <v-progress-linear color="primary" indeterminate :active="updateValue.loading"/>
  </v-card>
</template>

<script setup>
import {newActionTracker} from '@/api/resource';
import useAuthSetup from '@/composables/useAuthSetup';
import {useAirTemperature} from '@/traits/airTemperature/airTemperature.js';
import {camelToSentence} from '@/util/string';
import debounce from 'debounce';
import {computed, reactive, ref, watch} from 'vue';

const {blockActions} = useAuthSetup();

const props = defineProps({
  value: {
    type: Object, // of AirTemperature.AsObject
    default: () => ({})
  },
  loading: {
    type: Boolean,
    default: false
  }
});
const emit = defineEmits([
  'updateAirTemperature' // of number | AirTemperature.AsObject | UpdateAirTemperatureRequest.AsObject
]);

const {
  setPoint,
  tempRange,
  airTempData
} = useAirTemperature(() => props.value);

const updateValue = reactive(newActionTracker());

// Local ref for optimistic slider updates — syncs from server but not while user is dragging
const localSetPoint = ref(setPoint.value ?? tempRange.value.low);
watch(setPoint, (v) => { if (v !== undefined) localSetPoint.value = v; }, {immediate: true});

const emitSetPointDebounced = debounce((v) => emit('updateAirTemperature', v), 200);

const sliderSetPoint = computed({
  get: () => localSetPoint.value,
  set: (v) => {
    localSetPoint.value = v;
    emitSetPointDebounced(v);
  }
});

const sliderSetPointStr = computed(() => {
  return setPoint.value !== undefined ? localSetPoint.value.toFixed(1) + '°C' : '-';
});

</script>

<style scoped>
.v-list-item {
  min-height: auto;
}

.slider-label {
  min-width: 4em;
  text-align: right;
}
</style>
