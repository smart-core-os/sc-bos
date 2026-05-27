<template>
  <v-card elevation="0" tile>
    <!-- Brightness -->
    <v-list tile class="ma-0 pa-0">
      <v-list-subheader class="text-title-caps-large text-neutral-lighten-3">Lighting</v-list-subheader>
      <v-list-item class="py-1">
        <v-list-item-title class="text-body-small text-capitalize">Brightness</v-list-item-title>
      </v-list-item>
    </v-list>
    <v-slider
        class="mx-4 my-2"
        color="accent"
        track-color="neutral-lighten-1"
        :disabled="blockActions || !value"
        hide-details
        :min="0"
        :max="100"
        :step="1"
        v-model="sliderBrightness">
      <template #prepend>
        <v-icon>{{ icon }}</v-icon>
      </template>
      <template #append>
        <v-tooltip v-if="lowConfidence" location="top">
          <template #activator="{ props: tooltipProps }">
            <span v-bind="tooltipProps" class="text-body-1 slider-label">{{ levelStr }}*</span>
          </template>
          Low confidence reading
        </v-tooltip>
        <span v-else class="text-body-1 slider-label">{{ levelStr }}</span>
      </template>
    </v-slider>
    <v-card-actions class="px-4">
      <v-btn
          v-for="control in brightnessControl"
          :disabled="control.disabled"
          :key="control.label"
          size="small"
          variant="tonal"
          @click="control.onClick">
        {{ control.label }}
      </v-btn>
      <v-spacer/>
    </v-card-actions>

    <!-- Presets -->
    <v-container v-if="presets.length > 0" class="pa-4">
      <v-list tile class="ma-0 mt-2 pa-0">
        <v-list-item class="py-1 pl-0">
          <v-list-item-title class="text-body-small text-capitalize">Presets</v-list-item-title>
        </v-list-item>
      </v-list>
      <v-btn
          v-for="preset in presets"
          block
          class="py-1 mx-0 mt-1 mb-2 preset"
          :color="getColor(preset.title, currentPresetTitle)"
          elevation="0"
          :key="preset.name"
          size="small"
          width="100%"
          max-width="575"
          @click="updateBrightnessPreset(preset)">
        <span class="text-truncate">
          {{ preset.title ? preset.title : preset.name }}
        </span>
      </v-btn>
    </v-container>
    <v-progress-linear color="primary" indeterminate :active="loading"/>
  </v-card>
</template>

<script setup>
import useAuthSetup from '@/composables/useAuthSetup';
import {
  useBrightness,
  useDescribeBrightness,
  usePullBrightness,
  useUpdateBrightness
} from '@/traits/light/light.js';
import debounce from 'debounce';
import {computed, ref, watch} from 'vue';

const {blockActions} = useAuthSetup();
const props = defineProps({
  name: {
    type: String,
    default: ''
  }
});

const {value, loading: pullLoading} = usePullBrightness(() => props.name);
const {response: support, loading: supportLoading} = useDescribeBrightness(() => props.name);
const {updateBrightness, loading: updateLoading} = useUpdateBrightness(() => props.name);
const {levelStr, level, icon, lowConfidence, presets, currentPresetTitle} = useBrightness(value, support);

const loading = computed(() => pullLoading.value || supportLoading.value || updateLoading.value);

// Local ref for optimistic slider updates — syncs from server but not while user is dragging
const localLevel = ref(0);
watch(level, (v) => { localLevel.value = v; }, {immediate: true});

const updateBrightnessDebounced = debounce((v) => updateBrightness(v), 200);

const sliderBrightness = computed({
  get: () => localLevel.value,
  set: (v) => {
    localLevel.value = v;
    updateBrightnessDebounced(v);
  }
});

/**
 * @param {string} title
 * @param {string} currentPresetTitle
 * @return {string}
 */
function getColor(title, currentPresetTitle) {
  return title === currentPresetTitle ? 'primary' : 'neutral-lighten-1';
}

/**
 * @param {LightPreset.AsObject} preset
 * @return {Promise<Brightness.AsObject>}
 */
function updateBrightnessPreset(preset) {
  return updateBrightness({preset: preset});
}

const brightnessControl = computed(() => [
  {
    disabled: blockActions.value,
    label: 'On',
    onClick: () => {
      localLevel.value = 100;
      updateBrightness(100);
    }
  },
  {
    disabled: blockActions.value,
    label: 'Off',
    onClick: () => {
      localLevel.value = 0;
      updateBrightness(0);
    }
  }
]);
</script>

<style lang="scss" scoped>
.v-list-item {
  min-height: auto;
}

.slider-label {
  min-width: 3em;
  text-align: right;
}

.preset :deep(.v-btn__content) {
  max-width: 100%;
}
</style>
