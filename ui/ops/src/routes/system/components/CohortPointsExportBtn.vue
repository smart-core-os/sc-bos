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
    <v-snackbar v-model="showMessage" :color="messageColor" timeout="5000">
      {{ message }}
    </v-snackbar>
  </div>
</template>

<script setup>
import {useCohortStore} from '@/stores/cohort';
import {buildPointsCsv, collectCohortMessages} from '@/routes/automations/pointsExport';
import {dateStamp} from '@/util/date';
import {downloadCSVRows} from '@/util/downloadCSV';
import {storeToRefs} from 'pinia';
import {ref} from 'vue';

const {cohortNodes, serverNode} = storeToRefs(useCohortStore());

const loading = ref(false);
const showMessage = ref(false);
const message = ref('');
const messageColor = ref('warning');

/**
 * @param {string} text
 * @param {string} color - Vuetify colour token
 */
function notify(text, color) {
  message.value = text;
  messageColor.value = color;
  showMessage.value = true;
}

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
    const rows = buildPointsCsv(messages);
    // rows always contains the header row; a length of 1 means nothing was exported.
    if (rows.length <= 1) {
      notify('No points found to export.', 'info');
      return;
    }
    const label = (serverNode.value?.name || 'cohort').replaceAll('/', '_');
    downloadCSVRows(`points-list - ${label} - ${dateStamp()}.csv`, rows);
    if (errors.length > 0) {
      notify('Some data could not be retrieved; the points list may be incomplete.', 'warning');
    }
  } finally {
    loading.value = false;
  }
}
</script>

<style scoped>
</style>
