<template>
  <v-dialog :model-value="modelValue" max-width="800px" scrollable @update:model-value="emit('update:modelValue', $event)">
    <v-card v-if="modelValue">
      <v-card-title class="d-flex align-center">
        Live Logs – {{ name }}
        <v-spacer/>
        <v-btn icon="mdi-close" variant="text" @click="emit('update:modelValue', false)"/>
      </v-card-title>
      <v-card-text class="pa-0">
        <div ref="logEl" class="log-viewport">
          <div
              v-for="msg in messages"
              :key="msg._key"
              class="log-line">
            <span class="log-time">{{ formatTime(msg.timestamp) }}</span>
            <span :class="['log-level', `text-${levelColor[msg.level] ?? 'white'}`]">{{ levelName[msg.level] ?? '?' }}</span>
            <span class="log-logger">{{ msg.logger }}:</span>
            <span class="log-msg">{{ msg.message }}</span>
          </div>
        </div>
      </v-card-text>
      <v-card-actions>
        <v-btn @click="messages = []">Clear</v-btn>
        <v-btn prepend-icon="mdi-download" @click="downloadLogs">Download</v-btn>
        <v-spacer/>
        <v-chip v-if="streamError" color="error" size="small">{{ streamError.message }}</v-chip>
      </v-card-actions>
    </v-card>
  </v-dialog>
</template>

<script setup>
import {grpcWebEndpoint} from '@/api/config.js';
import {getDownloadLogUrl, pullLogMessages} from '@/api/ui/log.js';
import {triggerDownloadFromUrl} from '@/components/download/download.js';
import {nextTick, onUnmounted, ref, watch} from 'vue';

const props = defineProps({
  modelValue: {type: Boolean, default: false},
  name: {type: String, default: ''}
});
const emit = defineEmits(['update:modelValue']);

const messages = ref([]);
const streamError = ref(null);
const logEl = ref(null);
let stream = null;
let _keyCounter = 0;

const levelColor = {
  1: 'grey',
  2: 'blue',
  3: 'amber',
  4: 'red',
  5: 'red',
  6: 'red',
  7: 'red'
};
const levelName = {
  1: 'DBG',
  2: 'INF',
  3: 'WRN',
  4: 'ERR',
  5: 'DPK',
  6: 'PNC',
  7: 'FTL'
};

function formatTime(timestamp) {
  if (!timestamp) return '--:--:--';
  return new Date(timestamp.seconds * 1000).toLocaleTimeString();
}

async function startStream() {
  messages.value = [];
  streamError.value = null;
  const endpoint = await grpcWebEndpoint();
  stream = pullLogMessages(
      endpoint,
      {name: props.name, initialCount: 200},
      (newMsgs) => {
        for (const m of newMsgs) {
          m._key = _keyCounter++;
          messages.value.push(m);
        }
        if (messages.value.length > 2000) {
          messages.value.splice(0, messages.value.length - 2000);
        }
      },
      (err) => {
        streamError.value = err;
      }
  );
}

function cancelStream() {
  if (stream) {
    stream.cancel();
    stream = null;
  }
}

watch(() => props.modelValue, (open) => {
  if (open) {
    startStream();
  } else {
    cancelStream();
  }
}, {immediate: true});

// Auto-scroll when near bottom
watch(messages, () => {
  nextTick(() => {
    if (!logEl.value) return;
    const el = logEl.value;
    if (el.scrollHeight - el.scrollTop - el.clientHeight < 100) {
      el.scrollTop = el.scrollHeight;
    }
  });
}, {deep: false});

async function downloadLogs() {
  const res = await getDownloadLogUrl({name: props.name, includeRotated: true});
  for (const file of res.filesList ?? []) {
    triggerDownloadFromUrl(file.url, file.filename);
  }
}

onUnmounted(() => cancelStream());
</script>

<style scoped>
.log-viewport {
  height: 500px;
  overflow-y: auto;
  background: #1e1e1e;
  font-family: monospace;
  font-size: 12px;
  padding: 8px;
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
  width: 3ch;
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
