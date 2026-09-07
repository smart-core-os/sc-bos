<template>
  <div class="audit-logs-page d-flex flex-column fill-height">
    <v-row class="flex-grow-0 ma-0 pa-4 align-center" dense>
      <h3 class="text-h3">Audit Logs</h3>
      <v-spacer/>
      <v-select
          v-model="selectedNode"
          :items="nodeItems"
          density="compact"
          hide-details
          label="Node"
          style="max-width: 300px"
          @update:model-value="onNodeChange"/>
    </v-row>

    <v-row class="flex-grow-0 ma-0 px-4 pb-2 align-center" dense>
      <v-spacer/>
      <span v-if="metadata" class="text-body-2 text-medium-emphasis mr-4">
        {{ metadata.fileCount }} {{ metadata.fileCount === 1 ? 'file' : 'files' }} · {{ formatBytes(metadata.totalSizeBytes) }}
      </span>
      <v-chip v-if="downloadError" class="mr-2" color="error" size="small">{{ downloadError }}</v-chip>
      <v-btn
          :disabled="!selectedNode || blockSystemEdit"
          class="mr-2"
          prepend-icon="mdi-download"
          size="small"
          variant="tonal"
          @click="downloadCurrent">
        Download Current
      </v-btn>
      <v-btn
          :disabled="!selectedNode || blockSystemEdit"
          prepend-icon="mdi-download-multiple"
          size="small"
          variant="tonal"
          @click="downloadAll">
        Download All
      </v-btn>
    </v-row>

    <v-row class="flex-grow-0 ma-0 px-4 pb-2" dense>
      <v-text-field
          v-model="search"
          clearable
          density="compact"
          hide-details
          placeholder="Search audit log…"
          prepend-inner-icon="mdi-magnify"
          style="max-width: 500px"/>
      <v-btn class="ml-2" size="small" variant="tonal" @click="clearMessages">Clear</v-btn>
      <v-chip v-if="streamError" class="ml-2" color="error" size="small">{{ streamError.message }}</v-chip>
    </v-row>

    <log-viewport :messages="filteredMessages" class="flex-grow-1 mx-4 mb-4"/>
  </div>
</template>

<script setup>
import {getDownloadLogUrl, getLogMetadata, pullLogMessages, pullLogMetadata} from '@/api/ui/log.js';
import {closeResource, newResourceValue} from '@/api/resource.js';
import {triggerDownloadFromUrl} from '@/components/download/download.js';
import useAuthSetup from '@/composables/useAuthSetup.js';
import LogViewport from '@/routes/system/components/log/LogViewport.vue';
import {levelName} from '@/routes/system/components/log/format.js';
import {useLogBuffer} from '@/routes/system/components/log/useLogBuffer.js';
import {useCohortStore} from '@/stores/cohort.js';
import {storeToRefs} from 'pinia';
import {computed, onUnmounted, reactive, ref, watch} from 'vue';
import {useRoute, useRouter} from 'vue-router';

const route = useRoute();
const router = useRouter();
const {blockSystemEdit} = useAuthSetup();

const {cohortNodes} = storeToRefs(useCohortStore());

// The audit log is exposed via the standard Log trait on a per-node device named
// "<node>/audit-log" (see setupAuditLog in pkg/app/controller.go). All requests
// route to that device by name against the connected server endpoint.
const auditName = (node) => `${node}/audit-log`;

// Only show nodes that expose an audit-log device. Audit is not a system service,
// so we probe the device directly and keep the nodes whose metadata resolves.
const nodesWithAudit = ref(new Set());
watch(cohortNodes, async (nodes) => {
  const results = await Promise.all(
      nodes.map(n => n.name
          ? getLogMetadata({name: auditName(n.name)}).then(() => n.name).catch(() => null)
          : null
      )
  );
  nodesWithAudit.value = new Set(results.filter(Boolean));
}, {immediate: true});

const nodeNames = computed(() =>
    cohortNodes.value.map(n => n.name).filter(n => n && nodesWithAudit.value.has(n))
);

const nodeItems = computed(() => nodeNames.value.map(n => ({title: n, value: n})));

const selectedNode = ref(route.query.node ?? null);

const search = ref('');
const streamError = ref(null);
const metadata = ref(null);
const downloadError = ref(null);
const messagesResource = reactive(newResourceValue());
const metadataResource = reactive(newResourceValue());

const {messages, clear: clearMessages} = useLogBuffer(messagesResource, {max: 2000});

const filteredMessages = computed(() => {
  if (!search.value) return messages.value;
  const q = search.value.toLowerCase();
  return messages.value.filter(m =>
      m.message?.toLowerCase().includes(q) ||
      m.logger?.toLowerCase().includes(q) ||
      m.source?.toLowerCase().includes(q) ||
      levelName[m.level]?.toLowerCase().includes(q)
  );
});

/**
 * @param {number|null} bytes
 * @return {string}
 */
function formatBytes(bytes) {
  if (bytes == null) return '';
  if (bytes < 1024) return bytes + ' B';
  if (bytes < 1024 * 1024) return (bytes / 1024).toFixed(1) + ' KB';
  return (bytes / (1024 * 1024)).toFixed(1) + ' MB';
}

/**
 * @param {string} node
 * @return {Promise<void>}
 */
async function startStreams(node) {
  cancelStreams();
  clearMessages();
  streamError.value = null;
  metadata.value = null;

  const name = auditName(node);
  pullLogMessages({name, initialCount: 200}, messagesResource);
  pullLogMetadata({name}, metadataResource);
}

/**
 * Cancels all active gRPC streams.
 */
function cancelStreams() {
  closeResource(messagesResource);
  closeResource(metadataResource);
}

watch(() => messagesResource.streamError, err => { streamError.value = err?.error ?? null; });

watch(() => metadataResource.value, md => { if (md) metadata.value = md; });

watch(selectedNode, (name) => {
  if (name) {
    startStreams(name);
  } else {
    cancelStreams();
  }
}, {immediate: true});

/**
 * @param {string|null} name
 */
function onNodeChange(name) {
  router.replace({query: name ? {node: name} : {}});
}

/**
 * @return {Promise<void>}
 */
async function downloadCurrent() {
  if (!selectedNode.value) return;
  downloadError.value = null;
  try {
    const res = await getDownloadLogUrl({name: auditName(selectedNode.value), includeRotated: false});
    for (const file of res.filesList ?? []) {
      triggerDownloadFromUrl(file.url, file.filename);
    }
  } catch (e) {
    downloadError.value = e.message ?? 'Download failed';
  }
}

/**
 * @return {Promise<void>}
 */
async function downloadAll() {
  if (!selectedNode.value) return;
  downloadError.value = null;
  try {
    const res = await getDownloadLogUrl({name: auditName(selectedNode.value), includeRotated: true});
    for (const file of res.filesList ?? []) {
      triggerDownloadFromUrl(file.url, file.filename);
    }
  } catch (e) {
    downloadError.value = e.message ?? 'Download failed';
  }
}

onUnmounted(() => cancelStreams());
</script>

<style scoped>
.audit-logs-page {
  height: 100%;
  overflow: hidden;
}
</style>
