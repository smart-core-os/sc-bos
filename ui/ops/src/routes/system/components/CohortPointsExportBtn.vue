<template>
  <div>
    <v-btn
        class="mt-2"
        icon="mdi-format-list-bulleted"
        size="small"
        :loading="loading"
        v-tooltip:bottom="'Download points list'"
        @click="download">
      <v-icon size="24"/>
    </v-btn>
    <v-snackbar v-model="showIncomplete" color="warning" timeout="5000">
      Some nodes could not be reached; the points list may be incomplete.
    </v-snackbar>
  </div>
</template>

<script setup>
import {useCohortStore} from '@/stores/cohort';
import {buildPointsCsv, collectCohortMessages} from '@/routes/automations/pointsExport';
import {downloadCSVRows} from '@/util/downloadCSV';
import {storeToRefs} from 'pinia';
import {ref} from 'vue';

const {cohortNodes} = storeToRefs(useCohortStore());

const loading = ref(false);
const showIncomplete = ref(false);

/**
 * Fans out ListExportedPoints across every udmi automation in the cohort, merges the
 * results, and downloads them as a CSV points list (one row per device).
 *
 * @return {Promise<void>}
 */
async function download() {
  loading.value = true;
  try {
    const {messages, errors} = await collectCohortMessages(cohortNodes.value);
    showIncomplete.value = errors.length > 0;
    const rows = buildPointsCsv(messages);
    const dateString = new Date().toISOString().slice(0, 10);
    downloadCSVRows(`points-list - building - ${dateString}.csv`, rows);
  } finally {
    loading.value = false;
  }
}
</script>

<style scoped>
</style>
