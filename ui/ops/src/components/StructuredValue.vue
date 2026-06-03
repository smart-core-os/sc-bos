<template>
  <div v-if="isStructured" class="structured-value">
    <div class="d-flex align-center justify-end ga-1">
      <span v-if="collapsed" class="text-medium-emphasis">{{ summary }}</span>
      <v-btn
          size="x-small"
          variant="text"
          density="compact"
          :icon="collapsed ? 'mdi-chevron-right' : 'mdi-chevron-down'"
          :aria-label="collapsed ? 'Expand' : 'Collapse'"
          @click="collapsed = !collapsed"/>
    </div>
    <pre v-if="!collapsed" class="ma-0">{{ JSON.stringify(displayValue, null, 2) }}</pre>
  </div>
  <template v-else>{{ displayValue }}</template>
</template>

<script setup>
import JSON5 from 'json5';
import {computed, ref} from 'vue';

// Renders a value, pretty printing it behind a collapsible toggle when it is - or parses as - an object or array.
// String values that look like JSON are parsed leniently, supporting JSON5 syntax (unquoted keys, single quotes,
// trailing commas, etc.) as well as strict JSON.
const props = defineProps({
  value: {
    type: [String, Number, Boolean, Object, Array],
    default: undefined
  }
});

// Leniently parse a string as JSON5 if it looks like a JSON object, array, or quoted string.
// Returns the original string unchanged when it is not parseable JSON.
const parseOnce = (v) => {
  if (typeof v !== 'string') return v;
  const trimmed = v.trim();
  if (trimmed.startsWith('{') || trimmed.startsWith('[') || trimmed.startsWith('"')) {
    try {
      return JSON5.parse(trimmed);
    } catch { /* not JSON(5), fall through */ }
  }
  return v;
};

// Recursively decode JSON that may be double-encoded - a JSON string nested inside another
// JSON string. The More map carries string values, so structured data often arrives as a
// quoted/escaped string (e.g. "{\"device\":1}"); without unwrapping it renders with escaped
// quotes. We re-parse string leaves until they stop changing, then recurse into objects/arrays.
const deepParse = (v) => {
  const parsed = parseOnce(v);
  if (parsed !== v) return deepParse(parsed); // unwrapped a layer (incl. a quoted string), keep going
  if (Array.isArray(v)) return v.map(deepParse);
  if (v !== null && typeof v === 'object') {
    return Object.fromEntries(Object.entries(v).map(([k, val]) => [k, deepParse(val)]));
  }
  return v;
};

const displayValue = computed(() => deepParse(props.value));

const isStructured = computed(() => displayValue.value !== null && typeof displayValue.value === 'object');

const collapsed = ref(true);

const summary = computed(() => {
  const v = displayValue.value;
  if (Array.isArray(v)) return `[…] (${v.length})`;
  return `{…} (${Object.keys(v).length})`;
});
</script>

<style scoped>
.structured-value pre {
  white-space: pre-wrap;
  word-break: break-word;
  font-family: ui-monospace, SFMono-Regular, Menlo, Consolas, monospace;
  text-align: left;
  margin: 0;
}
</style>