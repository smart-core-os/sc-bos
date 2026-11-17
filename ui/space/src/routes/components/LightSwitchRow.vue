<template>
  <div>
    <div class="d-flex align-center">
      <v-icon size="35">mdi-lightbulb-outline</v-icon>
      <span class="text-h6 flex-grow-1 ml-4">{{ title || name }}</span>
      <v-switch
          :aria-label="title || name"
          color="accent"
          :disabled="!groupValue.value || disabled"
          hide-details
          inset
          :model-value="isOn"
          @update:model-value="toggle"/>
    </div>
    <v-progress-linear :active="updateTracker.loading" color="primary" indeterminate/>
  </div>
</template>

<script setup>
import {closeResource, newActionTracker, newResourceValue} from '@/api/resource';
import {pullBrightness, updateBrightness} from '@/api/sc/traits/light';
import {useRoundTrip} from '@/routes/components/useRoundTrip';
import {computed, onUnmounted, reactive, toRef, watch} from 'vue';

const props = defineProps({
  // the full name of the light group, e.g. zones/rooms/copper/feature
  name: {
    type: String,
    default: ''
  },
  title: {
    type: String,
    default: ''
  },
  disabled: {
    type: Boolean,
    default: false
  }
});
const emit = defineEmits(['change']);

const groupValue = reactive(
    /** @type {ResourceValue<Brightness.AsObject, PullBrightnessResponse.AsObject>} */
    newResourceValue());

watch(() => props.name, (name) => {
  closeResource(groupValue);
  if (name && name !== '') {
    pullBrightness({name}, groupValue);
  }
}, {immediate: true});

onUnmounted(() => {
  closeResource(groupValue);
});

const {localValue, value} = useRoundTrip(toRef(groupValue, 'value'));

// Any fitting on reads as on, so a partly-lit group is turned fully off by a single tap.
const isOn = computed(() => (value.value?.levelPercent ?? 0) > 0);

const updateTracker = reactive(
    /** @type {ActionTracker<Brightness.AsObject>} */
    newActionTracker());

/**
 * @param {boolean|null} on
 */
function toggle(on) {
  if (!groupValue.value) return; // can't update before the current value is known
  const levelPercent = on ? 100 : 0;
  localValue.value = {...groupValue.value, levelPercent};
  updateBrightness({name: props.name, brightness: {levelPercent}}, updateTracker);
  emit('change', on);
}
</script>
