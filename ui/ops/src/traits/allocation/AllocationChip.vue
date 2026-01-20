<template>
  <div class="d-flex align-center justify-center overlap" :class="[layoutClass, variantClass]">
    <v-chip v-if="hasAllocationTotal && hasUnallocationTotal" v-bind="chipAttrs">
      {{ allocationTotal }} / {{ totalAllocation }}
    </v-chip>
    <v-avatar class="icon-container" v-bind="avatarAttrs" v-tooltip="avatarTooltip">
      <v-icon :size="props.size - 10" :icon="movementIcon" :color="movementColor"/>
    </v-avatar>
  </div>
</template>

<script setup>
import {computed} from 'vue';

const props = defineProps({
  allocationTotal: {
    type: Number,
    default: 0
  },
  unallocationTotal: {
    type: Number,
    default: 0
  },
  size: {
    type: [Number, String],
    default: 40
  },
  variant: {
    type: String,
    default: 'outlined'
  },
  layout: {
    type: String,
    default: 'right'
  },
  color: {
    type: String,
    default: ''
  }
});

const hasAllocationTotal = computed(() => typeof props.allocationTotal === 'number');
const hasUnallocationTotal = computed(() => typeof props.unallocationTotal === 'number');

const totalAllocation = computed(() => props.allocationTotal + props.unallocationTotal);

const movementIcon = computed(() => {
  if (props.allocationTotal === 0 && props.unallocationTotal === 0) {
    return 'mdi-help-circle-outline';
  }
  if (props.allocationTotal > 0) {
    return 'mdi-check-circle';
  }
  return 'mdi-circle-outline';
});

const movementColor = computed(() => {
  const total = totalAllocation.value;
  if (total === 0) {
    return 'neutral-lighten-2';
  }

  const percentage = (props.allocationTotal / total) * 100;

  // the levels are arbitrary but seem sensible, can be adjusted later
  if (percentage <= 30) {
    return 'green';
  } else if (percentage <= 60) {
    return 'yellow';
  } else if (percentage <= 90) {
    return 'orange';
  } else {
    return 'red';
  }
});

const avatarTooltip = computed(() => {
  const total = totalAllocation.value;
  if (total === 0) {
    return 'No allocations';
  }
  const percentage = Math.round((props.allocationTotal / total) * 100);
  return `${percentage}% allocated`;
});

// layout and sizing for the chip
const chipSize = computed(() => {
  const s = +props.size;
  if (s < 32) return 'x-small';
  if (s < 44) return 'small';
  if (s < 56) return 'default';
  if (s < 68) return 'large';
  return 'x-large';
});

const sizeVar = computed(() => {
  return props.size + 'px';
});

const layoutClass = computed(() => `allocation-chip--layout-${props.layout ?? 'right'}`);
const variantClass = computed(() => `allocation-chip--variant-${props.variant ?? 'outlined'}`);

const avatarAttrs = computed(() => {
  const attrs = {
    color: props.color,
    size: props.size,
    variant: props.variant
  };
  if (props.variant.startsWith('outlined')) {
    attrs.variant = 'outlined';
  }
  return attrs;
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

<style scoped lang="scss">
.overlap {
  --size: v-bind(sizeVar);
  --r: calc(var(--size) / 2);
}

.v-chip {
  mask-image: url('data:image/svg+xml;utf8,<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 100 100" fill="none"><circle r="50" cx="50" cy="50" fill="black"/></svg>'), linear-gradient(#fff, #fff);
  mask-size: auto var(--size);
  mask-repeat: no-repeat;
  mask-composite: exclude;

  font-size: calc(var(--size) * 0.4);
  height: auto;
  padding-block: .15em;

  overflow: visible;
  min-width: min-content;

  > * {
    // small adjustment to make the text appear more central vertically
    margin-top: .15em;
  }
}

.allocation-chip {
  &--layout {
    &-start, &-left {
      flex-direction: row-reverse;

      .v-chip {
        mask-position: calc(-1 * var(--r)) center, center;
        padding-left: calc(var(--r) + .6em);
        margin-left: calc(var(--r) * -1);
        border-left-color: transparent;
        border-bottom-left-radius: 0;
        border-top-left-radius: 0;
      }
    }

    &-end, &-right {
      flex-direction: row;
      justify-content: start;

      .v-chip {
        mask-position: calc(100% + var(--r)) center, center;
        padding-right: calc(var(--r) + .6em);
        margin-right: calc(var(--r) * -1);
        border-right-color: transparent;
        border-bottom-right-radius: 0;
        border-top-right-radius: 0;
      }
    }

    &-top {
      flex-direction: column-reverse;

      .v-chip {
        mask-position: center calc(-1 * var(--size) + .6em), center;
        margin-top: -.6em;
      }
    }

    &-bottom {
      flex-direction: column;

      .v-chip {
        mask-position: center calc(100% + var(--size) - .6em), center;
        margin-bottom: -.6em;
      }
    }
  }

  &--variant {
    &-outlined-filled {
      .v-chip, .v-avatar {
        color: rgb(var(--v-theme-on-surface));
        border-color: rgb(var(--v-theme-on-surface));
        background-color: rgb(var(--v-theme-surface));
      }
    }

    &-outlined-inverted {
      .v-chip, .v-avatar {
        color: rgb(var(--v-theme-surface));
        border-color: rgb(var(--v-theme-surface));
        background-color: rgb(var(--v-theme-on-surface));
      }
    }
  }
}
</style>

