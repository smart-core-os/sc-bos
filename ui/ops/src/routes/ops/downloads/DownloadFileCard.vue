<template>
  <v-data-table
      :items="tableItems"
      :headers="headers"
      class="elevation-1">
    <template #item.fileName="{ item }">
      {{ item.fileName }}
    </template>
    <template #item.download="{ item }">
      <v-btn
          icon="mdi-download"
          size="small"
          variant="text"
          @click="downloadFile(item.fileName)"/>
    </template>
  </v-data-table>
</template>

<script setup>
import {useUiConfigStore} from '@/stores/uiConfig.js';
import {computed, defineProps} from 'vue';
import {useDownloads} from '@/routes/ops/downloads/useDownloads.js';


const props = defineProps({
  serverEndpoint: {
    type: String,
    default: null
  }
});

const uiConfig = useUiConfigStore();

const serverEndpoint = computed(() => {
  return props.serverEndpoint ?? uiConfig.config.ops.downloadsServerEndpoint ?? '';
})

const {listAvailableFiles, downloadFile} = useDownloads(
    `${serverEndpoint.value}/known-files`,
    `${serverEndpoint.value}/download-file`
);

const headers = [
  {title: 'File Name', key: 'fileName', sortable: true},
  {title: 'Download', key: 'download', sortable: false}
];

const tableItems = computed(() =>
    listAvailableFiles.value.map(fileName => ({fileName}))
);
</script>

<style lang="scss" scoped>
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

.v-data-table {
  :deep(.v-table__wrapper) {
    // Toolbar titles have a leading margin of 20px, table cells have a leading padding of 16px.
    // Correct for this and align the leading edge of text in the first column with the toolbar title.
    padding: 0 4px;
  }
}
</style>
