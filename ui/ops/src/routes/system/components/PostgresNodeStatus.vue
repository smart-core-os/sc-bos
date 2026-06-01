<template>
  <template v-if="!resource.streamError && resource.value">
    <div v-if="resource.value.bytes" class="pg-node-row">
      <v-icon size="10" color="success" class="pg-status-dot">mdi-circle</v-icon>
      <span class="pg-node-name" :title="node.name">{{ node.name }}</span>
      <span class="pg-stat">{{ bytesLabel }}</span>
    </div>
    <div v-else class="pg-node-row pg-node-row--error">
      <v-icon size="10" color="error" class="pg-status-dot">mdi-circle</v-icon>
      <span class="pg-node-name" :title="node.name">{{ node.name }}</span>
      <span class="pg-stat text-error">disconnected</span>
    </div>
  </template>
</template>

<script setup>
import {usePullDataRetention} from '@/traits/dataRetention/data-retention.js';
import {formatBytes} from '@/util/number.js';
import {computed, reactive, watch} from 'vue';

const props = defineProps({
  node: {type: Object, default: () => null}
});

const emit = defineEmits(['update:active']);

const resource = reactive(usePullDataRetention(
    computed(() => props.node.name + '/stores/postgres'),
    () => false
));

watch(
    () => !!resource.value && !resource.streamError,
    (active) => emit('update:active', active),
    {immediate: true}
);

const bytesLabel = computed(() => {
  if (!resource.value?.bytes) return '';
  const s = resource.value;
  const utilization = (s.bytes?.used != null && s.bytes?.capacity != null && s.bytes.capacity > 0)
      ? (s.bytes.used / s.bytes.capacity) * 100
      : null;
  const usedStr = formatBytes(s.bytes?.used);
  const totalStr = s.bytes?.capacity != null ? ` / ${formatBytes(s.bytes.capacity)}` : '';
  const pctStr = utilization != null ? ` (${utilization.toFixed(1)}%)` : '';
  return `${usedStr}${totalStr}${pctStr}`;
});
</script>

<style scoped>
.pg-node-row {
  display: flex;
  align-items: flex-start;
  gap: 5px;
  margin-bottom: 6px;
  font-size: 11px;
  position: relative;
}

.pg-status-dot {
  margin-top: 1px;
  flex-shrink: 0;
}

.pg-node-name {
  font-weight: 500;
  flex-shrink: 0;
  max-width: 90px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.pg-stat {
  opacity: 0.65;
  font-size: 10px;
  white-space: nowrap;
}
</style>
