<template>
  <v-card elevation="0" tile>
    <v-list tile class="ma-0 pa-0">
      <v-list-subheader class="text-title-caps-large text-neutral-lighten-3">Open / Close</v-list-subheader>

      <v-list-item class="py-1">
        <v-list-item-title class="text-body-small text-capitalize">Open Percentage</v-list-item-title>
        <template #append>
          <v-list-item-subtitle class="text-body-1">{{ openStr }}</v-list-item-subtitle>
        </template>
      </v-list-item>
    </v-list>

    <v-slider
        class="mx-4 my-2"
        :model-value="sliderValue"
        @update:model-value="onSliderInput"
        @end="onSliderEnd"
        :min="sliderMin"
        :max="sliderMax"
        :step="sliderStep"
        thumb-label
        hide-details
        color="accent"
        :disabled="blockActions || updateTracker?.loading"/>

    <v-card-actions v-if="presets.length > 0" class="px-4">
      <v-select
          class="flex-grow-1"
          density="compact"
          variant="outlined"
          hide-details
          label="Preset"
          :items="presetItems"
          :model-value="currentPreset"
          :disabled="blockActions || updateTracker?.loading"
          @update:model-value="setPreset"/>
    </v-card-actions>

    <v-progress-linear color="primary" indeterminate :active="updateTracker?.loading"/>
  </v-card>
</template>

<script setup>
import useAuthSetup from '@/composables/useAuthSetup';
import {useOpenClosePositions} from '@/traits/openClose/openClose.js';
import {computed, ref, watch} from 'vue';

const {blockActions} = useAuthSetup();

const props = defineProps({
  value: {
    type: Object, // of OpenClosePositions.AsObject
    default: () => {}
  },
  info: {
    type: Object, // of PositionsSupport.AsObject
    default: () => null
  },
  updateTracker: {
    type: Object, // of ActionTracker<OpenClosePositions.AsObject>
    default: () => null
  },
  name: {
    type: String,
    default: ''
  }
});
const emit = defineEmits([
  'updatePositions' // OpenClosePositions.AsObject
]);

const {state, openStr, openPercent} = useOpenClosePositions(() => props.value);

const support = computed(() => props.info?.response ?? props.info ?? null);
const presets = computed(() => support.value?.presetsList ?? []);
const presetItems = computed(() => presets.value.map(p => ({title: p.title || p.name, value: p.name})));
const currentPreset = computed(() => props.value?.preset?.name ?? null);

const openPercentAttrs = computed(() => support.value?.openPercentAttributes ?? null);
const sliderMin = computed(() => openPercentAttrs.value?.bounds?.min ?? 0);
const sliderMax = computed(() => openPercentAttrs.value?.bounds?.max ?? 100);
// proto step of 0 means "continuous"; v-slider needs a positive number so fall back to 1.
const sliderStep = computed(() => openPercentAttrs.value?.step || 1);

// While the user is interacting, dragValue holds the local position so the thumb
// tracks the cursor without round-tripping through the device. It is cleared
// once the device echoes back a value matching the user's release, at which
// point sliderValue falls back to mirroring openPercent.
const dragValue = ref(null);
const sliderValue = computed(() => dragValue.value ?? openPercent.value ?? 0);

watch(openPercent, (v) => {
  if (dragValue.value !== null && v === dragValue.value) {
    dragValue.value = null;
  }
});

/**
 * @param {number} v
 */
function onSliderInput(v) {
  dragValue.value = v;
}

/**
 * @param {number} v
 */
function onSliderEnd(v) {
  dragValue.value = v;
  // Preserve the current state's direction so the server addresses the right
  // element on multi-direction devices (and doesn't create a new direction-0
  // element). The update mask scopes the write to open_percent so other
  // fields like resistance / target_open_percent / open_percent_tween aren't
  // reset to defaults.
  const direction = state.value?.direction ?? 0;
  emit('updatePositions', {
    states: {statesList: [{openPercent: v, direction}]},
    updateMask: {pathsList: ['states.open_percent']}
  });
}

/**
 * @param {string} name
 */
function setPreset(name) {
  if (!name) return;
  emit('updatePositions', {preset: {name}});
}
</script>
