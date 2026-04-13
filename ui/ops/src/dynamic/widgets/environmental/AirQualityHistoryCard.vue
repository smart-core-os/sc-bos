<template>
  <v-card :class="props.class" :style="props.style" class="d-flex flex-column">
    <v-toolbar v-if="!hideToolbar" color="transparent">
      <v-toolbar-title class="text-h4" style="overflow-wrap: break-word">{{ title }}</v-toolbar-title>
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
                    <span v-if="item.isDashed" class="baseline-legend-swatch" :style="{borderColor: item.bgColor}"/>
                    <v-checkbox-btn v-else :model-value="!item.hidden" readonly :color="item.bgColor" density="compact"/>
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
      <air-quality-history-chart
          class="flex-grow-1 ma-n2"
          v-bind="$attrs"
          :source="source"
          :start="_start"
          :end="_end"
          :offset="_offset"
          :occupancy="occupancy"
          :show-baseline="props.showBaseline"
          :baseline-shift="props.baselineShift"
          ref="chartRef"/>
    </v-card-text>
    <div v-if="props.showBaseline" class="d-flex ga-8 justify-center pb-3 text-caption opacity-70">
      <span class="d-flex align-center ga-3">
        <span class="legend-dot"/>
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
import AirQualityHistoryChart from '@/dynamic/widgets/environmental/AirQualityHistoryChart.vue';
import {useLocalProp} from '@/util/vue.js';
import {computed, ref, toRef} from 'vue';

const props = defineProps({
  source: {
    type: [String, Array],
    required: true,
  },
  title: {
    type: String,
    default: 'Air Quality'
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
  // Optional occupancy sensor name(s). When provided, an occupied/unoccupied shading band is
  // overlaid on the chart so CO₂ spikes can be correlated with occupancy transitions.
  occupancy: {
    type: [String, Array],
    default: null,
  },
  // When true, overlays a dashed prior-period line for each source.
  showBaseline: {
    type: Boolean,
    default: false,
  },
  // 'day', 'week', 'month' — how far back to shift the prior period window.
  baselineShift: {
    type: String,
    default: 'week',
  },
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
  const names = [];
  for (const [i, item] of legendItems.value.entries()) {
    if (!item.hidden) {
      const datasetNames = chartRef.value?.datasetNames;
      if (!datasetNames) continue;
      names.push(chartRef.value.datasetNames[i]);
    }
  }

  if (names.length === 0) {
    return _source.value;
  } else {
    return names;
  }
};

const onDownloadClick = async () => {
  const deviceNames = visibleDeviceNames();
  const conditions = Array.isArray(deviceNames) 
    ? {conditionsList: [{field: 'name', stringIn: {stringsList: deviceNames}}]}
    : {conditionsList: [{field: 'name', stringEqual: deviceNames}]};

  await triggerDownload(
      props.title?.toLowerCase()?.replace(' ', '-') ?? 'air-quality',
      conditions,
      {startTime: startDate.value, endTime: endDate.value},
      {
        includeColsList: [
          {name: 'timestamp', title: 'Reading Time'},
          {name: 'md.name', title: 'Device Name'},
          // see devices/download_data.go for list of available fields
          {name: 'iaq.co2', title: 'CO2 (ppm)'},
          {name: 'iaq.voc', title: 'VOC (ppm)'},
          {name: 'iaq.pressure', title: 'Pressure (hPa)'},
          {name: 'iaq.comfort', title: 'Comfort'},
          {name: 'iaq.infectionrisk', title: 'Infection Risk (%)'},
          {name: 'iaq.score', title: 'Score (%)'},
          {name: 'iaq.pm1', title: 'PM 1 micron (ug/m3)'},
          {name: 'iaq.pm25', title: 'PM 2.5 micron (ug/m3)'},
          {name: 'iaq.pm10', title: 'PM 10 micron (ug/m3)'},
          {name: 'iaq.airchange', title: 'Air Changes per Hour'},
        ]
      }
  )
}
</script>

<style scoped>
.baseline-legend-swatch {
  display: inline-block;
  width: 20px;
  height: 0;
  border-top: 2px dashed;
  margin: 0 10px;
  vertical-align: middle;
  flex-shrink: 0;
}

.legend-dot {
  display: inline-block;
  width: 10px;
  height: 10px;
  border-radius: 2px;
  background: currentColor;
  opacity: 0.8;
}

.legend-line {
  display: inline-block;
  width: 20px;
  height: 2px;
  background-image: linear-gradient(to right, currentColor 50%, transparent 50%);
  background-size: 8px 2px;
  background-repeat: repeat-x;
  opacity: 0.8;
}
</style>