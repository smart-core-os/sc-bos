<template>
  <template v-if="retention">
    <div v-if="retention.bytes != null" class="resource-row">
      <span class="resource-label">Used: </span>
      <v-progress-linear
          v-if="utilization != null"
          :model-value="utilization"
          :color="utilizationColor"
          height="3"
          rounded
          class="resource-bar"/>
      <span class="resource-value" style="width: auto">{{ bytesLabel }}</span>
    </div>
    <div v-if="retention.items?.used != null" class="resource-row">
      <span class="resource-label">Items: </span>
      <span class="resource-value" style="width: auto">{{ itemsLabel }}</span>
    </div>
  </template>
</template>

<script setup>
import {formatBytes} from '@/util/number.js';
import {computed} from 'vue';

const props = defineProps({
  retention: {type: Object, default: () => null},
  itemLabel: {type: String, default: 'item'}
});

const utilization = computed(() => {
  const b = props.retention?.bytes;
  if (b?.used == null || b?.capacity == null || b.capacity <= 0) return null;
  return (b.used / b.capacity) * 100;
});

const utilizationColor = computed(() => {
  const u = utilization.value;
  if (u >= 80) return 'error';
  if (u >= 60) return 'warning';
  return 'success';
});

const bytesLabel = computed(() => {
  const b = props.retention?.bytes;
  if (!b) return '';
  const usedStr = formatBytes(b.used);
  const totalStr = b.capacity != null ? ` / ${formatBytes(b.capacity)}` : '';
  const pctStr = utilization.value != null ? ` (${utilization.value.toFixed(1)}%)` : '';
  return `${usedStr}${totalStr}${pctStr}`;
});

const itemsLabel = computed(() => {
  const items = props.retention?.items;
  if (items?.used == null) return '';
  const totalItems = items.capacity != null ? ` / ${items.capacity.toLocaleString()}` : '';
  return `${items.used.toLocaleString()}${totalItems} ${props.itemLabel}s`;
});
</script>

<style scoped>
.resource-row {
  display: flex;
  align-items: center;
  gap: 6px;
  margin-bottom: 4px;
}

.resource-bar {
  flex: 1;
  max-width: 60px;
}

.resource-label {
  flex: 1;
  min-width: 0;
  font-size: 11px;
  opacity: 0.7;
}

.resource-value {
  width: 40px;
  text-align: right;
  flex-shrink: 0;
  font-size: 11px;
}
</style>
