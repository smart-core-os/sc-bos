<template>
  <div class="root" :style="{width: props.size + 'px', height: props.size + 'px'}">
    <svg :viewBox="viewBox" fill="none" xmlns="http://www.w3.org/2000/svg">
      <clipPath :id="id">
        <rect v-bind="clipAttrs('right', 'top')"/>
        <rect v-bind="clipAttrs('right', 'bottom')"/>
        <rect v-bind="clipAttrs('left', 'bottom')"/>
        <rect v-bind="clipAttrs('left', 'top')"/>
      </clipPath>
      <circle v-for="(s, i) in segments" :key="i"
              v-bind="{...circleAttrs, ...s}" pathLength="100"
              :clip-path="`url(#${id})`"/>
      />
    </svg>
    <v-icon :icon="iconStr" :size="iconSize" class="mx-auto"/>
  </div>
</template>

<script setup>
import {isNullOrUndef} from '@/util/types.js';
import {computed, ref, useId, watchEffect} from 'vue';

const props = defineProps({
  size: {
    type: [Number, String],
    default: 24
  },
  value: {
    type: [Number],
    default: 24
  },
  cold: {
    type: [Number],
    default: 18
  },
  normal: {
    type: [Number],
    default: 20
  },
  warm: {
    type: [Number],
    default: 25
  },
  hot: {
    type: [Number],
    default: 28
  },
  icon: {
    type: String,
    default: ''
  }
});

const comfort = defineModel('comfort', {
  type: String,
  default: 'normal' // one of 'cold', 'normal', 'warm', 'hot'
});
const showCold = computed(() => true);
const showNormal = computed(() => ['normal', 'warm', 'hot'].includes(comfort.value));
const showWarm = computed(() => ['warm', 'hot'].includes(comfort.value));
const showHot = computed(() => ['hot'].includes(comfort.value));

watchEffect(() => {
  if (isNullOrUndef(props.value)) return;
  if (props.value < props.cold) comfort.value = 'cold';
  else if (props.value < props.normal) comfort.value = 'normal';
  else if (props.value < props.warm) comfort.value = 'warm';
  else comfort.value = 'hot';
});

const iconStr = computed(() => {
  if (props.icon) return props.icon;
  if (showHot.value) return 'mdi-fire';
  if (showWarm.value) return 'mdi-stop-circle';
  if (showNormal.value) return 'mdi-stop-circle';
  return 'mdi-snowflake';
});

// layout and sizing for the svg
const id = useId();
const svgSize = ref(200);
const viewBox = computed(() => `0 0 ${svgSize.value} ${svgSize.value}`);
const pad = computed(() => 0);
const gap = computed(() => svgSize.value / 40);
const cx = computed(() => svgSize.value / 2);
const cy = computed(() => svgSize.value / 2);
const r = computed(() => (svgSize.value / 2) - pad.value);
const strokeWidth = computed(() => svgSize.value / 6);
const iconSize = computed(() => props.size * 0.5);
// x,y can be left,right or top,bottom
const clipAttrs = (x, y) => {
  const attrs = {x: 0, y: 0, width: svgSize.value / 2, height: svgSize.value / 2};
  const halfGap = gap.value / 2;
  switch (x) {
    case 'left':
      attrs.x = -halfGap;
      break;
    case 'right':
      attrs.x = cx.value + halfGap;
      break;
  }
  switch (y) {
    case 'top':
      attrs.y = -halfGap;
      break;
    case 'bottom':
      attrs.y = cy.value + halfGap;
      break;
  }
  // convert to strings
  Object.keys(attrs).forEach(k => attrs[k] = attrs[k].toFixed(4));
  return attrs;
};
const circleAttrs = computed(() => ({
  'r': (r.value - strokeWidth.value / 2 - pad.value / 2).toFixed(4),
  'cx': cx.value.toFixed(4),
  'cy': cy.value.toFixed(4),
  'stroke-width': strokeWidth.value.toFixed(4),
  'stroke-dasharray': `25 75`
}));
const segments = computed(() => {
  return [
    {stroke: '#38BDF8', transform: `rotate(-90 ${cx.value} ${cy.value})`, style: `opacity: ${showCold.value ? 1 : 0.2}`},
    {stroke: '#FBBF24', transform: `rotate(0 ${cx.value} ${cy.value})`, style: `opacity: ${showNormal.value ? 1 : 0.2}`},
    {stroke: '#F97316', transform: `rotate(90 ${cx.value} ${cy.value})`, style: `opacity: ${showWarm.value ? 1 : 0.2}`},
    {stroke: '#E11D48', transform: `rotate(180 ${cx.value} ${cy.value})`, style: `opacity: ${showHot.value ? 1 : 0.2}`}
  ];
});
</script>

<style scoped>
.root {
  display: grid;
  justify-content: center;
  align-items: center;
}

.root > * {
  grid-row: 1;
  grid-column: 1;
}

svg {
  justify-self: stretch;
  align-self: stretch;
}

circle {
  transition: opacity 0.3s cubic-bezier(0.4, 0, 0.2, 1);
}
</style>

