<template>
  <div :class="['building-floors', { 'building-floors--compact': props.compact }]">
    <div v-if="props.compact" class="floor-label-header">Floor</div>
    <floor-list
        :class="['floor-list-wrap', { 'floor-list-wrap--compact': props.compact }]"
        :floors="props.floors"
        :compact="props.compact">
      <template #floor="{floor}">
        <div :class="['floor-item', { 'floor-item--compact': props.compact }]">
          <div :class="['floor-title', { 'floor-title--compact': props.compact }]">
            {{ floor.title }}
          </div>
        </div>
      </template>
    </floor-list>
  </div>
</template>

<script setup>
/**
 * @typedef {Object} Floor
 * @property {string} title - display name of the floor
 * @property {number} level - 0 for ground, negative for basements (counting down), positive for upper floors (counting up). Defaults to len - i - 1.
 */

import FloorList from '@/components/FloorList.vue';

const props = defineProps({
  floors: {
    type: Array, // of Floor
    required: true
  },
  selectedFloor: {
    type: Number,
    default: null
  },
  // When true, renders each floor as a compact fixed-height row with a matching header.
  // Useful when there are many floors (20+) that would otherwise be squished.
  compact: {
    type: Boolean,
    default: false
  }
});
</script>

<style scoped>
.building-floors {
  height: 100%;
}

.building-floors--compact {
  display: flex;
  flex-direction: column;
}

/* Matches the height of FloorTraitCells' .column-headers exactly:
   same font-size, padding-bottom, margin-bottom, border-bottom. */
.floor-label-header {
  flex-shrink: 0;
  display: flex;
  align-items: center;
  justify-content: center;
  padding-bottom: 8px;
  margin-bottom: 8px;
  border-bottom: 1px solid rgba(255, 255, 255, 0.12);
  font-size: 0.75rem;
  font-weight: 600;
  color: rgba(255, 255, 255, 0.7);
  text-transform: uppercase;
  letter-spacing: 0.5px;
}

.floor-list-wrap {
  height: 100%;
}

.floor-list-wrap--compact {
  flex: 1;
  min-height: 0;
  overflow-y: auto;
}

.floor-item {
  flex: 1;
  display: flex;
  justify-content: center;
  align-items: center;
  background-color: rgb(var(--v-theme-surface));
  border-radius: 4px;
}

.floor-item--compact {
  flex: 0 0 20px;
  height: 20px;
  border-radius: 3px;
}

.floor-title--compact {
  font-size: 0.75rem;
}
</style>
