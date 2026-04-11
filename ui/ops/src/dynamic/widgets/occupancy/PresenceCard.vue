<template>
  <content-card class="pt-4 pb-5">
    <div class="d-flex flex-row mb-2">
      <v-card-title class="text-h4 pl-4">Presence</v-card-title>
      <status-alert :resource="occupancyErr"/>
    </div>
    <v-col cols="12" class="d-flex flex-column pl-4">
      <v-row class="d-flex flex-column align-left px-3 pb-2">
        <v-chip :color="iconColor" size="small" label class="mb-1 mt-n3 align-self-start">
          <v-icon start :icon="icon"/>
          {{ stateStr }}
        </v-chip>
        <div class="d-flex flex-row align-start ma-0 text-caption font-weight-regular">
          <span class="mr-1">Last updated:</span>
          <span>{{ timeAgo }}</span>
        </div>
      </v-row>
    </v-col>
  </content-card>
</template>

<script setup>
import ContentCard from '@/components/ContentCard.vue';
import {DAY, HOUR, MINUTE, SECOND, useNow} from '@/components/now.js';
import StatusAlert from '@/components/StatusAlert.vue';
import {useOccupancy, usePullOccupancy} from '@/traits/occupancy/occupancy.js';
import {formatTimeAgo} from '@/util/date.js';
import {computed} from 'vue';

const props = defineProps({
  source: {
    type: String,
    required: true
  }
});

const {value: occupancyValue, streamError: occupancyErr} = usePullOccupancy(() => props.source);
const {stateStr, icon, iconColor, lastUpdate} = useOccupancy(occupancyValue);

// Create a lastChecked timestamp (for second to be used in the status popup
const {now} = useNow(SECOND);

// Create a timeAgo computed property to display time in words
const timeAgo = computed(() => {
  if (!lastUpdate.value) return 'Never';
  return formatTimeAgo(lastUpdate.value, now.value, MINUTE, HOUR, DAY);
});
</script>
