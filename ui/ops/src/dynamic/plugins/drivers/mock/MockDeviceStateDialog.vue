<template>
  <v-dialog :model-value="modelValue" max-width="500" @update:model-value="$emit('update:modelValue', $event)">
    <v-card :title="`State: ${device?.name ?? ''}`">
      <v-card-text>
        <div v-if="!traitControls.length" class="text-body-2 text-medium-emphasis">
          No forceable traits on this device.
        </div>
        <div v-for="ctrl in traitControls" :key="ctrl.trait" class="mb-4">
          <div class="d-flex align-center justify-space-between mb-1">
            <span class="text-caption text-medium-emphasis">{{ ctrl.label }}</span>
            <v-btn
                :prepend-icon="ctrl.autoActive ? 'mdi-play-circle-outline' : 'mdi-pause-circle-outline'"
                :text="ctrl.autoActive ? 'Auto' : 'Paused'"
                :color="ctrl.autoActive ? undefined : 'warning'"
                density="compact"
                variant="text"
                size="small"
                @click="doToggleAuto(ctrl)"/>
          </div>

          <!-- Toggle pair (OccupancySensor, MotionSensor, OnOff, LockUnlock) -->
          <v-btn-toggle v-if="ctrl.type === 'toggle'" density="compact" mandatory
                        :model-value="ctrl.value"
                        @update:model-value="doForce(ctrl, $event)">
            <v-btn v-for="opt in ctrl.options" :key="opt.value" :value="opt.value" size="small">
              {{ opt.label }}
            </v-btn>
          </v-btn-toggle>

          <!-- Slider (BrightnessSensor, Light, FanSpeed) -->
          <div v-else-if="ctrl.type === 'slider'" class="d-flex align-center gap-4">
            <v-slider
                :model-value="ctrl.value"
                :min="ctrl.min" :max="ctrl.max" :step="ctrl.step"
                hide-details
                @end="doForce(ctrl, $event)"/>
            <span class="text-body-2 text-no-wrap" style="min-width: 4em">
              {{ ctrl.value?.toFixed(ctrl.decimals ?? 0) }}{{ ctrl.unit ?? '' }}
            </span>
          </div>

          <!-- Number input (AirTemperature ambient) -->
          <v-text-field
              v-else-if="ctrl.type === 'number'"
              :model-value="ctrl.value"
              type="number"
              :suffix="ctrl.unit"
              density="compact"
              variant="outlined"
              hide-details
              style="max-width: 160px"
              @change="doForce(ctrl, parseFloat($event.target.value))"/>
        </div>
        <v-alert v-if="error" type="error" density="compact" class="mt-2">{{ error }}</v-alert>
      </v-card-text>
      <v-card-actions>
        <v-spacer/>
        <v-btn text="Close" @click="$emit('update:modelValue', false)"/>
      </v-card-actions>
    </v-card>
  </v-dialog>
</template>

<script setup>
import {forceTraitValue, setDeviceAutomation} from '@/api/sc/traits/mock.js';
import {computed, ref, watch} from 'vue';

const props = defineProps({
  modelValue: {type: Boolean, default: false},
  device: {type: Object, default: null}
});
defineEmits(['update:modelValue']);

const error = ref(null);

// Per-trait control state (local, not synced from server — just for display)
const localValues = ref({});

watch(() => props.modelValue, open => {
  if (open) {
    error.value = null;
    localValues.value = {};
  }
});

const TRAIT = {
  OccupancySensor: 'smartcore.traits.OccupancySensor',
  MotionSensor: 'smartcore.traits.MotionSensor',
  BrightnessSensor: 'smartcore.traits.BrightnessSensor',
  AirTemperature: 'smartcore.traits.AirTemperature',
  OnOff: 'smartcore.traits.OnOff',
  Light: 'smartcore.traits.Light',
  FanSpeed: 'smartcore.traits.FanSpeed',
  LockUnlock: 'smartcore.traits.LockUnlock',
};

// Key for per-trait automation active state in localValues
const autoKey = trait => `__auto__${trait}`;

const traitSet = computed(() => new Set((props.device?.traits ?? []).map(t => t.name)));

/**
 * @param {{toJson: function(*): object}} ctrl
 * @param {*} value
 * @return {string}
 */
function buildValueJson(ctrl, value) {
  return JSON.stringify(ctrl.toJson(value));
}

