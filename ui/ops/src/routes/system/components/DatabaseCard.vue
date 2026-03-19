<template>
  <v-card v-show="hasAnyPostgres" width="260px" class="ma-2 postgres-card">
    <div class="postgres-card__accent"/>
    <v-card-text class="pa-2 pb-2" style="position: relative; overflow: hidden;">

      <!-- Watermark icon -->
      <v-icon class="postgres-card__watermark">mdi-database</v-icon>

      <!-- Header -->
      <div class="d-flex align-center" style="position: relative; gap: 6px;">
        <span class="text-subtitle-2 font-weight-bold">PostgreSQL</span>
        <v-chip size="x-small" variant="flat" color="#336791" class="flex-shrink-0">db</v-chip>
      </div>

      <v-divider class="mt-1 mb-2"/>

      <!-- Per-node status rows -->
      <postgres-node-status
          v-for="cohortNode in cohortNodes"
          :key="cohortNode.name"
          :node="cohortNode"
          @update:active="nodeHasPostgres[cohortNode.name] = $event"/>

    </v-card-text>
  </v-card>
</template>

<script setup>
import {usePullStorage} from '@/traits/storage/storage.js';
import {useCohortStore} from '@/stores/cohort.js';
import {storeToRefs} from 'pinia';
import {computed, defineComponent, h, reactive, watch} from 'vue';

const {cohortNodes} = storeToRefs(useCohortStore());

const nodeHasPostgres = reactive({});
const hasAnyPostgres = computed(() => Object.values(nodeHasPostgres).some(Boolean));

const formatBytes = (n) => {
  if (n == null) return '—';
  if (n < 1024) return `${n} B`;
  if (n < 1024 ** 2) return `${(n / 1024).toFixed(1)} KB`;
  if (n < 1024 ** 3) return `${(n / 1024 ** 2).toFixed(1)} MB`;
  return `${(n / 1024 ** 3).toFixed(2)} GB`;
};

const PostgresNodeStatus = defineComponent({
  props: {
    node: {type: Object, default: () => null}
  },
  emits: ['update:active'],
  setup(props, {emit}) {
    const resource = reactive(usePullStorage(
        computed(() => props.node.name + '/stores/postgres'),
        () => false
    ));

    watch(
        () => !!resource.value && !resource.streamError,
        (active) => emit('update:active', active),
        {immediate: true}
    );

    return () => {
      if (resource.streamError || !resource.value) return null;

      if (resource.value.bytes) {
        const s = resource.value;
        const usedStr = formatBytes(s.bytes?.used);
        const totalStr = s.bytes?.capacity != null ? ` / ${formatBytes(s.bytes.capacity)}` : '';
        const pctStr = s.bytes?.utilization != null ? ` (${s.bytes.utilization.toFixed(1)}%)` : '';
        const bytesLabel = `${usedStr}${totalStr}${pctStr}`;

        const itemsLabel = s.items?.used != null
            ? `${s.items.used.toLocaleString()}${s.items.capacity != null ? ` / ${s.items.capacity.toLocaleString()}` : ''} rows`
            : null;

        return h('div', {class: 'pg-node-row'}, [
          h('v-icon', {size: 10, color: 'success', class: 'pg-status-dot'}, () => 'mdi-circle'),
          h('span', {class: 'pg-node-name', title: props.node.name}, props.node.name),
          h('div', {class: 'pg-node-stats'}, [
            h('span', {class: 'pg-stat'}, bytesLabel),
            itemsLabel ? h('span', {class: 'pg-stat'}, itemsLabel) : null
          ])
        ]);
      }

      return h('div', {class: 'pg-node-row pg-node-row--error'}, [
        h('v-icon', {size: 10, color: 'error', class: 'pg-status-dot'}, () => 'mdi-circle'),
        h('span', {class: 'pg-node-name', title: props.node.name}, props.node.name),
        h('span', {class: 'pg-stat text-error'}, 'disconnected')
      ]);
    };
  }
});
</script>

<style scoped>
.postgres-card__accent {
  height: 3px;
  width: 100%;
  background: #336791;
}

.postgres-card__watermark {
  position: absolute;
  bottom: -12px;
  right: -8px;
  font-size: 96px !important;
  opacity: 0.04;
  color: #336791 !important;
  pointer-events: none;
  user-select: none;
}

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

.pg-node-stats {
  display: flex;
  flex-direction: column;
  gap: 1px;
  min-width: 0;
}

.pg-stat {
  opacity: 0.65;
  font-size: 10px;
  white-space: nowrap;
}
</style>
