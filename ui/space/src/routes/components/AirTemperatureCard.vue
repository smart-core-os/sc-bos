<template>
  <v-card elevation="0">
    <v-card-title class="d-flex align-center pl-7">
      <span class="text-h4 font-weight-medium flex-grow-1">Temperature</span>
      <v-switch
          v-if="hasHvacAutoSwitch"
          color="accent"
          :disabled="blockActions"
          hide-details
          inset
          :model-value="hvacIsAuto"
          @update:model-value="autoMode">
        <template #prepend>
          <span class="text-caption text-uppercase">Auto</span>
        </template>
      </v-switch>
    </v-card-title>
    <v-card-text>
      <v-slider
          track-color="primary"
          track-fill-color="accent"
          :disabled="blockActions"
          hide-details="auto"
          :max="temperatureRange.high"
          :min="temperatureRange.low"
          step="0.1"
          v-model="setPoint">
        <template #prepend>
          <v-icon size="35">mdi-thermometer</v-icon>
        </template>
        <template #append>
          <div class="values d-flex mr-1 align-center">
            <span class="text-h5" style="opacity: .5">{{ currentTemp.toFixed(1) }}</span>
            <v-icon class="mx-1" style="opacity: .5" size="20">mdi-menu-right</v-icon>
            <span class="text-h5">{{ setPoint.toFixed(1) }}&deg;C</span>
          </div>
        </template>
      </v-slider>
    </v-card-text>
    <v-progress-linear :active="updateValue.loading" color="primary" indeterminate/>
  </v-card>
</template>

<script setup>
import {closeResource, newActionTracker, newResourceValue} from '@/api/resource';
import {pullAirTemperature, updateAirTemperature} from '@/api/sc/traits/air-temperature';
import {pullModeValues, updateModeValues} from '@/api/sc/traits/mode';
import useAuthSetup from '@/composables/useAuthSetup';
import {useRoundTrip} from '@/routes/components/useRoundTrip';
import debounce from 'debounce';
import {computed, onUnmounted, reactive, ref, toRef, watch} from 'vue';

const {blockActions} = useAuthSetup();

const temperatureRange = ref({
  low: 19.0,
  high: 25.0
});

const props = defineProps({
  name: {
    type: String,
    default: ''
  }
});

const airTempValue = reactive(newResourceValue());

watch(() => props.name, async (name) => {
  // close existing stream if present
  closeResource(airTempValue);
  // create new stream
  if (name && name !== '') {
    pullAirTemperature({name: name}, airTempValue);
  }
}, {immediate: true});

onUnmounted(() => {
  closeResource(airTempValue);
});

/**
 * Calculates the percentage value of the current temperature based on the temperature range
 *
 * @return {number}
 */
const currentTemp = computed(() => airTempValue.value?.ambientTemperature?.valueCelsius ?? 0);
const {localValue, value} = useRoundTrip(toRef(airTempValue, 'value'));
const setPoint = computed({
  get() {
    if (value.value) {
      return value.value.temperatureSetPoint?.valueCelsius ?? 0;
    }
    return 0;
  },
  set(value) {
    // prevent setting a value before current value has been fetched
    if (airTempValue.value !== null) {
      if (localValue.value?.temperatureSetPoint?.valueCelsius !== value) {
        localValue.value = {
          ...airTempValue.value,
          temperatureSetPoint: {
            valueCelsius: value
          }
        };
      }
      autoMode(false);
      changeSetPointDebounced(value);
    }
  }
});

const updateValue = reactive(newResourceValue());

/**
 * @param {number} value
 */
function changeSetPoint(value) {
  /* @type {UpdateAirTemperatureRequest.AsObject} */
  const req = {
    name: props.name,
    state: {
      temperatureSetPoint: {
        valueCelsius: value
      }
    },
    updateMask: {pathsList: ['temperature_set_point']}
  };

  updateAirTemperature(req, updateValue);
}

const changeSetPointDebounced = debounce((val) => changeSetPoint(val));


const modeValuesResource = reactive(
    /** @type {ResourceValue<ModeValues.AsObject, PullModeValuesResponse.AsObject>} */
    newResourceValue());
const updateModeValuesTracker = reactive(
    /** @type {ActionTracker<ModeValues.AsObject>} */
    newActionTracker());

watch(() => props.name, async (name) => {
  // close existing stream if present
  closeResource(modeValuesResource);
  // create new stream
  if (name && name !== '') {
    pullModeValues({name: name}, modeValuesResource);
  }
}, {immediate: true});

onUnmounted(() => {
  closeResource(modeValuesResource);
});

const modeValuesMap = computed(() => {
  const out = {};
  if (modeValuesResource.value) {
    for (const [k, v] of modeValuesResource.value.valuesMap) {
      out[k] = v;
    }
  }
  return out;
});
const hvacIsAuto = computed(() => {
  return modeValuesMap.value['hvac.mode'] === 'auto';
});
const hasHvacAutoSwitch = computed(() => {
  if (!modeValuesResource.value) return false;
  return modeValuesMap.value['hvac.mode'] !== undefined;
});

/**
 * @param {boolean} value
 */
function autoMode(value) {
  if (!modeValuesResource.value) return; // can't update without all the data
  if (hvacIsAuto.value === value) return; // already in the desired state
  const req = {
    name: props.name,
    modeValues: modeValuesResource.value
  };
  req.modeValues.valuesMap = req.modeValues.valuesMap.map(kv => {
    if (kv[0] === 'hvac.mode') {
      kv[1] = value ? 'auto' : 'manual';
    }
    return kv;
  });
  updateModeValues(req, updateModeValuesTracker);
}
</script>

<style lang="scss" scoped>
</style>
