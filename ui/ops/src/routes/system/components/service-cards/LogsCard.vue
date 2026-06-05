<template>
  <v-card flat tile>
    <v-card-title class="text-subtitle-2 text-title-caps-large text-neutral-lighten-3 d-flex align-center">
      Logs
      <v-spacer/>
      <v-btn
          :disabled="!messages.length"
          icon="mdi-arrow-expand"
          size="small"
          title="Pop out logs"
          variant="text"
          @click="popout = true"/>
      <v-btn
          :disabled="!messages.length"
          icon="mdi-download"
          size="small"
          title="Export logs"
          variant="text"
          @click="exportLogs"/>
      <v-btn
          :icon="expanded ? 'mdi-chevron-up' : 'mdi-chevron-down'"
          size="small"
          variant="text"
          @click="expanded = !expanded"/>
    </v-card-title>
    <v-expand-transition>
      <!-- v-if rather than v-show so a collapsed card doesn't keep re-rendering the stream -->
      <v-card-text v-if="expanded" class="px-4 pt-0 pb-3">
        <div v-if="unsupported" class="text-body-small text-neutral-lighten-4 py-2">
          Logs are not available on this node.
        </div>
        <div v-else-if="streamError && !messages.length" class="text-body-small text-error py-2">
          Error loading logs: {{ streamError }}
        </div>
        <div v-else-if="!messages.length" class="text-body-small text-neutral-lighten-4 py-2">
          No recent logs for this service.
        </div>
        <log-search-view v-else :messages="messages" class="card-log-view"/>
      </v-card-text>
    </v-expand-transition>

    <v-dialog v-model="popout" max-width="1200px">
      <v-card>
        <v-card-title class="d-flex align-center">
          {{ popoutTitle }}
          <v-spacer/>
          <v-btn
              :disabled="!messages.length"
              icon="mdi-download"
              title="Export logs"
              variant="text"
              @click="exportLogs"/>
          <v-btn icon="mdi-close" variant="text" @click="popout = false"/>
        </v-card-title>
        <v-card-text class="pb-4">
          <log-search-view :messages="messages" class="popout-log-view"/>
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
import {computed, onUnmounted, reactive, ref, watch} from 'vue';

const sidebar = useSidebarStore();

const serviceId = computed(() => sidebar.data?.service?.id);
const serviceType = computed(() => sidebar.data?.service?.type);
const nodeName = computed(() => sidebar.data?.nodeName);

const expanded = ref(true);
const popout = ref(false);
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

const popoutTitle = computed(() => serviceId.value ? `Logs - ${serviceId.value}` : 'Logs');

/**
 * (Re-)opens the log stream for the currently selected service.
 */
function startStream() {
  closeResource(messagesResource);
  clearMessages();
  unsupported.value = false;
  streamError.value = null;
  popout.value = false;
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

// Nodes without the log system respond with UNIMPLEMENTED/NOT_FOUND; show a
// quiet message rather than an error. Other errors are shown as errors - the
// stream retries automatically and a successful batch clears them.
watch(() => messagesResource.streamError, (err) => {
  if (!err) return;
  const code = err.error?.code;
  if (code === 12 /* UNIMPLEMENTED */ || code === 5 /* NOT_FOUND */) {
    unsupported.value = true;
  } else {
    streamError.value = err.error?.message ?? 'stream error';
  }
});

// The sidebar component instance is reused as the user selects different
// services, so re-stream whenever the selection changes.
watch([serviceId, serviceType, nodeName], startStream, {immediate: true});
onUnmounted(() => closeResource(messagesResource));

/**
 * Downloads the currently buffered messages as a text file.
 */
function exportLogs() {
  triggerTextDownload(messagesToText(messages.value), `${serviceId.value ?? 'service'}-logs.txt`);
}
</script>

<style scoped>
.card-log-view {
  max-height: 440px;
}

.popout-log-view {
  height: 70vh;
}
</style>
