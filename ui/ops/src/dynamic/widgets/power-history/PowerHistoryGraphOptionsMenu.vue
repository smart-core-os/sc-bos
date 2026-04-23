<template>
  <v-tooltip location="bottom">
    <template #activator="{ props }">
      <div v-bind="props">
        <v-menu
            location="bottom right"
            :close-on-content-click="false"
            min-width="275px">
          <template #activator="{ props: _props }">
            <v-btn
                icon="mdi-dots-vertical"
                size="small"
                variant="text"
                v-bind="_props">
              <v-icon size="24"/>
            </v-btn>
          </template>

          <v-list>
            <v-list-item @click.stop="showConversionToggle = !showConversionToggle">
              <v-list-item-title>Unit Type</v-list-item-title>
              <template #append>
                <v-list-item-action end>
                  <v-switch
                      class="ml-4 my-auto no-pointer-events"
                      density="compact"
                      readonly
                      hide-details
                      :model-value="showConversionToggle">
                    <template #prepend>
                      <span class="text-grey-lighten-1">kW</span>
                    </template>
                    <template #append>
                      <span class="text-grey-lighten-1">CO₂</span>
                    </template>
                  </v-switch>
                </v-list-item-action>
              </template>
            </v-list-item>

            <v-list-item>
              <v-list-item-title>Duration</v-list-item-title>
              <template #append>
                <v-list-item-action end>
                  <v-btn-toggle
                      v-model="activeDuration"
                      color="primary"
                      density="compact"
                      variant="outlined"
                      divided
                      mandatory>
                    <v-btn
                        v-for="option in durationOptions"
                        size="small"
                        :key="option.id"
                        :value="option.value">
                      <span class="text-caption">{{ option.text }}</span>
                    </v-btn>
                  </v-btn-toggle>
                </v-list-item-action>
              </template>
            </v-list-item>

            <v-list-item @click="emits('exportCsv')">
              <v-list-item-title class="">Export CSV...</v-list-item-title>
            </v-list-item>
          </v-list>
        </v-menu>
      </div>
    </template>
    <span>Options</span>
  </v-tooltip>
</template>

<script>
import {DAY, HOUR, MINUTE} from '@/util/date.js';

// Defining the options for the duration type buttons
export const durationOptions = [
  {
    text: '24H',
    value: {
      id: '24H',
      span: 20 * MINUTE,
      timeFrame: 24 * HOUR
    }
  },
  {
    text: '1W',
    value: {
      id: '1W',
      span: 2 * HOUR,
      timeFrame: 7 * DAY
    }
  },
  {
    text: '30D',
    value: {
      id: '30D',
      span: 6 * HOUR,
      timeFrame: 30 * DAY
    }
  }
];
</script>
<script setup>
const emits = defineEmits(['exportCsv']);

// Computed property to toggle between kW and CO2
// Syncs with the parent component
const showConversionToggle = defineModel('showConversion', {
  type: Boolean,
  default: false
});

// Computed property to toggle between the duration options
// Syncs with the parent component
const activeDuration = defineModel('durationOption', {
  type: Object,
  default: () => durationOptions[0].value
});
</script>

<style lang="scss" scoped>
.no-pointer-events {
  pointer-events: none;
}
</style>
