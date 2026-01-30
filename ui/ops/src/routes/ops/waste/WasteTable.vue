<template>
  <div class="ml-2">
    <v-row class="mt-0 ml-0 pl-0">
      <h3 class="text-h3 pt-2 pb-6">Waste Records</h3>
      <v-spacer/>
    </v-row>

    <content-card class="px-8 mt-8">
      <v-data-table-server
          :headers="allHeaders"
          v-bind="tableAttrs"
          disable-sort
          :items-length="queryTotalCount"
          class="pt-4">
        <template #item.createTime="{ item }">
          {{ timestampToDate(item.wasteCreateTime).toLocaleString() }}
        </template>
        <template #item.weight="{ item }">
          {{ item.weight.toFixed(2) }} {{ unit }}
        </template>
        <template #item.co2Saved="{ item }">
          <span v-if="item.co2Saved">
            {{ item.co2Saved.toFixed(2) }} {{ co2SavedUnit }}
          </span>
          <span v-else>-</span>
        </template>
        <template #item.landSaved="{ item }">
          <span v-if="item.landSaved">
            {{ item.landSaved.toFixed(2) }} {{ landSavedUnit }}
          </span>
          <span v-else>-</span>
        </template>
        <template #item.treesSaved="{ item }">
          <span v-if="item.treesSaved">
            {{ item.treesSaved.toFixed(2) }}
          </span>
          <span v-else>-</span>
        </template>
        <template #item.disposalMethod="{ item }">
          {{ getDisposalMethod(item.disposalMethod) }}
        </template>
      </v-data-table-server>
    </content-card>
  </div>
</template>

<script setup>
import {timestampToDate} from '@/api/convpb.js';
import ContentCard from '@/components/ContentCard.vue';
import {useDataTableCollection} from '@/composables/table.js';
import {useWasteRecordsCollection} from '@/composables/waste.js';
import {useCohortStore} from '@/stores/cohort.js';
import {useUiConfigStore} from '@/stores/uiConfig.js';
import {computed, ref} from 'vue';

const props = defineProps({
  source: {
    type: String,
    default: ''
  },
  unit: {
    type: String,
    default: 'kg'
  },
  co2SavedUnit: {
    type: String,
    default: 'kg'
  },
  landSavedUnit: {
    type: String,
    default: 'km²'
  }
})

const uiConfig = useUiConfigStore();
const cohort = useCohortStore();
const name = computed(() => props.source || (uiConfig.config?.ops?.waste?.source ?? cohort.hubNode?.name ?? ''));

const unit = computed(() => props.unit || uiConfig.config?.ops?.waste?.unit || 'kg');
const co2SavedUnit = computed(() => props.co2SavedUnit || uiConfig.config?.ops?.waste?.co2SavedUnit || 'kg');
const landSavedUnit = computed(() => props.landSavedUnit || uiConfig.config?.ops?.waste?.landSavedUnit || 'km²');

const wasteRecordsRequest = computed(() => ({
  name: name.value
}));
const wantCount = ref(20);
const wasteRecordsOptions = computed(() => ({
  wantCount: wantCount.value
}));
const wasteRecordsCollection = useWasteRecordsCollection(wasteRecordsRequest, wasteRecordsOptions);
const tableAttrs = useDataTableCollection(wantCount, wasteRecordsCollection);

// Calculate the total number of items in the query
const queryTotalCount = computed(() => {
  return wasteRecordsCollection.totalItems.value;
});

const allHeaders = [
  {title: 'Waste Created', value: 'createTime', width: '15em'},
  {title: 'Area', value: 'area', width: '40%'},
  {title: 'Disposal Method', value: 'disposalMethod', width: '30%'},
  {title: 'Weight', value: 'weight', width: '10em', align: 'end'},
  {title: 'Stream', value: 'stream', width: '30%'},
  {title: 'System', value: 'system', width: '30%'},
  {title: 'Co2 Saved', value: 'co2Saved', width: '10em', align: 'end'},
  {title: 'Land Saved', value: 'landSaved', width: '10em', align: 'end'},
  {title: 'Trees Saved', value: 'treesSaved', width: '10em', align: 'end'},
];

const getDisposalMethod = (disposalMethod) => {
  switch (disposalMethod) {
    case 1:
      return 'General Waste';
    case 2:
      return 'Recycling';
    default:
      return 'Unknown';
  }
}

</script>

<style scoped>
:deep(table) {
  table-layout: fixed;
}

.hide-pagination {
  :deep(.v-data-table-footer__info),
  :deep(.v-pagination__last) {
    display: none;
  }

  :deep(.v-pagination__first) {
    margin-left: 16px;
  }
}
</style>