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
    <v-progress-linear
        height="34"
        class="mx-4 my-2"
        :model-value="tempProgress"
        bg-color="neutral-lighten-1"
        bg-opacity="1"
        color="accent"/>
    <v-card-actions class="px-4">
      <v-spacer/>
      <v-btn
          size="small"
          variant="tonal"
          @click="changeSetPoint(-0.5)"
          :disabled="blockActions || (props.value?.temperatureSetPoint === undefined)">
        Down
      </v-btn>
      <v-btn
          size="small"
          variant="tonal"
          @click="changeSetPoint(0.5)"
          :disabled="blockActions || (props.value?.temperatureSetPoint === undefined)">
        Up
      </v-btn>
    </v-card-actions>
    <v-progress-linear color="primary" indeterminate :active="updateValue.loading"/>
  </v-card>
</template>

<script setup>
import {newActionTracker} from '@/api/resource';
import useAuthSetup from '@/composables/useAuthSetup';
import {useAirTemperature} from '@/traits/airTemperature/airTemperature.js';
import {camelToSentence} from '@/util/string';
import {reactive} from 'vue';

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
  tempProgress,
  airTempData
} = useAirTemperature(() => props.value);

const updateValue = reactive(newActionTracker());

/**
 * @param {number} value
 */
function changeSetPoint(value) {
  if (props.value?.temperatureSetPoint?.valueCelsius !== undefined) {
    emit('updateAirTemperature', props.value.temperatureSetPoint.valueCelsius + value);
  }
}

</script>

<style scoped>
.v-list-item {
  min-height: auto;
}

.v-progress-linear {
  width: auto;
}
</style>
