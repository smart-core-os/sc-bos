<template>
  <v-card elevation="0" tile>
    <v-list tile class="ma-0 pa-0">
      <v-list-subheader class="text-title-caps-large text-neutral-lighten-3">Allocation</v-list-subheader>
      <v-list-item v-for="item of table" :key="item.label" class="py-1">
        <v-list-item-title class="text-body-small text-capitalize">
          {{ item.label }}
        </v-list-item-title>

        <template #append>
          <v-list-item-subtitle class="text-body-1">
            {{ item.value }} {{ item.unit }}
          </v-list-item-subtitle>
        </template>
      </v-list-item>
    </v-list>

    <v-progress-linear color="primary" indeterminate :active="loading"/>
  </v-card>
</template>

<script setup>

import {useAllocation} from '@/traits/allocation/allocation.js';

const props = defineProps({
  value: {
    type: Object, // of type allocation.AsObject
    default: () => {
    }
  },
  loading: {
    type: Boolean,
    default: false
  }
});

const {table} = useAllocation(() => props.value);
</script>

<style scoped>
.v-list-item {
  min-height: auto;
}

.v-progress-linear {
  width: auto;
}
</style>
