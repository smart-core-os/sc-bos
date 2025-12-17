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
              <v-list-item v-if="Object.keys(props.scaleValues).length > 0" title="Metric">
                <template #append>
                  <v-btn-toggle
                      mandatory v-model="metricType"
                      variant="outlined" density="compact"
                      divided class="ml-4">
                    <v-btn :value="'unscaled'"
                           size="small"
                           :text="unit"
                           width="5rem"
                           min-width="auto"
                           class="px-0"/>
                    <v-btn :value="'scaled'"
                           :text="'/' + props.scaleUnit"
                           size="small"
                           width="5rem"
                           min-width="auto"
                           class="px-0"/>
                  </v-btn-toggle>
                </template>
              </v-list-item>
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
        <bar ref="chartRef" :options="chartOptions" :data="chartData" :plugins="[vueLegendPlugin, themeColorPlugin]"/>
        <div v-if="showNoData" class="no-data-overlay">
          <no-data-graphic class="no-data-graphic"/>
        </div>
      </div>
    </v-card-text>
    <meter-tooltip :data="tooltipData" :edges="edges" :tick-unit="tickUnit" :unit="displayUnit" :hide-total-consumption="metricType === 'scaled'"/>
  </v-card>
</template>

<script setup>
import {useDateScale} from '@/components/charts/date.js';
import {useExternalTooltip, useThemeColorPlugin, useVueLegendPlugin} from '@/components/charts/plugins.js';
import {triggerDownload} from '@/components/download/download.js';
import {computeDatasets, datasetSourceName} from '@/dynamic/widgets/meter/chart.js';
import MeterTooltip from '@/dynamic/widgets/meter/MeterTooltip.vue';
import PeriodChooserRows from '@/components/PeriodChooserRows.vue';
import {useDescribeMeterReading} from '@/traits/meter/meter.js';
import {isNullOrUndef} from '@/util/types.js';
import {useLocalProp} from '@/util/vue.js';
import {BarElement, Chart as ChartJS, Legend, LinearScale, TimeScale, Title, Tooltip} from 'chart.js'
import {startOfDay, startOfYear} from 'date-fns';
import {computed, ref, toRef, watch} from 'vue';
import {Bar} from 'vue-chartjs';
import 'chartjs-adapter-date-fns';
import {useMeterConsumption, useMetersConsumption} from './consumption.js';
import NoDataGraphic from '@/dynamic/widgets/general/no-data-in-date-range.svg';

ChartJS.register(Title, Tooltip, BarElement, LinearScale, TimeScale, Legend);
const chartRef = ref(null);

const props = defineProps({
  title: {
    type: String,
    default: 'Energy Usage'
  },
  totalConsumptionName: {
    type: String,
    default: undefined,
  },
  totalProductionName: {
    type: String,
    default: undefined,
  },
  subConsumptionNames: {
    type: [Array],
    default: () => [],
  },
  subProductionNames: {
    type: [Array],
    default: () => [],
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
    default: '500px',
  },
  scaleValues: {
    type: Object,
    default: () => ({}),
  },
  scaleUnit: {
    type: String,
    default: 'm²',
  }
});

const rootClasses = computed(() => {
  return {
    [`density-${props.density}`]: true
  }
})

// we assume here that all the meters share the same unit, so asking about any will be enough.
const nameForDescribe = computed(() => {
  if (!isNullOrUndef(props.totalConsumptionName)) return props.totalConsumptionName;
  if (!isNullOrUndef(props.totalProductionName)) return props.totalProductionName;
  const toName = (item) => {
    if (typeof item === 'string') return item;
    return item.name;
  }
  if (props.subConsumptionNames.length > 0) return toName(props.subConsumptionNames[0]);
  if (props.subProductionNames.length > 0) return toName(props.subProductionNames[0]);
  return undefined;
})

const metricType = ref('unscaled');
const {response: meterInfo} = useDescribeMeterReading(nameForDescribe);
const unit = computed(() => meterInfo.value?.usageUnit);
const displayUnit = computed(() => {
  const base = meterInfo.value?.usageUnit;
  if (base && metricType.value === 'scaled') {
    return `${base}/${props.scaleUnit}`;
  }
  return base;
});

const _start = useLocalProp(toRef(props, 'start'));
const _end = useLocalProp(toRef(props, 'end'));
const _offset = useLocalProp(toRef(props, 'offset'));

const {edges, pastEdges, tickUnit, startDate, endDate} = useDateScale(_start, _end, _offset);

const totalConsumption = useMeterConsumption(toRef(props, 'totalConsumptionName'), pastEdges);
const totalProduction = useMeterConsumption(toRef(props, 'totalProductionName'), pastEdges);

const subConsumptions = useMetersConsumption(toRef(props, 'subConsumptionNames'), pastEdges);
const subProductions = useMetersConsumption(toRef(props, 'subProductionNames'), pastEdges);

const {
  external: tooltipExternal,
  data: tooltipData,
} = useExternalTooltip(edges, tickUnit, displayUnit);
const {legendItems, vueLegendPlugin} = useVueLegendPlugin();
const {themeColorPlugin} = useThemeColorPlugin();

const chartOptions = computed(() => {
  return /** @type {import('chart.js').ChartOptions} */ {
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
        stacked: true,
        title: {
          display: true,
          text: displayUnit.value
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
        stacked: true,
        grid: {
          offset: false, // bars default to true here, put ticks back inline with grid lines
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
  };
});

const chartLabels = computed(() => edges.value.slice(0, -1));
const chartData = computed(() => {
  const baseData = {
    labels: chartLabels.value,
    datasets: [
      ...computeDatasets('Consumption', totalConsumption, toRef(props, 'subConsumptionNames'), subConsumptions),
      ...computeDatasets('Production', totalProduction, toRef(props, 'subProductionNames'), subProductions, true),
    ]
  };
  if (metricType.value === 'scaled') {
    baseData.datasets.forEach(ds => {
      const sourceName = ds[datasetSourceName];
      const scale = props.scaleValues[sourceName] || 1;
      ds.data = ds.data.map(val => val / scale);
    });
  }
  return baseData;
});

const hasData = computed(() => {
  return chartData.value.datasets.some(ds => ds.data.some(val => val !== 0 && val));
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

// download CSV...
const visibleNames = () => {
  const names = [];
  const namesByTitle = {
    'Other Consumption': props.totalConsumptionName,
    'Total Consumption': props.totalConsumptionName,
    'Other Production': props.totalProductionName,
    'Total Production': props.totalProductionName,
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
          {name: 'timestamp', title: 'Reading Time'},
          {name: 'md.name', title: 'Device Name'},
          {name: 'meter.usage', title: (props.title || 'Energy Usage') + (unit.value ? ` (${unit.value})` : '')},
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
