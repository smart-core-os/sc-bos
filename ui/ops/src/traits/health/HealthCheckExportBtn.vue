<template>
  <div>
    <v-btn
        icon="mdi-file-download"
        size="small"
        :loading="loading"
        v-tooltip:bottom="'Export health checks as CSV'"
        @click="download">
      <v-icon size="24"/>
    </v-btn>
    <v-snackbar v-model="showMessage" :color="messageColor" timeout="5000">
      {{ message }}
    </v-snackbar>
  </div>
</template>

<script setup>
import {downloadHealthCheckCsv} from '@/traits/health/healthCheckExport.js';
import {ref} from 'vue';

const props = defineProps({
  /**
   * The device query to export, i.e. `devices.query.value` from the table's useDevices.
   * Covers the table's selection, trait, search and filter chips.
   */
  query: {
    type: Object, // of Device.Query.AsObject
    default: null
  }
});

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
 * Pages every device matching the query and downloads their health checks as a CSV,
 * one row per check.
 *
 * @return {Promise<void>}
 */
async function download() {
  loading.value = true;
  try {
    const {rowCount, error} = await downloadHealthCheckCsv(props.query);
    if (error) {
      // Nothing was downloaded at all if the very first page failed, so say which happened.
      notify(rowCount === 0 ?
        'Health checks could not be retrieved; nothing was exported.' :
        'Some devices could not be retrieved; the export may be incomplete.', 'warning');
    } else if (rowCount === 0) {
      notify('No health checks to export.', 'info');
    }
  } finally {
    loading.value = false;
  }
}
</script>

<style scoped>
</style>
