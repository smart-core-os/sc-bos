<template>
  <v-card elevation="0" tile>
    <v-list tile class="ma-0 pa-0">
      <v-list-subheader class="text-title-caps-large text-neutral-lighten-3">Temperature</v-list-subheader>
      <v-list-item v-for="(val, key) of temperatureData" :key="key" class="py-1">
        <v-list-item-title class="text-body-small text-capitalize">{{ camelToSentence(key) }}</v-list-item-title>
        <template #append>
          <v-list-item-subtitle class="text-capitalize font-weight-medium text-body-1">{{ val }}</v-list-item-subtitle>
        </template>
      </v-list-item>
    </v-list>
  </v-card>
</template>

<script setup>
import {useTemperature} from '@/traits/temperature/temperature.js';
import {camelToSentence} from '@/util/string';

const props = defineProps({
  value: {
    type: Object, // of Temperature.AsObject
    default: () => ({})
  },
  loading: {
    type: Boolean,
    default: false
  }
});

const {
  temperatureData
} = useTemperature(() => props.value);
</script>

<style scoped>
.v-list-item {
  min-height: auto;
}
</style>

