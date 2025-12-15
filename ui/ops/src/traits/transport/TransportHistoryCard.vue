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
import {batchLargeArray, iterateLargeArray} from '@/util/array.js';
import equal from 'fast-deep-equal/es6';
import {onUnmounted, ref, watch} from 'vue';

const props = defineProps({
  history: {
    type: Array, // of type TransportRecord.AsObject
    default: () => []
  }
});


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


watch(props.history, (arr) => {
  arr.sort((a, b) => a.recordTime.seconds - b.recordTime.seconds);

  let prev = null;
  const now = new Date();
  reset();

  iterateLargeArray(batchLargeArray(arr), (item) => {
    if (equal(clean(item.transport, ignoreFields), clean(prev?.transport || {}, ignoreFields))) {
      prev = item;
      return;
    }
    prev = item;
    const date = new Date(item.recordTime.seconds * 1000);
    const diffTime = Math.abs(now - date);
    const diffDays = Math.ceil(diffTime / (1000 * 60 * 60 * 24));
    if (diffDays <= 1) {
      table.value.day += 1;
    }
    if (diffDays <= 7) {
      table.value.week += 1;
    }
    if (diffDays <= 30) {
      table.value.month += 1;
    }
  }, true);
}, {immediate: true, deep: true});
</script>

<style scoped>
</style>