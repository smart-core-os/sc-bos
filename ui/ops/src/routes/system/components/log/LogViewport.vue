<template>
  <div ref="logEl" class="log-viewport">
    <div
        v-for="line in lines"
        :key="line.msg._key"
        class="log-line">
      <span class="log-time">{{ line.time }}</span>
      <span v-if="line.msg.source" class="log-source" :title="line.msg.source">{{ line.msg.source }}</span>
      <span :class="['log-level', `text-${levelColor[line.msg.level] ?? 'white'}`]">{{ levelName[line.msg.level] ?? '?' }}</span>
      <!-- eslint-disable-next-line max-len -->
      <span class="log-logger"><template v-for="(p, i) in line.loggerParts" :key="i"><mark v-if="p.matchIndex != null" :class="{'log-match-active': p.matchIndex === activeMatch}">{{ p.text }}</mark><template v-else>{{ p.text }}</template></template>:</span>
      <!-- eslint-disable-next-line max-len -->
      <span class="log-msg"><template v-for="(p, i) in line.msgParts" :key="i"><mark v-if="p.matchIndex != null" :class="{'log-match-active': p.matchIndex === activeMatch}">{{ p.text }}</mark><template v-else>{{ p.text }}</template></template></span>
    </div>
  </div>
</template>

<script setup>
import {formatFields, formatTime, levelColor, levelName, splitHighlight} from '@/routes/system/components/log/format.js';
import {computed, nextTick, ref, watch} from 'vue';

const props = defineProps({
  messages: {
    type: Array,
    default: () => []
  },
  // Ctrl+F style search: highlights occurrences without filtering lines.
  search: {
    type: String,
    default: ''
  },
  // Global index of the currently focused match; scrolled into view on change.
  activeMatch: {
    type: Number,
    default: 0
  }
});

const emit = defineEmits(['match-count']);

const logEl = ref(null);

// Per-message split and time formatting are cached by _key so appends and
// buffer trims don't re-split every buffered line; the cache is rebuilt when
// the search term changes. Cached match indices are line-local and offset to
// global positions below.
let splitCache = new Map();
let splitCacheSearch = '';

/**
 * Returns parts with each matchIndex offset by base, reusing the original
 * array when there's nothing to renumber.
 *
 * @param {Array<{text: string, matchIndex?: number}>} parts
 * @param {number} base
 * @return {Array<{text: string, matchIndex?: number}>}
 */
function offsetParts(parts, base) {
  if (base === 0 || !parts.some(p => p.matchIndex != null)) return parts;
  return parts.map(p => p.matchIndex != null ? {...p, matchIndex: p.matchIndex + base} : p);
}

// Each line's logger and message text split into plain/highlighted parts,
// with match indices numbered globally across all lines for navigation.
const linesAndCount = computed(() => {
  if (props.search !== splitCacheSearch) {
    splitCache = new Map();
    splitCacheSearch = props.search;
  }
  const cache = splitCache;
  let idx = 0;
  const lines = props.messages.map(msg => {
    let entry = cache.get(msg._key);
    if (!entry) {
      entry = {
        time: formatTime(msg.timestamp),
        logger: splitHighlight(msg.logger ?? '', props.search),
        body: splitHighlight(`${msg.message ?? ''}${formatFields(msg.fieldsMap)}`, props.search)
      };
      cache.set(msg._key, entry);
    }
    const line = {
      msg,
      time: entry.time,
      loggerParts: offsetParts(entry.logger.parts, idx),
      msgParts: offsetParts(entry.body.parts, idx + entry.logger.count)
    };
    idx += entry.logger.count + entry.body.count;
    return line;
  });
  // drop cache entries for messages evicted from the buffer
  if (cache.size > props.messages.length + 1000) {
    const live = new Set(props.messages.map(m => m._key));
    for (const k of cache.keys()) {
      if (!live.has(k)) cache.delete(k);
    }
  }
  return {lines, count: idx};
});
const lines = computed(() => linesAndCount.value.lines);

watch(() => linesAndCount.value.count, count => emit('match-count', count), {immediate: true});

// Auto-scroll to the bottom when new messages arrive, unless the user has
// scrolled up to read older entries. Watch the last message's _key rather
// than the length so appends still trigger once the buffer is capped; _keys
// only ever increase, so a filtered list changing shape (e.g. the Logs page
// search) doesn't trigger a scroll, only genuinely new messages do.
watch(() => props.messages[props.messages.length - 1]?._key, (newKey, oldKey) => {
  if (newKey == null || (oldKey != null && newKey <= oldKey)) return; // not an append
  if (props.search) return; // don't fight match navigation while searching
  nextTick(() => {
    const el = logEl.value;
    if (!el) return;
    if (el.scrollHeight - el.scrollTop - el.clientHeight < 100) {
      el.scrollTop = el.scrollHeight;
    }
  });
});

// Bring the active match into view when it, or the search term, changes.
watch([() => props.activeMatch, () => props.search], () => {
  nextTick(() => {
    logEl.value?.querySelector('.log-match-active')?.scrollIntoView({block: 'nearest'});
  });
});
</script>

<style scoped>
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

/* source node, only shown for aggregated (e.g. gateway) log streams */
.log-source {
  color: #4ec9b0;
  flex-shrink: 0;
  font-weight: bold;
  max-width: 16ch;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
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

.log-line mark {
  background: #806e00;
  color: #fff;
  border-radius: 2px;
}

.log-line mark.log-match-active {
  background: #ff9632;
  color: #000;
}
</style>
