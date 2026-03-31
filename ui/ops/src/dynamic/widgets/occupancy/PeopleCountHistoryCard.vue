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
      <people-count-history-chart class="flex-grow-1 ma-n2" v-bind="$attrs" :total-occupancy-name="totalOccupancyName"
                                  :start="_start" :end="_end" :offset="_offset"/>
    </v-card-text>
  </v-card>
</template>

<script setup>
import {useDateScale} from '@/components/charts/date.js';
import {triggerDownload} from '@/components/download/download.js';
import PeriodChooserRows from '@/components/PeriodChooserRows.vue';
import PeopleCountHistoryChart from '@/dynamic/widgets/occupancy/PeopleCountHistoryChart.vue';
import {useLocalProp} from '@/util/vue.js';
import {toRef} from 'vue';

const props = defineProps({
  totalOccupancyName: {
    type: String,
    required: true,
  },
  title: {
    type: String,
    default: 'People Count'
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
  },
  downloadEnterLeave: {
    type: Boolean,
    default: false,
  }
});

const _start = useLocalProp(toRef(props, 'start'));
const _end = useLocalProp(toRef(props, 'end'));
const _offset = useLocalProp(toRef(props, 'offset'));
const {startDate, endDate} = useDateScale(_start, _end, _offset);

const onDownloadClick = async () => {
  if (!props.downloadEnterLeave) {
    await triggerDownload(
        props.title?.toLowerCase()?.replace(' ', '-') ?? 'people-count',
        {conditionsList: [{field: 'name', stringEqual: props.totalOccupancyName}]},
        {startTime: startDate.value, endTime: endDate.value},
        {
          includeColsList: [
            {name: 'timestamp', title: 'Reading Time'},
            {name: 'md.name', title: 'Device Name'},
            // see devices/download_data.go for list of available fields
            {name: 'occupancy.state', title: 'State'},
            {name: 'occupancy.peoplecount', title: 'People Count'},
          ]
        }
    )
    return;
  }

  await triggerDownload(
      props.title?.toLowerCase()?.replace(' ', '-') ?? 'people-count-enter-leave',
      {conditionsList: [{field: 'name', stringEqual: props.totalOccupancyName}]},
      {startTime: startDate.value, endTime: endDate.value},
      {
        includeColsList: [
          {name: 'timestamp', title: 'Event Time'},
          {name: 'md.name', title: 'Device Name'},
          // see devices/download_data.go for list of available fields
          {name: 'enterleave.entertotal', title: 'Enter Total'},
          {name: 'enterleave.leavetotal', title: 'Leave Total'},
        ]
      }
  )
}
</script>

<style scoped>

</style>