<template>
  <div class="logs-page d-flex flex-column fill-height">
    <v-row class="flex-grow-0 ma-0 pa-4 align-center" dense>
      <h3 class="text-h3">Logs</h3>
      <v-spacer/>
      <v-select
          v-model="selectedNode"
          :items="nodeNames"
          density="compact"
          hide-details
          label="Node"
          style="max-width: 300px"
          @update:model-value="onNodeChange"/>
    </v-row>

    <v-row class="flex-grow-0 ma-0 px-4 pb-2 align-center" dense>
      <v-select
          v-model="selectedLevel"
          :items="levelOptions"
          :disabled="!selectedNode || blockSystemEdit"
          density="compact"
          hide-details
          item-title="label"
          item-value="value"
          label="Log Level"
          style="max-width: 150px"
          @update:model-value="onLevelChange"/>
      <v-chip v-if="levelError" class="ml-2" color="error" size="small">{{ levelError }}</v-chip>
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
          placeholder="Search logs…"
          prepend-inner-icon="mdi-magnify"
          style="max-width: 500px"/>
      <v-btn class="ml-2" size="small" variant="tonal" @click="clearMessages">Clear</v-btn>
      <v-chip v-if="streamError" class="ml-2" color="error" size="small">{{ streamError.message }}</v-chip>
    </v-row>

    <log-viewport :messages="filteredMessages" class="flex-grow-1 mx-4 mb-4"/>
  </div>
</template>

<script setup>
import {getDownloadLogUrl, getLogLevel, pullLogLevel, pullLogMessages, pullLogMetadata, updateLogLevel} from '@/api/ui/log.js';
import {closeResource, newResourceValue} from '@/api/resource.js';
import {getService} from '@/api/ui/services.js';
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

// Only show nodes that have the log system service configured.
const nodesWithLog = ref(new Set());
watch(cohortNodes, async (nodes) => {
  const results = await Promise.all(
      nodes.map(n => n.name
          ? getService({name: n.name + '/systems', id: 'log'}).then(() => n.name).catch(() => null)
          : null
      )
  );
  nodesWithLog.value = new Set(results.filter(Boolean));
}, {immediate: true});

const nodeNames = computed(() =>
    cohortNodes.value.map(n => n.name).filter(n => n && nodesWithLog.value.has(n))
);

const selectedNode = ref(route.query.node ?? null);

const search = ref('');
const streamError = ref(null);
const metadata = ref(null);
const selectedLevel = ref(null);
const levelError = ref(null);
const downloadError = ref(null);
const messagesResource = reactive(newResourceValue());
const levelResource = reactive(newResourceValue());
const metadataResource = reactive(newResourceValue());

const {messages, clear: clearMessages} = useLogBuffer(messagesResource, {max: 2000});

const levelOptions = [
  {label: 'DEBUG', value: 1},
  {label: 'INFO', value: 2},
  {label: 'WARN', value: 3},
  {label: 'ERROR', value: 4}
];

const filteredMessages = computed(() => {
  if (!search.value) return messages.value;
  const q = search.value.toLowerCase();
  return messages.value.filter(m =>
      m.message?.toLowerCase().includes(q) ||
      m.logger?.toLowerCase().includes(q) ||
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
 * @param {string} name
 * @return {Promise<void>}
 */
async function startStreams(name) {
  cancelStreams();
  clearMessages();
  streamError.value = null;
  metadata.value = null;
  selectedLevel.value = null;
  levelError.value = null;

  pullLogMessages({name, initialCount: 200}, messagesResource);
  pullLogLevel({name}, levelResource);
  pullLogMetadata({name}, metadataResource);

  // Also seed level immediately
  try {
    const lvl = await getLogLevel({name});
    if (selectedNode.value === name && selectedLevel.value == null) {
      selectedLevel.value = lvl.level || null;
    }
  } catch {
    // level not supported — leave null
  }
}

/**
 * Cancels all active gRPC streams.
 */
function cancelStreams() {
  closeResource(messagesResource);
  closeResource(levelResource);
  closeResource(metadataResource);
}

watch(() => messagesResource.streamError, err => { streamError.value = err?.error ?? null; });

watch(() => levelResource.value, lvl => { if (lvl != null) selectedLevel.value = lvl; });
watch(() => levelResource.streamError, err => { levelError.value = err?.error?.message ?? null; });

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
 * @param {number} level
 * @return {Promise<void>}
 */
async function onLevelChange(level) {
  if (!selectedNode.value) return;
  try {
    levelError.value = null;
    await updateLogLevel({name: selectedNode.value, level});
  } catch (e) {
    levelError.value = e.message ?? 'Failed to set level';
  }
}

/**
 * @return {Promise<void>}
 */
async function downloadCurrent() {
  if (!selectedNode.value) return;
  downloadError.value = null;
  try {
    const res = await getDownloadLogUrl({name: selectedNode.value, includeRotated: false});
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
    const res = await getDownloadLogUrl({name: selectedNode.value, includeRotated: true});
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
.logs-page {
  height: 100%;
  overflow: hidden;
}
</style>
