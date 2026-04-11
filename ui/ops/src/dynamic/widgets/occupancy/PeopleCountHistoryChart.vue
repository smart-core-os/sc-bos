<template>
  <div class="chart__container">
    <bar :data="chartData" :options="chartOptions" :plugins="[themeColorPlugin]"/>
    <div v-if="showNoData" class="no-data-overlay">
      <no-data-graphic class="no-data-graphic"/>
    </div>
  </div>
</template>

<script setup>
import {useDateScale, getTooltipDateFormat} from '@/components/charts/date.js';
import {useThemeColorPlugin} from '@/components/charts/plugins.js';
import {defineChartOptions} from '@/components/charts/util.js';
import {shiftFnFromStr} from '@/dynamic/widgets/occupancy/baseline.js';
import {useMaxPeopleCount} from '@/dynamic/widgets/occupancy/occupancy.js';
import {useLocalProp} from '@/util/vue.js';
import {BarElement, CategoryScale, Chart as ChartJS, Legend, LinearScale, LineElement, PointElement, TimeScale, Title, Tooltip} from 'chart.js'
import {startOfDay, startOfYear, format as fmtDate} from 'date-fns';
import {computed, ref, toRef, toValue, watch} from 'vue';
import {Bar} from 'vue-chartjs';
import 'chartjs-adapter-date-fns';
import NoDataGraphic from '@/dynamic/widgets/general/no-data-in-date-range.svg';

ChartJS.register(Title, Tooltip, BarElement, CategoryScale, LinearScale, TimeScale, Legend, LineElement, PointElement);

const props = defineProps({
  totalOccupancyName: {
    type: String,
    default: null
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
    default: false
  },
  // 'day', 'week', 'month' — how far back to shift the baseline
  baselineShift: {
    type: String,
    default: 'week'
  },
});

const _start = useLocalProp(toRef(props, 'start'));
const _end = useLocalProp(toRef(props, 'end'));
const _offset = useLocalProp(toRef(props, 'offset'));

const {edges, pastEdges, tickUnit, startDate, endDate} = useDateScale(_start, _end, _offset);

const shiftFn = computed(() => shiftFnFromStr(props.baselineShift));

// Conditionally use baseline comparison or just current data
const baselineEdges = computed(() => {
  if (!props.showBaseline) return pastEdges.value;
  return toValue(pastEdges).map(shiftFn.value);
});

const totalOccupancyCounts = useMaxPeopleCount(toRef(props, 'totalOccupancyName'), pastEdges);
const baselineCounts = useMaxPeopleCount(
    computed(() => props.showBaseline ? props.totalOccupancyName : null),
    baselineEdges
);

const {themeColorPlugin} = useThemeColorPlugin();

const chartOptions = computed(() => {
  return defineChartOptions({
    responsive: true,
    maintainAspectRatio: false,
    borderRadius: 3,
    borderWidth: 1,
    interaction: {
      mode: 'index', // a single tooltip with all stacked datasets at the same x location in it
      intersect: false,
    },
    plugins: {
      legend: {
        display: false, // we use a custom legend plugin and vue for this
      },
      tooltip: {
        callbacks: {
          title: () => '',
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
        stacked: false, // Don't stack when mixing bar and line
        title: {
          display: true,
          text: 'People Count'
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
        stacked: false, // Don't stack when mixing bar and line
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
  });
});
const chartLabels = computed(() => edges.value.slice(0, -1));
const chartData = computed(() => {
  const datasets = [];

  // Add baseline dataset first as a dotted line (if showing baseline)
  if (props.showBaseline && baselineCounts.value.length > 0) {
    datasets.push({
      type: 'line',
      label: 'Prior period',
      data: baselineCounts.value.map(data => data.y),
      dates: baselineCounts.value.map(data => data.x),
      backgroundColor: 'transparent',
      borderColor: '#aaaaaa',
      borderWidth: 2,
      borderDash: [5, 5], // Dotted line
      pointRadius: 0, // No points on the line
      pointHoverRadius: 4,
      fill: false,
      tension: 0.1, // Slight curve
    });
  }

  // Add current dataset as blue bars
  datasets.push({
    type: 'bar',
    label: props.showBaseline ? 'Current' : 'Total People Count',
    data: totalOccupancyCounts.value.map(data => data.y),
    dates: totalOccupancyCounts.value.map(data => data.x),
    backgroundColor: props.showBaseline ? '#2196F399' : undefined,
    borderColor: props.showBaseline ? '#2196F3' : undefined,
    borderWidth: props.showBaseline ? 1 : undefined,
  });

  return {
    labels: chartLabels.value,
    datasets
  };
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
</script>

<style scoped lang="scss">
.chart__container {
  min-height: 100%;
  /* The chart seems to have a padding no mater what we do, this gets rid of it */
  margin: -6px;
  position: relative;
}
</style>