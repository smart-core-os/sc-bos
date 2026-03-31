<template>
  <v-card elevation="0" tile>
    <v-list tile class="ma-0 pa-0">
      <div v-if="props.history.length === 0">
        <v-list-subheader class="text-title-caps-large text-neutral-lighten-3">No Transport History</v-list-subheader>
      </div>
      <div v-else>
        <v-list-subheader class="text-title-caps-large text-neutral-lighten-3">Transport Usage</v-list-subheader>
        <v-list-subheader class="text-title-sentence-large text-neutral-lighten-3">
          Transport Calls Over
        </v-list-subheader>
        <v-list-item v-for="label in Object.keys(table)" :key="label" class="py-1">
          <v-list-item-title class="text-body-small text-capitalize">
            {{ `Last ${label}` }}
          </v-list-item-title>

          <template #append>
            <v-list-item-subtitle class="text-body-1">
              {{ table[label] }}
            </v-list-item-subtitle>
          </template>
        </v-list-item>
      </div>
    </v-list>
  </v-card>
</template>

<script setup>
import equal from 'fast-deep-equal/es6';
import {chunk} from 'lodash';
import {nextTick, onUnmounted, ref, watch} from 'vue';

const props = defineProps({
  history: {
    type: Array, // of type TransportRecord.AsObject
    default: () => []
  }
});

const DAY_MILLISECONDS = 24 * 60 * 60 * 1000;
const table = ref({day: 0, week: 0, month: 0});

const reset = () => {
  table.value.day = 0;
  table.value.week = 0;
  table.value.month = 0;
};

onUnmounted(() => {
  reset();
});


const clean = (obj, ignoreFields) => {
  return Object.fromEntries(Object.entries(obj).filter(([k]) => !ignoreFields.includes(k)));
};

const ignoreFields = ['passengerAlarm', 'doorsList', 'load'];

let isProcessing = false;
let shouldCancel = false;

watch(props.history, async (arr) => {
  if (isProcessing) {
    shouldCancel = true;
    while (isProcessing) {
      await nextTick();
    }
  }

  isProcessing = true;
  shouldCancel = false;
  // copy first to not mutate the history ref and sort by recordTime to compare previous entry
  const sortedArr = [...arr].sort((a, b) => b.recordTime.seconds - a.recordTime.seconds);
  const batched = chunk(sortedArr, 20);

  let prev = null;
  const now = new Date();
  reset();
  for (let batch of batched) {
    if (shouldCancel) {
      isProcessing = false;
      break;
    }

    for (let item of batch) {
      if (shouldCancel) {
        isProcessing = false;
        break;
      }
      if (equal(clean(item.transport, ignoreFields), clean(prev?.transport || {}, ignoreFields))) {
        prev = item;
        continue;
      }
      prev = item;
      const date = new Date(item.recordTime.seconds * 1000);
      const diffTime = Math.abs(now - date);
      const diffDays = Math.ceil(diffTime / DAY_MILLISECONDS);
      if (diffDays <= 1) {
        table.value.day += 1;
      }
      if (diffDays <= 7) {
        table.value.week += 1;
      }
      if (diffDays <= 30) {
        table.value.month += 1;
      }
      if (diffDays > 30) {
        break; // no need to continue further
      }
    }

    await nextTick();
  }
  isProcessing = false;
}, {immediate: true, deep: true});
</script>

<style scoped>
.v-list-item {
  min-height: auto;
}

</style>