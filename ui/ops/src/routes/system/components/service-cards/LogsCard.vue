<template>
  <v-card flat tile>
    <v-card-subtitle class="text-subtitle-2 text-title-caps-large text-neutral-lighten-3 py-4">Logs</v-card-subtitle>
    <v-card-text class="px-4 pt-0">
      <v-btn
          block
          prepend-icon="mdi-text-long"
          variant="tonal"
          :disabled="!canViewLogs"
          @click="dialog = true">
        View Logs
      </v-btn>
    </v-card-text>

    <v-dialog v-model="dialog" max-width="1200px">
      <v-card>
        <v-card-title class="d-flex align-center">
          {{ dialogTitle }}
          <v-spacer/>
          <v-btn
              :disabled="!messages.length"
              icon="mdi-download"
              title="Export logs"
              variant="text"
              @click="exportLogs"/>
          <v-btn icon="mdi-close" variant="text" @click="dialog = false"/>
        </v-card-title>
        <v-card-text class="pb-4">
          <div v-if="unsupported" class="text-body-small text-neutral-lighten-4 py-2">
            Logs are not available on this node.
          </div>
          <div v-else-if="streamError && !messages.length" class="text-body-small text-error py-2">
            Error loading logs: {{ streamError }}
          </div>
          <div v-else-if="!messages.length" class="text-body-small text-neutral-lighten-4 py-2">
            No recent logs for this service.
          </div>
          <log-search-view v-else :messages="messages" class="log-view"/>
        </v-card-text>
      </v-card>
    </v-dialog>
  </v-card>
</template>

<script setup>
import {closeResource, newResourceValue} from '@/api/resource.js';
import {logFields, pullLogMessages} from '@/api/ui/log.js';
import {triggerTextDownload} from '@/components/download/download.js';
import LogSearchView from '@/routes/system/components/log/LogSearchView.vue';
import {messagesToText} from '@/routes/system/components/log/format.js';
import {useLogBuffer} from '@/routes/system/components/log/useLogBuffer.js';
import {useSidebarStore} from '@/stores/sidebar';
import {StatusCode} from 'grpc-web';
import {computed, onUnmounted, reactive, ref, watch} from 'vue';

const sidebar = useSidebarStore();

const serviceId = computed(() => sidebar.data?.service?.id);
const serviceType = computed(() => sidebar.data?.service?.type);
const nodeName = computed(() => sidebar.data?.nodeName);

const dialog = ref(false);
const unsupported = ref(false);
const streamError = ref(null);
const messagesResource = reactive(newResourceValue());

const {messages, clear: clearMessages} = useLogBuffer(messagesResource, {
  max: 500,
  onBatch: () => {
    // a (re)tried stream has recovered
    unsupported.value = false;
    streamError.value = null;
  }
});

const canViewLogs = computed(() => Boolean(nodeName.value && serviceId.value));
const dialogTitle = computed(() => serviceId.value ? `Logs - ${serviceId.value}` : 'Logs');

/**
 * Opens the log stream for the currently selected service.
 */
function startStream() {
  stopStream();
  if (!nodeName.value || !serviceId.value) return;
  pullLogMessages({
    name: nodeName.value,
    initialCount: 50,
    fieldFilter: {
      [logFields.serviceId]: serviceId.value,
      // only constrain the kind when we know it, an absent value would match nothing
      ...(serviceType.value && {[logFields.serviceKind]: serviceType.value})
    }
  }, messagesResource);
}

/**
 * Closes the log stream and resets the buffer and error state.
 */
function stopStream() {
  closeResource(messagesResource);
  clearMessages();
  unsupported.value = false;
  streamError.value = null;
}

// Nodes without the log system respond with UNIMPLEMENTED/NOT_FOUND; show a
// quiet message rather than an error. Other errors are shown as errors - the
// stream retries automatically and a successful batch clears them.
watch(() => messagesResource.streamError, (err) => {
  if (!err) return;
  const code = err.error?.code;
  if (code === StatusCode.UNIMPLEMENTED || code === StatusCode.NOT_FOUND) {
    unsupported.value = true;
  } else {
    streamError.value = err.error?.message ?? 'stream error';
  }
});

// Only stream while the dialog is open.
watch(dialog, (open) => {
  if (open) startStream();
  else stopStream();
});

// The sidebar component instance is reused as the user selects different
// services; close the dialog (stopping any stream) when the selection changes.
watch([serviceId, serviceType, nodeName], () => {
  dialog.value = false;
});
onUnmounted(stopStream);

/**
 * Downloads the currently buffered messages as a text file.
 */
function exportLogs() {
  triggerTextDownload(messagesToText(messages.value), `${serviceId.value ?? 'service'}-logs.txt`);
}
</script>

<style scoped>
.log-view {
  height: 70vh;
}
</style>
