<template>
  <v-card class="d-flex flex-column" :class="rootClasses">
    <v-toolbar class="chart-header" color="transparent" v-if="!props.hideToolbar">
      <v-toolbar-title class="text-h4" style="overflow-wrap: break-word">{{ props.title }}</v-toolbar-title>
      <v-btn
          icon="mdi-dots-vertical"
          size="small"
          variant="text">
        <v-icon size="24"/>
        <v-menu activator="parent" location="bottom right" offset="8" :close-on-content-click="false">
          <v-card min-width="24em">
            <v-list density="compact">
              <v-list-subheader title="Sources"/>
              <v-list-item
                  v-for="(item, index) in legendItems"
                  :key="index"
                  @click="item.onClick(item.hidden)"
                  :title="item.text">
                <template #prepend>
                  <v-list-item-action start>
                    <span v-if="item.isDashed" class="peak-legend-swatch" :style="{borderColor: item.bgColor}"/>
                    <v-checkbox-btn v-else :model-value="!item.hidden" readonly :color="item.bgColor" density="compact"/>
                  </v-list-item-action>
                </template>
              </v-list-item>
              <v-list-subheader title="Data"/>
              <period-chooser-rows v-model:start="_start" v-model:end="_end" v-model:offset="_offset"/>
              <v-list-item title="Export CSV..."
                           @click="onDownloadClick" :disabled="downloadBtnDisabled"
                           v-tooltip:bottom="'Download a CSV of the chart data'"/>
            </v-list>
          </v-card>
        </v-menu>
      </v-btn>
    </v-toolbar>
    <v-card-text class="flex-1-1-100 pt-0">
      <div class="chart__container"
           @mouseenter="onChartEnter"
           @mouseleave="onChartLeave">
        <line-chart ref="chartRef"
                    :options="chartOptions"
                    :data="chartData"
                    :plugins="[vueLegendPlugin, themeColorPlugin]"/>
        <div v-if="showNoData" class="no-data-overlay">
          <no-data-graphic class="no-data-graphic"/>
        </div>
      </div>
    </v-card-text>
    <demand-tooltip :data="tooltipData" :edges="edges" :tick-unit="tickUnit" :unit="unit" :show-total="stacked"/>
  </v-card>
</template>

<script setup>
import {useDateScale} from '@/components/charts/date.js';
import {useExternalTooltip, useThemeColorPlugin, useVueLegendPlugin} from '@/components/charts/plugins.js';
import {defineChartOptions} from '@/components/charts/util.js';
import {triggerDownload} from '@/components/download/download.js';
import {computeDatasets, datasetSourceName} from '@/dynamic/widgets/energy/chart.js';
import {Units, useDemand, useDemands, usePeakDemand, usePeakDemands, usePresentMetric} from '@/dynamic/widgets/energy/demand.js';
import * as vColors from 'vuetify/util/colors';
import DemandTooltip from '@/dynamic/widgets/energy/DemandTooltip.vue';
import PeriodChooserRows from '@/components/PeriodChooserRows.vue';
import {useLocalProp} from '@/util/vue.js';
import {Chart as ChartJS, Legend, LinearScale, LineElement, PointElement, TimeScale, Title, Tooltip} from 'chart.js'
import {startOfDay, startOfYear} from 'date-fns';
import {computed, ref, toRef, toValue, watch} from 'vue';
import {Line as LineChart} from 'vue-chartjs';
import 'chartjs-adapter-date-fns';
import NoDataGraphic from '@/dynamic/widgets/general/no-data-in-date-range.svg';

ChartJS.register(Title, Tooltip, LineElement, LinearScale, PointElement, TimeScale, Legend);
const chartRef = ref(null);

const props = defineProps({
  title: {
    type: String,
    default: 'Power Demand'
  },
  // The name of the device that represents the total electrical demand.
  totalDemandName: {
    type: String,
    default: undefined,
  },
  // A list of names for electric devices that will be rendered
  demandNames: {
    type: [Array],
    default: () => [],
  },
  // Whether demand series should be stacked.
  // "auto" means stack if there is a total demand series, otherwise don't stack.
  // "stack" means stack series on top of each other as a sum.
  // "none" means don't stack, even if there is a total demand series.
  stacking: {
    type: String,
    default: 'auto',
  },
  // The electric demand metric to use.
  // See ElectricDemand for available properties.
  // "auto" will select the first present metric from "realPower", "apparentPower", "reactivePower", "current".
  // All dataset will use the same metric.
  metric: {
    type: String,
    default: 'auto'
  },
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
  density: {
    type: String,
    default: 'default' // 'comfortable', 'compact'
  },
  hideToolbar: {
    type: Boolean,
    default: false,
  },
  minChartHeight: {
    type: [String, Number],
    default: '100%',
  },
  // When true, overlays a dashed line per sub-demand showing the peak value in each bucket.
  // Peak lines share the same colour as their corresponding average bar.
  showPeaks: {
    type: Boolean,
    default: false,
  },
});

const rootClasses = computed(() => {
  return {
    [`density-${props.density}`]: true
  }
})

