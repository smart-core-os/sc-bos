<template>
  <div>
    <v-btn
        color="primary"
        variant="flat"
        prepend-icon="mdi-download"
        append-icon="mdi-menu-down"
        :loading="loading">
      Download points list
      <v-menu activator="parent" location="bottom end">
        <v-list density="compact">
          <v-list-item title="Per device" @click="download('device')"/>
          <v-list-item title="Per device type" @click="download('type')"/>
        </v-list>
      </v-menu>
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
 * results, and downloads them as a CSV points list in the given grouping mode.
 *
 * @param {'device'|'type'} mode
 * @return {Promise<void>}
 */
async function download(mode) {
  loading.value = true;
  try {
    const {messages, errors} = await collectCohortMessages(cohortNodes.value);
    showIncomplete.value = errors.length > 0;
    const rows = buildPointsCsv(messages, mode);
    const dateString = new Date().toISOString().slice(0, 10);
    const kind = mode === 'type' ? 'points-list-by-type' : 'points-list';
    downloadCSVRows(`${kind} - building - ${dateString}.csv`, rows);
  } finally {
    loading.value = false;
  }
}
</script>

<style scoped>
</style>