const traitControls = computed(() => {
  if (!props.device) return [];
  const out = [];

  if (traitSet.value.has(TRAIT.OccupancySensor)) {
    out.push({
      trait: TRAIT.OccupancySensor,
      label: 'Occupancy',
      type: 'toggle',
      get value() { return localValues.value[TRAIT.OccupancySensor] ?? 'UNOCCUPIED'; },
      get autoActive() { return localValues.value[autoKey(TRAIT.OccupancySensor)] ?? true; },
      options: [{label: 'Occupied', value: 'OCCUPIED'}, {label: 'Unoccupied', value: 'UNOCCUPIED'}],
      toJson: v => ({state: v}),
    });
  }

  if (traitSet.value.has(TRAIT.MotionSensor)) {
    out.push({
      trait: TRAIT.MotionSensor,
      label: 'Motion',
      type: 'toggle',
      get value() { return localValues.value[TRAIT.MotionSensor] ?? 'NOT_DETECTED'; },
      get autoActive() { return localValues.value[autoKey(TRAIT.MotionSensor)] ?? true; },
      options: [{label: 'Detected', value: 'DETECTED'}, {label: 'Not Detected', value: 'NOT_DETECTED'}],
      toJson: v => ({state: v}),
    });
  }

  if (traitSet.value.has(TRAIT.OnOff)) {
    out.push({
      trait: TRAIT.OnOff,
      label: 'On/Off',
      type: 'toggle',
      get value() { return localValues.value[TRAIT.OnOff] ?? 'OFF'; },
      get autoActive() { return localValues.value[autoKey(TRAIT.OnOff)] ?? true; },
      options: [{label: 'On', value: 'ON'}, {label: 'Off', value: 'OFF'}],
      toJson: v => ({state: v}),
    });
  }

  if (traitSet.value.has(TRAIT.LockUnlock)) {
    out.push({
      trait: TRAIT.LockUnlock,
      label: 'Lock',
      type: 'toggle',
      get value() { return localValues.value[TRAIT.LockUnlock] ?? 'UNLOCKED'; },
      get autoActive() { return localValues.value[autoKey(TRAIT.LockUnlock)] ?? true; },
      options: [{label: 'Locked', value: 'LOCKED'}, {label: 'Unlocked', value: 'UNLOCKED'}],
      toJson: v => ({position: v}),
    });
  }

  if (traitSet.value.has(TRAIT.Light)) {
    out.push({
      trait: TRAIT.Light,
      label: 'Light level',
      type: 'slider',
      get value() { return localValues.value[TRAIT.Light] ?? 100; },
      get autoActive() { return localValues.value[autoKey(TRAIT.Light)] ?? true; },
      min: 0, max: 100, step: 1, unit: '%',
      toJson: v => ({levelPercent: v}),
    });
  }

  if (traitSet.value.has(TRAIT.BrightnessSensor)) {
    out.push({
      trait: TRAIT.BrightnessSensor,
      label: 'Ambient brightness',
      type: 'slider',
      get value() { return localValues.value[TRAIT.BrightnessSensor] ?? 500; },
      get autoActive() { return localValues.value[autoKey(TRAIT.BrightnessSensor)] ?? true; },
      min: 0, max: 10000, step: 10, unit: ' lx', decimals: 0,
      toJson: v => ({brightnessLux: v}),
    });
  }

  if (traitSet.value.has(TRAIT.FanSpeed)) {
    out.push({
      trait: TRAIT.FanSpeed,
      label: 'Fan speed',
      type: 'slider',
      get value() { return localValues.value[TRAIT.FanSpeed] ?? 50; },
      get autoActive() { return localValues.value[autoKey(TRAIT.FanSpeed)] ?? true; },
      min: 0, max: 100, step: 1, unit: '%',
      toJson: v => ({percentage: v}),
    });
  }

  if (traitSet.value.has(TRAIT.AirTemperature)) {
    out.push({
      trait: TRAIT.AirTemperature,
      label: 'Ambient temperature',
      type: 'number',
      get value() { return localValues.value[TRAIT.AirTemperature] ?? 21; },
      get autoActive() { return localValues.value[autoKey(TRAIT.AirTemperature)] ?? true; },
      unit: '°C',
      toJson: v => ({ambientTemperature: {valueCelsius: v}}),
    });
  }

  return out;
});

/**
 * @param {{trait: string, toJson: function(*): object}} ctrl
 * @param {*} value
 */
async function doForce(ctrl, value) {
  const prevValue = localValues.value[ctrl.trait];
  localValues.value = {...localValues.value, [ctrl.trait]: value};
  error.value = null;
  try {
    await forceTraitValue({
      name: props.device.name,
      values: [{trait: ctrl.trait, valueProtojson: buildValueJson(ctrl, value)}],
    });
    if (ctrl.autoActive) {
      localValues.value = {...localValues.value, [autoKey(ctrl.trait)]: false};
      await setDeviceAutomation({
        name: props.device.name,
        automations: [{trait: ctrl.trait, active: false}],
      });
    }
  } catch (e) {
    // revert optimistic update on error
    localValues.value = {...localValues.value, [ctrl.trait]: prevValue};
    error.value = e.message ?? String(e);
  }
}

/**
 * @param {{trait: string, autoActive: boolean}} ctrl
 */
async function doToggleAuto(ctrl) {
  const newActive = !ctrl.autoActive;
  localValues.value = {...localValues.value, [autoKey(ctrl.trait)]: newActive};
  error.value = null;
  try {
    await setDeviceAutomation({
      name: props.device.name,
      automations: [{trait: ctrl.trait, active: newActive}],
    });
  } catch (e) {
    // revert optimistic update on error
    localValues.value = {...localValues.value, [autoKey(ctrl.trait)]: !newActive};
    error.value = e.message ?? String(e);
  }
}
</script>
