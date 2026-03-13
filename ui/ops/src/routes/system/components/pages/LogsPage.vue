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
          :disabled="!selectedNode"
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
      <v-btn
          :disabled="!selectedNode"
          class="mr-2"
          prepend-icon="mdi-download"
          size="small"
          variant="tonal"
          @click="downloadCurrent">
        Download Current
      </v-btn>
      <v-btn
          :disabled="!selectedNode"
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
      <v-btn class="ml-2" size="small" variant="tonal" @click="messages = []">Clear</v-btn>
      <v-chip v-if="streamError" class="ml-2" color="error" size="small">{{ streamError.message }}</v-chip>
    </v-row>

    <div ref="logEl" class="log-viewport flex-grow-1 mx-4 mb-4">
      <div
          v-for="msg in filteredMessages"
          :key="msg._key"
          class="log-line">
        <span class="log-time">{{ formatTime(msg.timestamp) }}</span>
        <span :class="['log-level', `text-${levelColor[msg.level] ?? 'white'}`]">{{ levelName[msg.level] ?? '?' }}</span>
        <span class="log-logger">{{ msg.logger }}:</span>
        <span class="log-msg">{{ msg.message }}{{ formatFields(msg.fieldsMap) }}</span>
      </div>
    </div>
  </div>
</template>

<script setup>
import {grpcWebEndpoint} from '@/api/config.js';
import {getDownloadLogUrl, getLogLevel, pullLogLevel, pullLogMessages, pullLogMetadata, updateLogLevel} from '@/api/ui/log.js';
import {getService} from '@/api/ui/services.js';
import {triggerDownloadFromUrl} from '@/components/download/download.js';
import {useCohortStore} from '@/stores/cohort.js';
import {storeToRefs} from 'pinia';
import {computed, nextTick, onUnmounted, ref, watch} from 'vue';
import {useRoute, useRouter} from 'vue-router';

const route = useRoute();
const router = useRouter();

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

const messages = ref([]);
const search = ref('');
const streamError = ref(null);
const logEl = ref(null);
const metadata = ref(null);
const selectedLevel = ref(null);
const levelError = ref(null);
let _keyCounter = 0;
let stream = null;
let levelStream = null;
let metadataStream = null;

const levelOptions = [
  {label: 'DEBUG', value: 1},
  {label: 'INFO', value: 2},
  {label: 'WARN', value: 3},
  {label: 'ERROR', value: 4}
];

const levelColor = {1: 'grey', 2: 'blue', 3: 'amber', 4: 'red', 5: 'red', 6: 'red', 7: 'red'};
const levelName = {1: 'DBG', 2: 'INF', 3: 'WRN', 4: 'ERR', 5: 'DPK', 6: 'PNC', 7: 'FTL'};

const filteredMessages = computed(() => {
  if (!search.value) return messages.value;
  const q = search.value.toLowerCase();
  return messages.value.filter(m =>
      m.message?.toLowerCase().includes(q) ||
      m.logger?.toLowerCase().includes(q) ||
      levelName[m.level]?.toLowerCase().includes(q)
  );
});

function formatFields(fieldsMap) {
  if (!fieldsMap?.length) return '';
  return '\t' + JSON.stringify(Object.fromEntries(fieldsMap));
}

function formatTime(timestamp) {
  if (!timestamp) return '--:--:--';
  return new Date(timestamp.seconds * 1000).toLocaleTimeString();
}

function formatBytes(bytes) {
  if (bytes == null) return '';
  if (bytes < 1024) return bytes + ' B';
  if (bytes < 1024 * 1024) return (bytes / 1024).toFixed(1) + ' KB';
  return (bytes / (1024 * 1024)).toFixed(1) + ' MB';
}

async function startStreams(name) {
  cancelStreams();
  messages.value = [];
  streamError.value = null;
  metadata.value = null;
  selectedLevel.value = null;
  levelError.value = null;

  const endpoint = await grpcWebEndpoint();

  stream = pullLogMessages(
      endpoint,
      {name, initialCount: 200},
      (newMsgs) => {
        for (const m of newMsgs) {
          m._key = _keyCounter++;
          messages.value.push(m);
        }
        if (messages.value.length > 2000) {
          messages.value.splice(0, messages.value.length - 2000);
        }
      },
      (err) => { streamError.value = err; }
  );

  levelStream = pullLogLevel(
      endpoint,
      {name},
      (lvl) => { selectedLevel.value = lvl.level; },
      (err) => { levelError.value = err.message; }
  );

  metadataStream = pullLogMetadata(
      endpoint,
      {name},
      (md) => { metadata.value = md; },
      null
  );

  // Also seed level immediately
  try {
    const lvl = await getLogLevel({name});
    if (selectedLevel.value == null) {
      selectedLevel.value = lvl.level;
    }
  } catch {
    // level not supported — leave null
  }
}

function cancelStreams() {
  stream?.cancel();
  stream = null;
  levelStream?.cancel();
  levelStream = null;
  metadataStream?.cancel();
  metadataStream = null;
}

watch(selectedNode, (name) => {
  if (name) {
    startStreams(name);
  } else {
    cancelStreams();
  }
}, {immediate: true});

watch(messages, () => {
  nextTick(() => {
    if (!logEl.value) return;
    const el = logEl.value;
    if (el.scrollHeight - el.scrollTop - el.clientHeight < 100) {
      el.scrollTop = el.scrollHeight;
    }
  });
}, {deep: false});

function onNodeChange(name) {
  router.replace({query: name ? {node: name} : {}});
}

async function onLevelChange(level) {
  if (!selectedNode.value) return;
  try {
    levelError.value = null;
    await updateLogLevel({name: selectedNode.value, level});
  } catch (e) {
    levelError.value = e.message ?? 'Failed to set level';
  }
}

async function downloadCurrent() {
  if (!selectedNode.value) return;
  try {
    const res = await getDownloadLogUrl({name: selectedNode.value, includeRotated: false});
    for (const file of res.filesList ?? []) {
      triggerDownloadFromUrl(file.url, file.filename);
    }
  } catch (e) {
    console.warn('Failed to download current log', e);
  }
}

async function downloadAll() {
  if (!selectedNode.value) return;
  try {
    const res = await getDownloadLogUrl({name: selectedNode.value, includeRotated: true});
    for (const file of res.filesList ?? []) {
      triggerDownloadFromUrl(file.url, file.filename);
    }
  } catch (e) {
    console.warn('Failed to download all logs', e);
  }
}

onUnmounted(() => cancelStreams());
</script>

<style scoped>
.logs-page {
  height: 100%;
  overflow: hidden;
}

.log-viewport {
  overflow-y: auto;
  background: #1e1e1e;
  font-family: monospace;
  font-size: 12px;
  padding: 8px;
  border-radius: 4px;
}

.log-line {
  display: flex;
  gap: 6px;
  line-height: 1.4;
  white-space: pre-wrap;
  word-break: break-all;
}

.log-time {
  color: #888;
  flex-shrink: 0;
}

.log-level {
  flex-shrink: 0;
  width: 4ch;
  font-weight: bold;
}

.log-logger {
  color: #aaa;
  flex-shrink: 0;
}

.log-msg {
  color: #eee;
}
</style>
