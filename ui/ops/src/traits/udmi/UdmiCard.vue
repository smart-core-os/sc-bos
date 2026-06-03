<template>
  <v-card elevation="0" tile>
    <v-list tile class="ma-0 pa-0" lines="three" density="compact">
      <v-list-subheader class="text-title-caps-large text-neutral-lighten-3">UDMI Event</v-list-subheader>
      <v-list-item class="py-1 mb-2" v-if="message.updateTime">
        <v-list-item-title class="text-body-small text-capitalize">Last updated</v-list-item-title>
        <v-list-item-subtitle class="text-capitalize text-body-1">
          {{ Intl.DateTimeFormat('en-GB', {dateStyle: 'short', timeStyle: 'long'}).format(message.updateTime) }}
        </v-list-item-subtitle>
      </v-list-item>
      <v-list-item class="py-1 mb-2" v-if="message.value">
        <v-list-item-title class="text-body-small text-capitalize">Topic</v-list-item-title>
        <v-list-item-subtitle class="text-capitalize">{{ message.value?.topic }}</v-list-item-subtitle>
      </v-list-item>
      <v-list-item class="py-1" v-for="(value, key) in messagePayload" :key="key" lines="one">
        <v-list-item-title class="text-body-small text-capitalize flex-fill">{{ key }}</v-list-item-title>
        <template #append>
          <v-list-item-subtitle
              v-if="isStructured(displayValue(value))"
              class="text-end flex-fill text-body-2 udmi-json">
            <div class="d-flex align-center justify-end ga-1">
              <span v-if="collapsed[key]" class="text-medium-emphasis">
                {{ summaryOf(displayValue(value)) }}
              </span>
              <v-btn
                  size="x-small"
                  variant="text"
                  density="compact"
                  :icon="collapsed[key] ? 'mdi-chevron-right' : 'mdi-chevron-down'"
                  :aria-label="collapsed[key] ? 'Expand' : 'Collapse'"
                  @click="toggle(key)"/>
            </div>
            <pre v-if="!collapsed[key]" class="ma-0">{{ JSON.stringify(displayValue(value), null, 2) }}</pre>
          </v-list-item-subtitle>
          <v-list-item-subtitle v-else class="text-capitalize text-end flex-fill text-body-1">
            {{ displayValue(value) }}
          </v-list-item-subtitle>
        </template>
      </v-list-item>
      <v-progress-linear color="primary" indeterminate :active="message.loading || message.value === null"/>
    </v-list>
  </v-card>
</template>

<script setup>

import {closeResource, newResourceValue} from '@/api/resource';
import JSON5 from 'json5';
import {pullExportMessages} from '@/api/sc/traits/udmi';
import {useErrorStore} from '@/components/ui-error/error';
import {computed, onMounted, onUnmounted, reactive, watch} from 'vue';

const props = defineProps({
  // unique name of the device
  name: {
    type: String,
    default: ''
  }
});

const message = reactive(newResourceValue());

const messagePayload = computed(() => {
  if (message.value === null) return {};
  return JSON.parse(message.value.payload);
});

const displayValue = (value) => {
  let v = value?.present_value ?? value;
  if (typeof v === 'string') {
    const trimmed = v.trim();
    if (trimmed.startsWith('{') || trimmed.startsWith('[')) {
      try {
        return JSON5.parse(trimmed);
      } catch { /* not JSON(5), fall through */ }
    }
  }
  return v;
};

const isStructured = (v) => v !== null && typeof v === 'object';

const collapsed = reactive({});
const toggle = (key) => {
  collapsed[key] = !collapsed[key];
};
const summaryOf = (v) => {
  if (Array.isArray(v)) return `[…] (${v.length})`;
  const keys = Object.keys(v);
  return `{…} (${keys.length})`;
};

// UI error handling
const errorStore = useErrorStore();
let unwatchMessageError;
onMounted(() => {
  unwatchMessageError = errorStore.registerValue(message);
});
onUnmounted(() => {
  if (unwatchMessageError) unwatchMessageError();
});

watch(() => props.name, async (name) => {
  // close existing stream if present
  closeResource(message);
  // create new
  if (name && name !== '') {
    pullExportMessages({name, includeLast: true}, message);
  }
}, {immediate: true});

onUnmounted(() => {
  closeResource(message);
});

</script>

<style scoped>
.v-list-item {
  min-height: auto;
}
.udmi-json {
  -webkit-line-clamp: unset;
  -webkit-box-orient: unset;
  display: block;
  overflow: visible;
  text-overflow: clip;
  white-space: normal;
}
.udmi-json pre {
  white-space: pre-wrap;
  word-break: break-word;
  font-family: ui-monospace, SFMono-Regular, Menlo, Consolas, monospace;
  text-align: left;
  margin: 0;
}
</style>
