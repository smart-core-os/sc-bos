<template>
  <div class="chart__container">
    <line-chart :data="chartData" :options="chartOptions" :plugins="[themeColorPlugin, vueLegendPlugin]"/>
    <chart-tooltip :data="tooltipData" :edges="edges" :tick-unit="tickUnit"
                   :format-value="(y) => (y != null ? new Intl.NumberFormat(undefined, {}).format(y) + (unit ? ' ' + unit : '') : '—')"/>
  </div>
</template>

<script setup>
import {useDateScale} from '@/components/charts/date.js';
import {useExternalTooltip, useThemeColorPlugin, useVueLegendPlugin} from '@/components/charts/plugins.js';
import ChartTooltip from '@/components/charts/ChartTooltip.vue';
import {defineChartOptions} from '@/components/charts/util.js';
import {useSoundLevelHistoryMetrics} from '@/dynamic/widgets/environmental/soundSensor.js';
import {shiftFnFromStr} from '@/dynamic/widgets/occupancy/baseline.js';
import {useLocalProp} from '@/util/vue.js';
import {sentenceCase} from 'change-case';
import {Chart as ChartJS, Legend, LinearScale, LineElement, PointElement, TimeScale, Title, Tooltip} from 'chart.js';
import {startOfDay, startOfYear} from 'date-fns';
import {computed, toRef, toValue} from 'vue';
import {Line as LineChart} from 'vue-chartjs';
import 'chartjs-adapter-date-fns';

const datasetSourceName = Symbol('datasetSourceName');

ChartJS.register(Title, Tooltip, LineElement, LinearScale, PointElement, TimeScale, Legend);

const props = defineProps({
  source: {
    type: [String, Array],
    default: null
  },
  metric: {
    type: String,
    default: 'soundPressureLevel'
  },
  unit: {
    type: String,
    default: null, // default calculated based on metric
  },
  start: {
    type: [String, Number, Date],
    default: 'day', // 'month', 'day', etc. meaning 'start of <day>' or a Date-like object
  },
  end: {
    type: [String, Number, Date],
    default: 'day', // 'month', 'day', etc. meaning 'end of <day>' or a Date-like object
  },
  offset: {
    type: [String, Number],
    default: 0, // when start/End is 'month', 'day', etc. offset that value into the past, like 'last month'
  },
  showBaseline: {
    type: Boolean,
    default: false,
  },
  baselineShift: {
    type: String,
    default: 'week',
  },
});

const _start = useLocalProp(toRef(props, 'start'));
const _end = useLocalProp(toRef(props, 'end'));
const _offset = useLocalProp(toRef(props, 'offset'));

const {edges, pastEdges, tickUnit, startDate, endDate} = useDateScale(_start, _end, _offset);

const shiftFn = computed(() => shiftFnFromStr(props.baselineShift));
const baselineEdges = computed(() => {
  if (!props.showBaseline) return [];
  return toValue(edges).map(shiftFn.value);
});

// Support both single source (string) and multiple sources (array)
const sources = computed(() => {
  if (Array.isArray(props.source)) {
    return props.source;
  } else if (props.source) {
    return [props.source];
  }
  return [];
});

const datasetNames = computed(() => {
  return chartData.value.datasets.map(item => {
    return item[datasetSourceName];
  });
});

// Always use the devices composable for consistency
const devices = useSoundLevelHistoryMetrics(sources, toRef(props, 'metric'), pastEdges);
const baselineDevices = useSoundLevelHistoryMetrics(
    computed(() => props.showBaseline ? sources.value : []),
    toRef(props, 'metric'),
    baselineEdges
);

const yAxisLabel = computed(() => {
  const s = sentenceCase(props.metric);
  if (props.unit) {
    return `${s} (${props.unit})`;
  }
  // Default unit based on metric
  if (props.metric === 'pressureLevel') {
    return `${s} (dB)`;
  }
  return s;
});

// Always use both plugins
const {legendItems, vueLegendPlugin} = useVueLegendPlugin();
const {themeColorPlugin} = useThemeColorPlugin();
const {external: tooltipExternal, data: tooltipData} = useExternalTooltip();

const unit = computed(() => {
  if (props.unit) return props.unit;
  if (props.metric === 'pressureLevel') return 'dB';
  return '';
});

const chartOptions = computed(() => {
  return defineChartOptions({
    responsive: true,
    maintainAspectRatio: false,
    borderRadius: 3,
    interaction: {
      mode: 'index', // a single tooltip with all stacked datasets at the same x location in it
      intersect: false,
    },
    plugins: {
      legend: {
        display: false, // we use a custom legend plugin and vue for this
      },
      tooltip: {
        enabled: false,
        external: tooltipExternal,
      }
    },
    scales: {
      y: {
        beginAtZero: true,
        title: {
          display: true,
          text: yAxisLabel.value,
          color: 'rgba(255, 255, 255, 0.7)',
        },
        border: {
          display: false
        },
        grid: {
          color(ctx) {
            if (ctx.tick.value === 0) return 'rgba(255, 255, 255, 0.25)';
            return 'rgba(255, 255, 255, 0.08)';
          },
          drawTicks: false,
        },
        ticks: {
          callback(value) {
            return new Intl.NumberFormat(undefined, {}).format(Math.abs(value));
          },
          color: 'rgba(255, 255, 255, 0.9)',
          padding: 8
        },
      },
      x: {
        type: 'time',
        min: startDate.value,
        max: endDate.value,
        stacked: false,
        border: {
          display: false
        },
        grid: {
          display: false,
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
          color: 'rgba(255, 255, 255, 0.9)',
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
  let datasets = [];
  const sourceList = sources.value;

  // Add current period datasets
  for (const name of sourceList) {
    const device = devices[name];
    if (!device) continue;
    const label = toValue(device.title) || name;
    const deviceData = toValue(device.data);
    const d = deviceData.map(p => ({x: p.x, y: p.y}));
    const dates = deviceData.map(p => p.x);
    if (d.length > 0) {
      // Add a padding point at the end of the chart to ensure the line renders fully across the x axis
      const lastChartEdge = edges.value[edges.value.length - 1];
      d.push({x: lastChartEdge, y: d[d.length - 1].y});
      dates.push(lastChartEdge);
    }
    datasets.push({
      label,
      data: d,
      dates,
      [datasetSourceName]: name,
      _pluginColor: true,
    });
  }

  // Add baseline (prior period) datasets if enabled
  if (props.showBaseline) {
    for (const name of sourceList) {
      const device = baselineDevices[name];
      if (!device) continue;
      const deviceData = toValue(device.data);
      const d = deviceData.map((p, j) => ({x: chartLabels.value[j], y: p.y}));
      const dates = deviceData.map(p => p.x);
      if (d.length > 0) {
        // Add a padding point at the end of the chart to ensure the line renders fully across the x axis
        const lastChartEdge = edges.value[edges.value.length - 1];
        d.push({x: lastChartEdge, y: d[d.length - 1].y});
        dates.push(lastChartEdge);
      }
      datasets.push({
        label: (toValue(device.title) || name) + ' (prior)',
        data: d,
        dates,
        [datasetSourceName]: name,
        _pluginColor: true,
        isDashed: true, // Custom property for legend/plugin
        borderDash: [5, 5],
        pointRadius: 0,
        pointHoverRadius: 4,
        tension: 0.3,
        borderColor: 'transparent', // pluginColor will override this
        backgroundColor: 'transparent',
      });
    }
  }

  return {
    labels: chartLabels.value,
    datasets
  };
});

// Expose chart reference for parent component
defineExpose({
  legendItems,
  datasetNames,
});
</script>

<style scoped>

</style>
