<template>
  <span><template v-for="(item, i) in items" :key="item.text">
    <span :class="item.class" :style="item.style">
      {{ item.text }}{{ i < items.length - 1 ? ', ' : '' }}
      <v-tooltip v-if="item.tooltip" location="bottom" activator="parent">{{ item.tooltip }}</v-tooltip>
    </span>
  </template></span>
</template>

<script setup>
import {isNullOrUndef} from '@/util/types.js';
import {computed} from 'vue';

const props = defineProps({
  value: {
    type: Object, // of Metadata.AsObject
    default: null
  }
});

/**
 * Returns the first non-nil value from the provided items.
 * If an item is a function, it will be called to get the value.
 *
 * @param {Array<function():any|any>} items
 * @return {*|null}
 */
const firstNonNil = (...items) => {
  for (const item of items) {
    let v = item;
    if (typeof v === 'function') v = v();
    if (isNullOrUndef(v)) continue;
    return v;
  }
  return null
}
/**
 * Converts a device name to a more human-readable text by removing leading numeric segments.
 * For example, "building/1/floor/2/room/3" becomes "room/3".
 *
 * @param {string} name - The device name to convert.
 * @return {string} - The converted text.
 */
const nameToText = (name) => {
  // split by / looking for the last sequence of parts that aren't just numbers
  const parts = name.split('/');
  for (let i = parts.length - 1; i >= 0; i--) {
    if (isNaN(parseInt(parts[i]))) {
      return parts.slice(i).join('/');
    }
  }
  return name;
}

const items = computed(() => {
  if (!props.value) return [{text: '-'}]
  /** @type {import('@smart-core-os/sc-api-grpc-web/traits/metadata_pb').Metadata.AsObject} */
  const md = props.value
  const items = [];

  // figure out the name
  let last = null;
  const push = (item) => {
    if (item.text === last || !item.text) return;
    items.push(item);
    last = item.text;
  }
  push({
    text: firstNonNil(
        md.appearance?.title,
        () => nameToText(md.name),
    ),
    tooltip: md.name,
  });
  push({
    text: firstNonNil(
        md.location?.zone,
        md.membership?.subsystem
    ),
    class: 'text-grey',
  });
  push({
    text: firstNonNil(
        md.location?.floor
    ),
    class: 'text-grey',
  });

  if (items.length === 0) {
    items.push({text: '-'})
  }
  return items
})
</script>

<style scoped>

</style>