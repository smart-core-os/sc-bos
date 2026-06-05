<template>
  <v-card flat tile>
    <v-card-subtitle class="text-subtitle-2 text-title-caps-large text-neutral-lighten-3 py-4">Config</v-card-subtitle>
    <v-card-text class="px-4 pt-0">
      <v-alert
          v-if="serviceError && !hasConfig"
          type="error"
          variant="tonal"
          class="mb-3 text-body-small"
          :text="serviceError"/>
      <!-- viewing config is read-only, so isn't gated on blockSystemEdit, only copying is -->
      <v-btn
          v-else
          block
          prepend-icon="mdi-code-json"
          variant="tonal"
          :disabled="!hasConfig"
          @click="dialog = true">
        View Config
      </v-btn>
      <v-snackbar v-model="copyConfirm" timeout="2000" color="success" max-width="250" min-width="200">
        <span class="text-body-large align-baseline"><v-icon start>mdi-check-circle</v-icon>Config copied</span>
      </v-snackbar>
    </v-card-text>

    <v-dialog v-model="dialog" max-width="800px" scrollable>
      <v-card>
        <v-card-title class="d-flex align-center">
          {{ dialogTitle }}
          <v-spacer/>
          <v-btn icon="mdi-content-copy" variant="text" title="Copy config" :disabled="blockSystemEdit" @click="copyConfig"/>
          <v-btn icon="mdi-close" variant="text" @click="dialog = false"/>
        </v-card-title>
        <v-card-text>
          <v-textarea
              v-model="config"
              auto-grow
              class="text-body-code"
              :error-messages="jsonError"
              variant="outlined"
              hide-details="auto"
              readonly/>
        </v-card-text>
      </v-card>
    </v-dialog>
  </v-card>
</template>

<script setup>
import useAuthSetup from '@/composables/useAuthSetup';
import {useSidebarStore} from '@/stores/sidebar';
import {computed, ref, watch} from 'vue';

const {blockSystemEdit} = useAuthSetup();

const sidebar = useSidebarStore();

const dialog = ref(false);
const jsonError = ref('');

const serviceError = computed(() => sidebar.data?.service?.error ?? '');
const hasConfig = computed(() => Boolean(sidebar.data?.service?.configRaw));
const serviceId = computed(() => sidebar.data?.service?.id);
const dialogTitle = computed(() => serviceId.value ? `Config - ${serviceId.value}` : 'Config');

// Close the dialog if the selected service changes while it's open.
watch(serviceId, () => {
  dialog.value = false;
});

const config = computed({
  get() {
    return sidebar.data.service?.configRaw ?? '';
  },
  set(value) {
    jsonError.value = '';
    try {
      sidebar.data.config = JSON.parse(value);
      /**
       * @param {Error} e
       */
    } catch (e) {
      jsonError.value = 'JSON error: ' + e.message;
    }
  }
});

const copyConfirm = ref(false);

/**
 *
 */
function copyConfig() {
  navigator.clipboard.writeText(config.value);
  copyConfirm.value = true;
}
</script>
