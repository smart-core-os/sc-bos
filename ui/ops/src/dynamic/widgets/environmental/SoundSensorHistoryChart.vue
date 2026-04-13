<template>
  <div class="chart__container">
    <line-chart :data="chartData" :options="chartOptions" :plugins="[themeColorPlugin, vueLegendPlugin]"/>
  </div>
</template>

<script setup>
import {useDateScale, getTooltipDateFormat} from '@/components/charts/date.js';
import {useThemeColorPlugin, useVueLegendPlugin} from '@/components/charts/plugins.js';
import {defineChartOptions} from '@/components/charts/util.js';
import {useSoundLevelHistoryMetrics} from '@/dynamic/widgets/environmental/soundSensor.js';
import {shiftFnFromStr} from '@/dynamic/widgets/occupancy/baseline.js';
import {useLocalProp} from '@/util/vue.js';
import {sentenceCase} from 'change-case';
import {Chart as ChartJS, Legend, LinearScale, LineElement, PointElement, TimeScale, Title, Tooltip} from 'chart.js';
import {startOfDay, startOfYear, format as fmtDate} from 'date-fns';
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

const {edges, pastEdges, tickUnit} = useDateScale(_start, _end, _offset);

const shiftFn = computed(() => shiftFnFromStr(props.baselineShift));
const baselineEdges = computed(() => {
  if (!props.showBaseline) return pastEdges.value;
  return toValue(pastEdges).map(shiftFn.value);
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
const chartOptions = computed(() => {
  return defineChartOptions({
    responsive: true,
    maintainAspectRatio: false,
    borderRadius: 3,
    borderWidth: 1,
    interaction: {
      mode: 'index', // a single tooltip with all stacked datasets at the same x location in it
    },
    plugins: {
      legend: {
        display: false, // we use a custom legend plugin and vue for this
      },
      tooltip: {
        callbacks: {
          label: (ctx) => {
            const dates = ctx.dataset.dates;
            const date = dates ? dates[ctx.dataIndex] : null;
            const fmt = getTooltipDateFormat(tickUnit.value);
            const dateStr = date ? ` (${fmtDate(date, fmt)})` : '';
            return `${ctx.dataset.label}${dateStr}: ${ctx.parsed.y != null ? new Intl.NumberFormat(undefined, {}).format(ctx.parsed.y) : '—'}`;
          }
        }
      }
    },
    scales: {
      y: {
        stacked: false,
        beginAtZero: true,
        title: {
          display: true,
          text: yAxisLabel.value
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
  let datasets = [];
  const sourceList = sources.value;

  // Add current period datasets
  for (const name of sourceList) {
    const device = devices[name];
    if (!device) continue;
    const label = toValue(device.title) || name;
    const deviceData = toValue(device.data);
    datasets.push({
      label,
      data: deviceData.map(d => d.y),
      dates: deviceData.map(d => d.x),
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
      datasets.push({
        label: (toValue(device.title) || name) + ' (prior)',
        data: deviceData.map(d => d.y),
        dates: deviceData.map(d => d.x),
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
