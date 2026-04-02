<template>
  <v-toolbar v-if="showExternalHeader" color="transparent" class="mb-6" density="compact">
    <h3 class="text-h3">{{ props.title }}</h3>
  </v-toolbar>

  <component :is="showExternalHeader ? VCard : 'div'" v-bind="tableWrapperProps">
    <v-data-table-server
        :headers="allHeaders"
        :hide-default-header="props.hideTableHeader"
        :hide-default-footer="_hidePaging"
        v-bind="tableAttrs"
        disable-sort
        :items-length="queryTotalCount"
        :fixed-header="props.fixedHeader"
        :style="tableStyles">
      <template #top v-if="showCardHeader">
        <v-toolbar color="transparent">
          <v-toolbar-title v-if="showCardHeader" class="text-h4">{{ props.title }}</v-toolbar-title>
        </v-toolbar>
      </template>
      <template #item.createTime="{ item }">
        {{ timestampToDate(item.securityEventTime).toLocaleString() }}
      </template>
    </v-data-table-server>
  </component>
</template>
<script setup>
import {timestampToDate} from '@/api/convpb';
import {useSecurityEventsCollection} from '@/composables/securityevents.js';
import {useDataTableCollection} from '@/composables/table.js';
import {useCohortStore} from '@/stores/cohort.js';
import {useUiConfigStore} from '@/stores/uiConfig.js';
import {useLocalProp} from '@/util/vue.js';
import {computed} from 'vue';
import {VCard} from 'vuetify/components';

const props = defineProps({
  name: {
    type: String,
    default: null
  },
  title: {
    type: String,
    default: 'Security Events'
  },
  variant: {
    type: String,
    default: 'default'
  },
  hideTableHeader: {
    type: Boolean,
    default: false
  },
  hidePaging: {
    type: Boolean,
    default: false
  },
  // when present, paging is disabled and only this many rows are shown
  fixedRowCount: {
    type: Number,
    default: null
  },
  fixedHeader: {
    type: Boolean,
    default: true
  },
  height: {
    type: [String, Number],
    default: undefined
  }
});

const uiConfig = useUiConfigStore();
const cohort = useCohortStore();
const name = computed(() => props.name ?? uiConfig.config.securityEventsSource ?? cohort.hubNode?.name ?? '');

const _variant = computed(() => {
  if (props.variant === 'default') return 'page';
  return props.variant;
})
const showExternalHeader = computed(() => _variant.value === 'page');
const showCardHeader = computed(() => _variant.value === 'card');
const _hidePaging = computed(() => Boolean(props.hidePaging || props.fixedRowCount));

const tableWrapperProps = computed(() => {
  if (showExternalHeader.value) {
    return {
      class: ['px-7', 'py-4']
    }
  } else {
    return {};
  }
});

const securityEventsRequest = computed(() => ({
  name: name.value
}));
const wantCount = useLocalProp(computed(() => props.fixedRowCount || 20));
const securityEventsOptions = computed(() => ({
  wantCount: wantCount.value
}));
const securityEventsCollection = useSecurityEventsCollection(securityEventsRequest, securityEventsOptions);
const tableOptions = computed(() => {
  return {
    itemsPerPage: props.fixedRowCount || undefined,
  }
})
const tableAttrs = useDataTableCollection(wantCount, securityEventsCollection, tableOptions);

// Calculate the total number of items in the query
const queryTotalCount = computed(() => {
  return securityEventsCollection.totalItems.value;
});

const allHeaders = [
  {title: 'Timestamp', value: 'createTime', width: '13em'},
  {title: 'Description', value: 'description', width: '60%'},
  {title: 'Priority', value: 'priority', width: '10em', align: 'end'},
  {title: 'Source', value: 'source.name', width: '30%'}
];

const tableStyles = computed(() => {
  if (!props.fixedHeader) {
    return {};
  }
  if (props.height) {
    return {height: typeof props.height === 'number' ? `${props.height}px` : props.height};
  }
  return {height: '80vh'};
});


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
