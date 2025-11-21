<template>
  <span :class="{stacked: props.stacked}">
    <span v-if="showPrefix" class="aux"><slot name="prefix">{{ prefix }}</slot></span>
    <slot>{{ value }}</slot>
    <span v-if="showSuffix" class="aux"><slot name="suffix">{{ suffix }}</slot></span>
  </span>
</template>

<script setup>
import {computed, useSlots} from 'vue';

const props = defineProps({
  value: {
    type: String,
    default: ''
  },
  scale: {
    type: Number,
    default: .5
  },
  prefix: {
    type: String,
    default: ''
  },
  suffix: {
    type: String,
    default: ''
  },
  stacked: {
    type: Boolean,
    default: false
  }
});
const slots = useSlots()
const showPrefix = computed(() => !!(props.prefix || (slots.prefix)));
const showSuffix = computed(() => !!(props.suffix || (slots.suffix)));
</script>

<style scoped>
.aux {
  font-size: calc(v-bind(scale) * 100%);
  font-weight: lighter;
}
.stacked {
  display: inline-flex;
  flex-direction: column;
  align-items: start;
  line-height: 1.1;
}
</style>