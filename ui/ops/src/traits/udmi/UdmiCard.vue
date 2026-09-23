<template>
  <v-card elevation="0" tile>
    <v-list tile class="ma-0 pa-0" lines="three" density="compact">
      <v-list-subheader class="text-title-caps-large text-neutral-lighten-3">
        UDMI Event<template v-if="version"> · v{{ version }}</template>
      </v-list-subheader>
      <v-list-item class="py-1 mb-2" v-if="eventTime">
        <v-list-item-title class="text-body-small text-capitalize">{{ eventTime.label }}</v-list-item-title>
        <v-list-item-subtitle class="text-body-1" :title="eventTime.raw">
          {{ eventTime.text }}
        </v-list-item-subtitle>
      </v-list-item>
      <v-list-item class="py-1 mb-2" v-if="message.value">
        <v-list-item-title class="text-body-small text-capitalize">Topic</v-list-item-title>
        <v-list-item-subtitle class="text-capitalize">{{ message.value?.topic }}</v-list-item-subtitle>
      </v-list-item>
      <div class="udmi-points">
        <v-list-item class="py-1" v-if="rawPayload">
          <v-list-item-subtitle class="text-body-2 udmi-json">
            <pre class="ma-0">{{ rawPayload }}</pre>
          </v-list-item-subtitle>
        </v-list-item>
        <v-list-item class="py-1" v-for="(value, key) in points" :key="key" lines="one">
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
      </div>
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

/**
 * The message payload parsed as JSON, or null if it isn't JSON at all.
 * Some devices publish payloads we can't interpret; those are shown verbatim via rawPayload
 * rather than blanking the card with a parse error thrown mid-render.
 *
 * @type {import('vue').ComputedRef<Object|null>}
 */
const parsedPayload = computed(() => {
  if (message.value === null) return null;
  try {
    const parsed = JSON.parse(message.value.payload);
    return isStructured(parsed) ? parsed : null;
  } catch {
    return null;
  }
});

/**
 * The payload text, when we couldn't parse it as a JSON object.
 *
 * @type {import('vue').ComputedRef<string>}
 */
const rawPayload = computed(() => {
  if (message.value === null || parsedPayload.value !== null) return '';
  return message.value.payload;
});

/**
 * The event time the device stamped into the payload, or '' if it didn't.
 *
 * UDMI messages - pointset events, state, and metadata alike - wrap their content in an
 * envelope carrying a top-level RFC 3339 timestamp and a schema version. A bare points map,
 * which the legacy topic and several drivers publish, has neither. A point named `timestamp`
 * doesn't get mistaken for one: point values are `{present_value: ...}` objects, not strings.
 *
 * @type {import('vue').ComputedRef<string>}
 */
const envelopeTime = computed(() => {
  const timestamp = parsedPayload.value?.timestamp;
  return typeof timestamp === 'string' ? timestamp : '';
});

/**
 * The UDMI schema version of the message, if the payload declares one.
 *
 * @type {import('vue').ComputedRef<string>}
 */
const version = computed(() => {
  if (!envelopeTime.value) return '';
  const v = parsedPayload.value.version;
  return typeof v === 'string' ? v : '';
});

/**
 * The points of the message, keyed by point name, whichever payload shape was published.
 *
 * @type {import('vue').ComputedRef<Object>}
 */
const points = computed(() => {
  const payload = parsedPayload.value;
  if (payload === null) return {};
  if (!envelopeTime.value) return payload; // a bare points map
  if (isStructured(payload.points)) return payload.points; // a pointset event
  // some other envelope, e.g. state or metadata: show all of it bar what the header already has
  const rest = {...payload};
  delete rest.timestamp;
  delete rest.version;
  return rest;
});

/**
 * The time to show in the card header.
 *
 * Prefers the event time the device stamped into the payload, which is what matters when
 * witnessing a device. Falls back to when we received the message - a browser clock reading -
 * for the bare points map payloads that carry no timestamp of their own.
 *
 * @type {import('vue').ComputedRef<{label: string, text: string, raw: string}|null>}
 */
const eventTime = computed(() => {
  if (envelopeTime.value) {
    return {label: 'Event time', text: formatTime(envelopeTime.value), raw: envelopeTime.value};
  }
  if (message.updateTime) {
    return {label: 'Last updated', text: formatTime(message.updateTime), raw: message.updateTime.toISOString()};
  }
  return null;
});

const dateTimeFormat = new Intl.DateTimeFormat('en-GB', {dateStyle: 'short', timeStyle: 'medium'});

// matches an RFC 3339 date-time, with any number of fractional second digits
const isoDateTimePattern = /^\d{4}-\d{2}-\d{2}[Tt]\d{2}:\d{2}:\d{2}(\.\d+)?([Zz]|[+-]\d{2}:\d{2})$/;

/**
 * Formats a date for display, dropping the sub-second precision devices tend to publish.
 * Values we can't read as a date are returned as given.
 *
 * @param {Date|string} value
 * @return {string}
 */
const formatTime = (value) => {
  const date = value instanceof Date ? value : new Date(value);
  if (isNaN(date.getTime())) return String(value);
  return dateTimeFormat.format(date);
};

const displayValue = (value) => {
  let v = value?.present_value ?? value;
  if (typeof v === 'string') {
    const trimmed = v.trim();
    if (isoDateTimePattern.test(trimmed)) {
      return formatTime(trimmed);
    }
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
/*
 Scroll the points within the card so the topic and timestamp above stay visible while
 witnessing. This has to be the scroll container itself: Vuetify sets overflow on .v-card
 and on a .v-list inside a navigation drawer, either of which would leave a position:
 sticky header inert relative to the drawer's own scrollport.
*/
.udmi-points {
  max-height: 60vh;
  overflow-y: auto;
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
