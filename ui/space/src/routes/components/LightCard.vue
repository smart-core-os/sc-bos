<template>
  <v-card elevation="0">
    <v-card-title class="d-flex align-center pl-7">
      <span class="text-h4 font-weight-medium flex-grow-1">Lights</span>
      <v-menu v-if="presets.length > 0" :close-on-content-click="true" location="top">
        <template #activator="{ props: menuProps }">
          <v-btn
              v-bind="menuProps"
              :disabled="blockActions"
              size="small"
              variant="tonal"
              prepend-icon="mdi-playlist-check">
            {{ currentPresetTitle || 'Presets' }}
          </v-btn>
        </template>
        <v-list density="compact" class="presets-list">
          <v-list-item
              v-for="preset in presets"
              :key="preset.name"
              :active="preset.title === currentPresetTitle"
              color="accent"
              @click="setPreset(preset)">
            <v-list-item-title>{{ preset.title || preset.name }}</v-list-item-title>
          </v-list-item>
        </v-list>
      </v-menu>
      <v-switch
          v-if="hasLightAutoSwitch"
          color="accent"
          :disabled="blockActions"
          hide-details
          :model-value="lightIsAuto"
          inset
          @update:model-value="autoMode">
        <template #prepend>
          <span class="text-caption text-uppercase">Auto</span>
        </template>
      </v-switch>
    </v-card-title>
    <v-card-text>
      <v-slider
          track-color="#0C0921"
          track-fill-color="#5A0066"
          :disabled="!lightValue.value || blockActions"
          hide-details="auto"
          v-model="brightness">
        <template #prepend>
          <v-icon size="35">mdi-lightbulb-outline</v-icon>
        </template>
        <template #append>
          <span class="text-h5 mr-1">{{ brightness }}%</span>
        </template>
      </v-slider>
    </v-card-text>
    <v-progress-linear :active="updateValue.loading" color="primary" indeterminate/>
  </v-card>
</template>

<script setup>
import {closeResource, newActionTracker, newResourceValue} from '@/api/resource';
import {describeBrightness, pullBrightness, updateBrightness} from '@/api/sc/traits/light';
import {pullModeValues, updateModeValues} from '@/api/sc/traits/mode';
import {useRoundTrip} from '@/routes/components/useRoundTrip';
import debounce from 'debounce';
import {computed, onMounted, onUnmounted, reactive, ref, toRef, watch} from 'vue';
import useAuthSetup from '@/composables/useAuthSetup';

const props = defineProps({
  // the unique name of the device
  name: {
    type: String,
    default: ''
  },
  manualTimeout: {
    type: Number,
    default: 30 * 60 * 1000 // 30 minutes
  }
});
const {blockActions} = useAuthSetup();

const lightValue = reactive(
    /** @type {ResourceValue<Brightness.AsObject, PullBrightnessResponse.AsObject>} */
    newResourceValue());

// if device name changes
watch(() => props.name, async (name) => {
  // close existing stream if present
  closeResource(lightValue);
  // create new stream
  if (name && name !== '') {
    pullBrightness({name: name}, lightValue);
  }
}, {immediate: true});

onUnmounted(() => {
  closeResource(lightValue);
});

const {localValue, value} = useRoundTrip(toRef(lightValue, 'value'));
const brightness = computed({
  get() {
    if (value.value) {
      return Math.round(value.value.levelPercent);
    }
    return '-';
  },
  set(value) {
    // prevent setting a value before current value has been fetched
    if (lightValue.value !== null) {
      if (localValue.value?.levelPercent !== value) {
        localValue.value = {
          ...lightValue.value,
          levelPercent: value
        };
      }
      autoMode(false);
      updateLightDebounced(value);
    }
  }
});

const updateValue = reactive(
    /** @type {ActionTracker<Brightness.AsObject>} */
    newActionTracker());

/**
 * @param {number} value
 */
function updateLight(value) {
  /* @type {UpdateBrightnessRequest.AsObject} */
  const req = {
    name: props.name,
    brightness: {
      levelPercent: Math.min(100, Math.round(value))
    }
  };

  updateBrightness(req, updateValue);
}

const updateLightDebounced = debounce((val) => updateLight(val));

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
const lightIsAuto = computed(() => {
  return modeValuesMap.value['lighting.mode'] === 'auto';
});
const hasLightAutoSwitch = computed(() => {
  if (!modeValuesResource.value) return false;
  return modeValuesMap.value['lighting.mode'] !== undefined;
});

const brightnessSupport = reactive(newActionTracker());

watch(() => props.name, (name) => {
  if (name && name !== '') {
    describeBrightness({name}, brightnessSupport);
  }
}, {immediate: true});

const presets = computed(() => brightnessSupport.response?.presetsList ?? []);
const currentPresetTitle = computed(() => value.value?.preset?.title ?? '');

/**
 * @param {LightPreset.AsObject} preset
 */
function setPreset(preset) {
  autoMode(false);
  updateBrightness({
    name: props.name,
    brightness: {preset}
  }, updateValue);
}

const manualTimeoutHandle = ref(0);

/**
 */
function scheduleManualTimeout() {
  clearTimeout(manualTimeoutHandle.value);
  manualTimeoutHandle.value = setTimeout(() => {
    if (lightIsAuto.value) return; // already in auto mode
    autoMode(true);
  }, props.manualTimeout);
}

onMounted(() => scheduleManualTimeout());

/**
 * @param {boolean} value
 */
function autoMode(value) {
  if (!modeValuesResource.value) return; // can't update without all the data
  if (lightIsAuto.value === value) return; // already in the desired state
  const req = {
    name: props.name,
    modeValues: modeValuesResource.value
  };
  req.modeValues.valuesMap = req.modeValues.valuesMap.map(kv => {
    if (kv[0] === 'lighting.mode') {
      kv[1] = value ? 'auto' : 'manual';
    }
    return kv;
  });
  updateModeValues(req, updateModeValuesTracker);
  if (value) clearTimeout(manualTimeoutHandle.value);
  else scheduleManualTimeout();
}

</script>

<style scoped>
.presets-list {
  background: rgba(224, 223, 222, 0.3) !important;
}
</style>
