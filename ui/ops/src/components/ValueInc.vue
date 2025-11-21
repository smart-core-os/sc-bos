<template>
  <span class="value-inc">
    <slot>{{ valueStr }}</slot>
    <v-icon :icon="icon" :color="color"/>
  </span>
</template>

<script setup>
import {computed} from 'vue';

const props = defineProps({
  value: {
    type: Number,
    default: 0
  },
  // the direction of the arrow.
  // When 'auto', the direction is down when value is negative, up when positive, and none when zero
  direction: {
    type: String,
    default: 'auto' // auto, up, down, none
  },
  // The color of the arrow.
  // When 'auto', positive values use the success color, negative values use the error color.
  // This behaviour can be inverted with lowerIsBetter.
  color: {
    type: String,
    default: 'auto' // auto, or vuetify color
  },
  // for auto modes, negative values are considered better and show good colours
  lowerIsBetter: {
    type: Boolean,
    default: false
  }
});

const valueStr = computed(() => {
  return `${Math.abs(props.value)}`
});
const icon = computed(() => {
  if (props.direction === 'none' || props.value === 0) {
    return 'mdi-circle-small';
  }
  const dir = props.direction === 'auto' ? (props.value > 0 ? 'up' : 'down') : props.direction;
  return dir === 'up' ? 'mdi-menu-up' : 'mdi-menu-down';
});
const color = computed(() => {
  if (props.color !== 'auto') {
    return props.color;
  }
  if (props.value === 0) {
    return 'grey';
  }
  const isGood = props.lowerIsBetter ? props.value < 0 : props.value > 0;
  return isGood ? 'success' : 'error';
})
</script>

<style scoped>
.value-inc {
  display: inline-flex;
  align-items: center;
}
</style>