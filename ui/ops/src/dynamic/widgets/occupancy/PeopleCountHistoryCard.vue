<template>
  <v-card :class="props.class" :style="props.style" class="d-flex flex-column">
    <v-toolbar v-if="!hideToolbar" color="transparent">
      <v-toolbar-title class="text-h4" style="overflow-wrap: break-word">{{ title }}</v-toolbar-title>
      <v-spacer/>
      <v-chip
          v-if="props.showBaseline && summaryPct !== null"
          :color="summaryPct > 0 ? 'success-lighten-1' : 'error-lighten-1'"
          size="small"
          label
          class="flex-shrink-0">
        <v-icon start :icon="summaryPct > 0 ? 'mdi-trending-up' : 'mdi-trending-down'"/>
        {{ Math.abs(summaryPct).toFixed(1) }}% vs prior period
      </v-chip>
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
                                  :start="_start" :end="_end" :offset="_offset"
                                  :show-baseline="props.showBaseline" :baseline-shift="baselineShift"/>
    </v-card-text>
    <div v-if="props.showBaseline" class="d-flex ga-8 justify-center pb-3 text-caption opacity-70">
      <span class="d-flex align-center ga-3">
        <span class="legend-dot" :style="{background: currentColor}"/>
        Current
      </span>
      <span class="d-flex align-center ga-3">
        <span class="legend-line"/>
        Prior period
      </span>
    </div>
  </v-card>
</template>

<script setup>
import {useDateScale} from '@/components/charts/date.js';
import {triggerDownload} from '@/components/download/download.js';
import PeriodChooserRows from '@/components/PeriodChooserRows.vue';
import {useOccupancyNormalized, shiftFnFromStr} from '@/dynamic/widgets/occupancy/baseline.js';
import PeopleCountHistoryChart from '@/dynamic/widgets/occupancy/PeopleCountHistoryChart.vue';
import {useLocalProp} from '@/util/vue.js';
import {computed, toRef} from 'vue';
import * as vColors from 'vuetify/util/colors';

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
  // When true, overlays a dashed prior-period line.
  showBaseline: {
    type: Boolean,
    default: true,
  },
  // 'day', 'week', 'month' — how far back to shift the baseline
  baselineShift: {
    type: String,
    default: 'week'
  },
  downloadEnterLeave: {
    type: Boolean,
    default: false,
  }
});

const _start = useLocalProp(toRef(props, 'start'));
const _end = useLocalProp(toRef(props, 'end'));
const _offset = useLocalProp(toRef(props, 'offset'));
const {startDate, endDate, pastEdges} = useDateScale(_start, _end, _offset);

const shiftFn = computed(() => shiftFnFromStr(props.baselineShift));

// Compute baseline comparison
const {summaryPct} = useOccupancyNormalized(
  toRef(props, 'totalOccupancyName'),
  pastEdges,
  shiftFn
);

const currentColor = vColors.blue.base;

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
.legend-dot {
  display: inline-block;
  width: 10px;
  height: 10px;
  border-radius: 2px;
  background: v-bind(currentColor);
}

.legend-line {
  display: inline-block;
  width: 20px;
  height: 2px;
  background-image: linear-gradient(to right, #aaaaaa 50%, transparent 50%);
  background-size: 8px 2px;
  background-repeat: repeat-x;
}
</style>