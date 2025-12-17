<template>
  <v-card :class="props.class" :style="props.style" class="d-flex flex-column">
    <v-toolbar v-if="!hideToolbar" color="transparent">
      <v-toolbar-title class="text-h4">{{ title }}</v-toolbar-title>
      <v-btn
          icon="mdi-dots-vertical"
          size="small"
          variant="text">
        <v-icon size="24"/>
        <v-menu activator="parent" location="bottom right" offset="8" :close-on-content-click="false">
          <v-card min-width="24em">
            <v-list density="compact">
              <v-list-subheader title="Devices"/>
              <v-list-item
                  v-for="(item, index) in legendItems"
                  :key="index"
                  @click="item.onClick(item.hidden)"
                  :title="item.text">
                <template #prepend>
                  <v-list-item-action start>
                    <v-checkbox-btn :model-value="!item.hidden" readonly :color="item.bgColor" density="compact"/>
                  </v-list-item-action>
                </template>
              </v-list-item>
              <v-list-subheader title="Data"/>
              <period-chooser-rows v-model:start="_start" v-model:end="_end" v-model:offset="_offset"/>
              <v-list-item title="Export CSV..."
                           @click="onDownloadClick"
                           v-tooltip:bottom="'Download a CSV of the chart data'"/>
            </v-list>
          </v-card>
        </v-menu>
      </v-btn>
    </v-toolbar>
    <v-card-text class="flex-grow-1 d-flex pt-0">
      <allocation-history-chart
          class="flex-grow-1 ma-n2"
          v-bind="$attrs"
          :source="props.source"
          :start="_start"
          :end="_end"
          :offset="_offset"
          ref="chartRef"/>
    </v-card-text>
  </v-card>
</template>
<script setup>

import {useDateScale} from '@/components/charts/date.js';
import {triggerDownload} from '@/components/download/download.js';
import PeriodChooserRows from '@/components/PeriodChooserRows.vue';
import AllocationHistoryChart from '@/dynamic/widgets/usage/AllocationHistoryChart.vue';
import {useLocalProp} from '@/util/vue.js';
import {computed, ref, toRef} from 'vue';

const props = defineProps({
  source: {
    type: [String, Array],
    required: true,
  },
  title: {
    type: String,
    default: 'Resource Allocation'
  },
  hideToolbar: {
    type: Boolean,
    default: false
  },
  class: {type: [String, Object, Array], default: undefined},
  style: {type: [String, Object, Array], default: undefined},
  start: {
    type: [String, Number, Date],
    default: 'day', // 'month', 'day', etc. meaning 'start of <day>' or a Date-like object
  },
  end: {
    type: [String, Number, Date],
    default: 'day' // 'month', 'day', etc. meaning 'end of <day>' or a Date-like object
  },
  offset: {
    type: [Number, String],
    default: 0, // when start/End is 'month', 'day', etc. offset that value into the past, like 'last month'
  }
});

const chartRef = ref(null);

const _source = computed(() => Array.isArray(props.source) ? props.source : [props.source]);

const _start = useLocalProp(toRef(props, 'start'));
const _end = useLocalProp(toRef(props, 'end'));
const _offset = useLocalProp(toRef(props, 'offset'));
const {startDate, endDate} = useDateScale(_start, _end, _offset);

// Get legend items from the chart component
const legendItems = computed(() => chartRef.value?.legendItems || []);
// Get visible device names for CSV export
const visibleDeviceNames = () => {
  return _source.value;
};

const onDownloadClick = async () => {
  const names = visibleDeviceNames();

  const conditions = Array.isArray(names)
      ? {conditionsList: [{field: 'name', stringIn: {stringsList: names}}]}
      : {conditionsList: [{field: 'name', stringEqual: names}]};

  await triggerDownload(
      props.title?.toLowerCase()?.replace('_', '-') ?? 'allocation',
      conditions,
      {startTime: startDate.value, endTime: endDate.value},
      {
        includeColsList: [
          {name: 'timestamp', title: 'Time'},
          {name: 'md.name', title: 'Device Name'},
          // see devices/download_data.go for list of available fields
          {name: "allocation.assignment", title: "Assignment"},
          {name: 'allocation.actor.title', title: 'User'},
          {name: 'allocation.period.startTime', title: 'Period Start'},
          {name: 'allocation.period.endTime', title: 'Period End'},
          {name: "allocation.groupId", title: "Group Name"}
        ]
      }
  );
}

</script>


<style scoped>

</style>