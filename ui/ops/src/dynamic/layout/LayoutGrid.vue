<template>
  <div class="grid--container" :class="{signage: signageEnabled}" :style="containerStyles">
    <div
        v-for="(cell, index) in props.cells"
        :key="cell.id ?? `${cell.component}-${index}`"
        :style="cellStyles(cell)"
        class="grid--cell-wrapper">
      <component
          :is="cellComponent(cell)"
          v-bind="cell.props"
          class="grid--cell"/>
      <v-tooltip v-if="cell.description" location="top start" offset="5">
        <template #activator="{props: tp}">
          <v-icon
              v-bind="tp"
              class="help-icon"
              size="18">
            mdi-help-circle-outline
          </v-icon>
        </template>
        <div class="help-tooltip-content">
          {{ cell.description }}
        </div>
      </v-tooltip>
    </div>
  </div>
</template>

<script setup>
import {computed, defineAsyncComponent} from 'vue';
import useSignage from '@/composables/signage.js';
const PlaceholderCard = defineAsyncComponent(() => import('@/dynamic/widgets/general/PlaceholderCard.vue'));

const props = defineProps({
  cells: {
    type: Array,
    required: true
  }
});
const countLines = (cells, inlineStartProp, inlineSpanProp) => {
  let lines = 0;
  for (const cell of cells) {
    const end = cell.loc[inlineStartProp] + cell.loc[inlineSpanProp];
    if (end > lines) {
      lines = end;
    }
  }
  return lines;
}
// these have -1 because there's 1 less column/row than lines: | col1 | col2 | <- cols = 2, lines = 3
const columnCount = computed(() => Math.max(1, countLines(props.cells, 'x', 'w') - 1));
const rowCount = computed(() => Math.max(1, countLines(props.cells, 'y', 'h') - 1));

const {enabled: signageEnabled, styles: signageStyles} = useSignage();

const containerStyles = computed(() => {
  return {
    '--column-count': columnCount.value,
    '--row-count': rowCount.value,
    ...signageStyles.value,
  };
});

const cellStyles = (cell) => {
  return {
    '--x': cell.loc.x,
    '--y': cell.loc.y,
    '--w': cell.loc.w,
    '--h': cell.loc.h,
  };
}
const cellComponent = (cell) => {
  return cell.component ?? PlaceholderCard;
}
</script>

<style scoped lang="scss">
.grid--container {
  --gap: 10px;
  display: grid;
  grid-template-columns: repeat(var(--column-count), 1fr);
  grid-template-rows: repeat(var(--row-count), 80px);
  grid-auto-rows: 80px;
  align-content: start;
  gap: var(--gap);
  padding: var(--gap);
}

.signage.grid--container {
  grid-template-rows: repeat(var(--row-count), minmax(80px, 1fr));
  grid-auto-rows: minmax(80px, 1fr);
  align-content: stretch;
  min-height: 100%;
  padding: var(--gap);
  // add scrollbar gutter to match the app one when needed
  scrollbar-gutter: stable;

  .grid--cell-wrapper {
    overflow: hidden;
  }
}

.grid--cell-wrapper {
  grid-column-start: var(--x);
  grid-column-end: span var(--w);
  grid-row-start: var(--y);
  grid-row-end: span var(--h);
  min-height: 0;
  min-width: 0;
  position: relative;
  display: flex;
  flex-direction: column;
}

.grid--cell {
  flex: 1;
  min-height: 0;
  min-width: 0;
  overflow: auto;
}

.help-icon {
  position: absolute;
  bottom: 8px;
  left: 8px;
  z-index: 10;
  opacity: 0.3;
  transition: opacity 0.2s;
  cursor: help;
}

.help-icon:hover {
  opacity: 1;
}

.help-tooltip-content {
  max-width: 320px;
  font-size: 0.85rem;
  line-height: 1.4;
  padding: 4px;
}
</style>