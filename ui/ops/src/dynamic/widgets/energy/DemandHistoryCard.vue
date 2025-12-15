<template>
  <v-card class="d-flex flex-column" :class="rootClasses">
    <v-toolbar class="chart-header" color="transparent" v-if="!props.hideToolbar">
      <v-toolbar-title class="text-h4">{{ props.title }}</v-toolbar-title>
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
                    <v-checkbox-btn :model-value="!item.hidden" readonly :color="item.bgColor" density="compact"/>
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
      <div class="chart__container">
        <line-chart ref="chartRef"
                    :options="chartOptions"
                    :data="chartData"
                    :plugins="[vueLegendPlugin, themeColorPlugin]"/>
        <div v-if="!hasData" class="no-data-overlay">
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
import {Units, useDemand, useDemands, usePresentMetric} from '@/dynamic/widgets/energy/demand.js';
import DemandTooltip from '@/dynamic/widgets/energy/DemandTooltip.vue';
import PeriodChooserRows from '@/components/PeriodChooserRows.vue';
import {useLocalProp} from '@/util/vue.js';
import {Chart as ChartJS, Legend, LinearScale, LineElement, PointElement, TimeScale, Title, Tooltip} from 'chart.js'
import {startOfDay, startOfYear} from 'date-fns';
import {computed, ref, toRef} from 'vue';
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
  }
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
const subDemands = useDemands(toRef(props, 'demandNames'), pastEdges, _metric);

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

const chartLabels = computed(() => edges.value.slice(0, -1));
const chartData = computed(() => {
  return {
    labels: chartLabels.value,
    datasets: [
      ...computeDatasets('Demand', totalDemand, toRef(props, 'demandNames'), subDemands),
    ]
  };
});

const hasData = computed(() => {
  return chartData.value.datasets.some(ds => ds.data.some(val => val !== 0 && val != null));
});

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

  :deep(.v-toolbar-title__placeholder) {
    overflow: visible;
  }
}

.chart__container {
  min-height: v-bind(minChartHeight);
  /* The chart seems to have a padding no mater what we do, this gets rid of it */
  margin: -6px;
  position: relative;
}
</style>