<template>
  <div :class="['floor-list', { 'floor-list--compact': props.compact }]">
    <template v-for="floor in floorItems" :key="floor.level">
      <slot name="floor" v-bind="{floor}"/>
      <template v-if="floor.level === 0">
        <slot name="ground">
          <v-divider :thickness="props.compact ? 2 : 3"/>
        </slot>
      </template>
    </template>
  </div>
</template>

<script setup>
/**
 * @typedef {Object} Floor
 * @property {string} title - display name of the floor
 * @property {number} level - 0 for ground, negative for basements (counting down), positive for upper floors (counting up). Defaults to len - i - 1.
 */

import {computed} from 'vue';

const props = defineProps({
  floors: {
    type: Array, // of Floor
    required: true
  },
  selectedFloor: {
    type: Number,
    default: null
  },
  // When true, floors use a fixed row height instead of stretching equally to fill the container.
  // The parent is responsible for overflow-y scrolling.
  compact: {
    type: Boolean,
    default: false
  }
});

const floorItems = computed(() => {
  const sorted = props.floors.map((floor, i) => ({
    ...floor,
    level: floor.level ?? (props.floors.length - i - 1)
  }));
  sorted.sort((a, b) => b.level - a.level);
  return sorted;
})
</script>

<style scoped>
.floor-list {
  display: flex;
  flex-direction: column;
  gap: 5px;
  justify-content: stretch;
  align-items: stretch;
}

.floor-list > *:not(.v-divider) {
  flex: 1;
}

/* compact: rows have a natural fixed height; parent handles overflow scrolling */
.floor-list--compact {
  gap: 2px;
}

.floor-list--compact > *:not(.v-divider) {
  flex: 0 0 auto;
}
</style>