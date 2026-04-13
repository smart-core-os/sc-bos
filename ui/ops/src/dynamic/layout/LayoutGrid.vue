<template>
  <div class="grid--container" :class="{signage: signageEnabled}" :style="containerStyles">
    <component
        :is="cellComponent(cell)"
        v-for="(cell, index) in props.cells"
        :key="cell.id ?? `${cell.component}-${index}`"
        :style="cellStyles(cell)"
        v-bind="cell.props"
        class="grid--cell"/>
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

  .grid--cell {
    overflow: hidden;
  }
}

.grid--cell {
  grid-column-start: var(--x);
  grid-column-end: span var(--w);
  grid-row-start: var(--y);
  grid-row-end: span var(--h);
  min-height: 0;
  min-width: 0;
  overflow: auto;
}
</style>