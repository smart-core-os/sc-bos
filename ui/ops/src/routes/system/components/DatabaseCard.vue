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
import PostgresNodeStatus from './PostgresNodeStatus.vue';
import {useCohortStore} from '@/stores/cohort.js';
import {storeToRefs} from 'pinia';
import {computed, reactive} from 'vue';

const {cohortNodes} = storeToRefs(useCohortStore());

const nodeHasPostgres = reactive({});
const hasAnyPostgres = computed(() => Object.values(nodeHasPostgres).some(Boolean));
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
</style>
