<template>
  <svg
      :width="size"
      :height="size"
      viewBox="0 0 360 360"
      xmlns="http://www.w3.org/2000/svg">
    <!-- Background track -->
    <path
        d="M 58.76 260.00 A 140 140 0 1 1 301.24 260.00"
        fill="none"
        stroke="#26004D"
        stroke-width="22"
        stroke-linecap="round"/>

    <!-- Value fill arc -->
    <path
        v-if="valuePath"
        :d="valuePath"
        fill="none"
        :stroke="color"
        stroke-width="22"
        stroke-linecap="round"
        style="transition: d 0.6s ease"/>

    <!-- Tip dot at value endpoint -->
    <circle
        v-if="valuePath"
        :cx="valueEndX"
        :cy="valueEndY"
        r="11"
        :fill="color"/>

    <!-- Value text -->
    <text
        x="180"
        y="180"
        text-anchor="middle"
        dominant-baseline="middle"
        fill="#F8F4F1"
        :font-size="centerFontSize"
        font-weight="700"
        font-family="Poppins, sans-serif">
      {{ displayText ?? displayValue }}
    </text>

    <!-- Sensor label -->
    <text
        x="180"
        y="348"
        text-anchor="middle"
        fill="#F8F4F1"
        fill-opacity="0.65"
        font-size="34"
        font-weight="600"
        font-family="Poppins, sans-serif">
      {{ label }}
    </text>
  </svg>
</template>

<script setup>
import {computed} from 'vue';

const props = defineProps({
  value:       {type: Number, default: null},
  min:         {type: Number, required: true},
  max:         {type: Number, required: true},
  unit:        {type: String, required: true},
  label:       {type: String, required: true},
  color:       {type: String, default: '#4caf50'},
  size:        {type: Number, default: 360},
  displayText: {type: String, default: null}
});

const fraction = computed(() => {
  if (props.value === null) return 0;
  return Math.max(0, Math.min(0.9999, (props.value - props.min) / (props.max - props.min)));
});

const valueEndX = computed(() => {
  const deg = 150 + fraction.value * 240;
  return 180 + 140 * Math.cos(deg * Math.PI / 180);
});

const valueEndY = computed(() => {
  const deg = 150 + fraction.value * 240;
  return 190 + 140 * Math.sin(deg * Math.PI / 180);
});

const valueLargeArc = computed(() => fraction.value * 240 > 180 ? 1 : 0);

const valuePath = computed(() => {
  if (props.value === null || fraction.value === 0) return '';
  return `M 58.76 260.00 A 140 140 0 ${valueLargeArc.value} 1 ${valueEndX.value.toFixed(2)} ${valueEndY.value.toFixed(2)}`;
});

const displayValue = computed(() => {
  if (props.value === null) return '—';
  if (props.max <= 1) return props.value.toFixed(2);
  if (props.max <= 10) return props.value.toFixed(1);
  return Math.round(props.value).toString();
});

const centerFontSize = computed(() => {
  if (!props.displayText) return 80;
  const len = props.displayText.length;
  if (len <= 4) return 70;
  if (len <= 8) return 56;
  return 44;
});
</script>
