<template>
  <div class="d-flex align-center justify-center">
    <v-chip v-if="hasMeasured" v-bind="chipAttrs">
      <span v-tooltip="'Current temperature'">{{ measuredStr }}</span>
    </v-chip>
  </div>
</template>

<script setup>
import {useTemperatureValues} from '@/traits/temperature/temperature.js';
import {computed, ref} from 'vue';

const props = defineProps({
  measured: {
    type: [Number, String],
    default: null
  },
  size: {
    type: [Number, String],
    default: 40
  },
  variant: {
    type: String,
    default: 'outlined'
  },
  color: {
    type: String,
    default: ''
  }
});

const measuredNum = computed(() => props.measured);
const {
  hasMeasured,
  measuredStr
} = useTemperatureValues(measuredNum, ref(undefined));

// layout and sizing for the chip
const chipSize = computed(() => {
  const s = +props.size;
  if (s < 32) return 'x-small';
  if (s < 44) return 'small';
  if (s < 56) return 'default';
  if (s < 68) return 'large';
  return 'x-large';
});

const chipAttrs = computed(() => {
  const attrs = {
    size: chipSize.value,
    variant: props.variant,
    color: props.color
  };
  if (props.variant.startsWith('outlined')) {
    attrs.variant = 'outlined';
  }
  return attrs;
});

</script>

<style scoped>
.v-chip {
  font-size: 0.9rem;
}
</style>