// figure out which property of the demand we should be using
const metricChoices = computed(() => {
  if (props.metric !== 'auto') return [props.metric];
  return ['realPower', 'apparentPower', 'reactivePower', 'current'];
});
const metricNames = computed(() => {
  return [props.totalDemandName, ...(props.demandNames ?? [])];
})
const _metric = usePresentMetric(metricChoices, metricNames);
const unit = computed(() => Units[_metric.value]);

// x-axis processing
const _start = useLocalProp(toRef(props, 'start'));
const _end = useLocalProp(toRef(props, 'end'));
const _offset = useLocalProp(toRef(props, 'offset'));
const {edges, pastEdges, tickUnit, startDate, endDate} = useDateScale(_start, _end, _offset);

const totalDemand = useDemand(toRef(props, 'totalDemandName'), pastEdges, _metric);
const peakTotalDemand = usePeakDemand(toRef(props, 'totalDemandName'), pastEdges, _metric);
const subDemands = useDemands(toRef(props, 'demandNames'), pastEdges, _metric);
const peakDemands = usePeakDemands(toRef(props, 'demandNames'), pastEdges, _metric);

const {
  external: tooltipExternal,
  data: tooltipData,
} = useExternalTooltip(edges, tickUnit, unit);
const {legendItems, vueLegendPlugin} = useVueLegendPlugin();
const {themeColorPlugin} = useThemeColorPlugin();

const stacked = computed(() => {
  if (props.stacking === 'stack') return true;
  if (props.stacking === 'none') return false;
  // 'auto'
  return !!props.totalDemandName;
})

const chartOptions = computed(() => {
  return /** @type {import('chart.js').ChartOptions} */ defineChartOptions({
    responsive: true,
    maintainAspectRatio: false,
    borderRadius: 3,
    borderWidth: 1,
    spanGaps: true,
    interaction: {
      mode: 'index', // a single tooltip with all stacked datasets at the same x location in it
    },
    plugins: {
      tooltip: {
        enabled: false,
        external: tooltipExternal
      },
      legend: {
        display: false, // we use a custom legend plugin and vue for this
      }
    },
    scales: {
      y: {
        stacked: stacked.value,
        title: {
          display: true,
          text: unit.value
        },
        border: {
          color: 'transparent'
        },
        grid: {
          color(ctx) {
            if (ctx.tick.value === 0) return '#fff4';
            return '#fff1';
          },
          drawTicks: false,
        },
        ticks: {
          callback(value) {
            return new Intl.NumberFormat(undefined, {}).format(Math.abs(value));
          },
          color: '#fff',
          padding: 8
        },
      },
      x: {
        type: 'time',
        stacked: false,
        grid: {
          color: '#fff1'
        },
        ticks: {
          maxTicksLimit: 11,
          includeBounds: true,
          callback(value) {
            const unit = tickUnit.value;
            if (unit === 'month' && value === startOfYear(value).getTime()) return this.format(value, this.options.time.displayFormats['year']);
            if (unit === 'hour' && value === startOfDay(value).getTime()) return this.format(value, this.options.time.displayFormats['day']);
            return this.format(value);
          },
          color: '#fff',
          padding: 8,
          maxRotation: 0
        },
        time: {
          unit: tickUnit.value,
          displayFormats: {
            hour: 'H:mm', // default: 4:00 AM
            day: 'd MMM', // default: Feb 10
            month: 'MMM', // default: Feb 2025 - we fix the ambiguity in ticks.callback
          }
        }
      }
    }
  });
});

// The same colour palette the themeColorPlugin applies in dataset order
const paletteColors = [
  vColors.blue.base,
  vColors.green.base,
  vColors.orange.base,
  vColors.yellow.base,
  vColors.red.base,
];

const chartLabels = computed(() => edges.value.slice(0, -1));
const chartData = computed(() => {
  const avgDatasets = computeDatasets('Demand', totalDemand, toRef(props, 'demandNames'), subDemands);

  if (!props.showPeaks) {
    return {labels: chartLabels.value, datasets: avgDatasets};
  }

  // Build one dashed line per sub-demand. They are added AFTER the avg datasets so the
  // themeColorPlugin (which processes datasets in order) has already assigned colours to the
  // avg bars. We manually pre-set the peak colours so the plugin skips them.
  const _names = toRef(props, 'demandNames').value ?? [];
  const peakDatasets = _names.flatMap((item, i) => {
    const name = typeof item === 'object' ? item.name : item;
    const series = peakDemands[name];
    if (!series) return [];
    const color = paletteColors[i % paletteColors.length];
    return [{
      type: 'line',
      _isPeak: true,
      [datasetSourceName]: name,
      label: `${toValue(series.title)} peak`,
      data: toValue(series.data).map(pt => pt.y),
      borderColor: color,
      backgroundColor: 'transparent',
      borderDash: [6, 3],
      borderWidth: 1, // revealed on hover via onChartEnter
      pointRadius: 0,
      pointHoverRadius: 0,
      spanGaps: true,
      fill: false,
      tension: 0,
      // Unique stack key keeps peak lines out of the avg stacking group
      stack: `peak-${name}`,
    }];
  });

  // Add a peak line for the total demand if present
  const totalPeakDatasets = [];
  if (peakTotalDemand.value?.length) {
    // Total avg bar uses the first palette colour (index 0 in avgDatasets)
    // but only if it's the only dataset.
    const totalColor = _names.length === 0 ? paletteColors[0] : '#ffffff';
    totalPeakDatasets.push({
      type: 'line',
      _isPeak: true,
      label: 'Total Demand peak',
      data: peakTotalDemand.value.map(pt => pt.y),
      borderColor: totalColor,
      backgroundColor: 'transparent',
      borderDash: [6, 3],
      borderWidth: 1, // revealed on hover via onChartEnter
      pointRadius: 0,
      pointHoverRadius: 0,
      spanGaps: true,
      fill: false,
      tension: 0,
      stack: 'peak-total',
    });
  }

  return {labels: chartLabels.value, datasets: [...avgDatasets, ...peakDatasets, ...totalPeakDatasets]};
});

const hasData = computed(() => {
  return chartData.value.datasets.some(ds => ds.data.some(val => val != null));
});

// Track if initial data load is complete to avoid showing no-data graphic during fetch
const hasLoadedData = ref(false);

// Reset loading state when date range changes
watch([startDate, endDate], () => {
  hasLoadedData.value = false;
});

watch(chartData, (data) => {
  // Check if we have any non-null data points (even if they're zero)
  const hasAnyData = data.datasets.some(ds => ds.data.some(val => val != null));
  if (hasAnyData && !hasLoadedData.value) {
    hasLoadedData.value = true;
  }
}, {immediate: true});

const showNoData = computed(() => !hasData.value && hasLoadedData.value);

// Show/hide peak datasets on chart hover
const onChartEnter = () => {
  const chart = chartRef.value?.chart;
  if (!chart) return;
  let changed = false;
  for (const ds of chart.data.datasets) {
    if (ds._isPeak) {
      ds.borderWidth = 1.5;
      ds.pointRadius = 3;
      ds.pointHoverRadius = 5;
      changed = true;
    }
  }
  if (changed) chart.update('none');
};
const onChartLeave = () => {
  const chart = chartRef.value?.chart;
  if (!chart) return;
  let changed = false;
  for (const ds of chart.data.datasets) {
    if (ds._isPeak) {
      ds.borderWidth = 1;
      ds.pointRadius = 0;
      ds.pointHoverRadius = 0;
      changed = true;
    }
  }
  if (changed) chart.update('none');
};

// download CSV...
const visibleNames = () => {
  const names = [];
  const namesByTitle = {
    'Other Demand': props.totalDemandName,
    'Total Demand': props.totalDemandName,
  };
  const chart = /** @type {import('chart.js').Chart} */ chartRef.value?.chart;
  if (!chart) return [];

  for (const legendItem of chart.legend.legendItems) {
    if (legendItem.hidden) continue;
    const dataset = chart.data.datasets[legendItem.datasetIndex];
    if (!dataset) continue;
    const name = dataset[datasetSourceName];
    if (name) {
      names.push(name);
    } else {
      const title = legendItem.text;
      const name = namesByTitle[title];
      if (name) names.push(name);
    }
  }

  return names;
};
const downloadBtnDisabled = computed(() => {
  return legendItems.value.every((item) => item.hidden);
})
const onDownloadClick = async () => {
  const names = visibleNames();
  if (names.length === 0) return;
  await triggerDownload(
      props.title?.toLowerCase()?.replace(' ', '-') ?? 'energy-usage',
      {conditionsList: [{field: 'name', stringIn: {stringsList: names}}]},
      {startTime: startDate.value, endTime: endDate.value},
      {
        includeColsList: [
          {name: 'timestamp', title: 'Time'},
          {name: 'name', title: 'Device Name'},
          {name: 'electric.realpower', title: 'Real Power (W)'},
          {name: 'electric.apparentpower', title: 'Apparent Power (VA)'},
          {name: 'electric.reactivepower', title: 'Reactive Power (VAR)'},
          {name: 'electric.powerfactor', title: 'Power Factor'},
          {name: 'electric.current', title: 'Current (A)'}
        ]
      }
  )
}
</script>

<style scoped lang="scss">
.density-comfortable,
.density-default {
  padding: 16px 24px;

  .v-toolbar {
    margin-bottom: 14px;
  }
}

.chart-header {
  align-items: center;

  :deep(.v-toolbar__content) {
    justify-content: end;
    flex-wrap: wrap;
  }

  :deep(.v-toolbar-title) {
    flex: 1 1 auto;
    overflow: visible;
    white-space: normal;
  }
  :deep(.v-toolbar-title__placeholder) {
    overflow: visible;
    white-space: normal;
  }
}

.chart__container {
  min-height: v-bind(minChartHeight);
  /* The chart seems to have a padding no mater what we do, this gets rid of it */
  margin: -6px;
  position: relative;
}

.peak-legend-swatch {
  display: inline-block;
  width: 20px;
  height: 0;
  border-top: 2px dashed;
  margin: 0 10px;
  vertical-align: middle;
  flex-shrink: 0;
}
</style>