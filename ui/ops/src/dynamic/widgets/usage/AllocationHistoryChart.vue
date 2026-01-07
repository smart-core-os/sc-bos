<template>
  <div>
    <bar :data="chartData" :options="chartOptions" :plugins="[themeColorPlugin, vueLegendPlugin]"/>
  </div>
</template>
<script setup>

import {useDateScale} from '@/components/charts/date.js';
import {useThemeColorPlugin, useVueLegendPlugin} from '@/components/charts/plugins.js';
import {defineChartOptions} from '@/components/charts/util.js';
import {useUsageCount} from '@/dynamic/widgets/usage/usage.js';
import {useLocalProp} from '@/util/vue.js';
import {BarElement, Chart as ChartJS, Legend, LinearScale, TimeScale, Title, Tooltip} from 'chart.js';
import {startOfDay, startOfYear} from 'date-fns';
import 'chartjs-adapter-date-fns';
import {computed, toRef} from 'vue';
import {Bar} from 'vue-chartjs';

const datasetSourceName = Symbol('datasetSourceName');

ChartJS.register(Title, Tooltip, BarElement, LinearScale, TimeScale, Legend);

const props = defineProps({
  source: {
    type: [String, Array],
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
  }
});

const _start = useLocalProp(toRef(props, 'start'));
const _end = useLocalProp(toRef(props, 'end'));
const _offset = useLocalProp(toRef(props, 'offset'));

const {edges, pastEdges, tickUnit} = useDateScale(_start, _end, _offset);
const totalUsageCounts = useUsageCount(toRef(props, 'source'), pastEdges);

const {themeColorPlugin} = useThemeColorPlugin();

const datasetNames = computed(() => {
  return chartData.value.datasets.map(item => {
    return item[datasetSourceName];
  });
});

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
      }
    },
    scales: {
      y: {
        stacked: true,
        title: {
          display: true,
          text: 'Locker Usage'
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
  });
});
const chartLabels = computed(() => edges.value.slice(0, -1));
const chartData = computed(() => {
  const datasets = [];
  const groupIds = new Set();

  for (const time of totalUsageCounts.value) {
    Object.keys(time.groups).forEach(id => groupIds.add(id));
  }

  for (const groupId of groupIds) {
    const data = [];
    for (const time of totalUsageCounts.value) {
      const count = time.groups[groupId] ?? 0;
      data.push(count);
    }
    datasets.push({
      label: groupId,
      data: data,
    });
  }

  return {
    labels: chartLabels.value,
    datasets: datasets,
  };
});

// Always use both plugins
const {legendItems, vueLegendPlugin} = useVueLegendPlugin();

// Expose chart reference for parent component
defineExpose({
  legendItems,
  datasetNames,
});
</script>

<style scoped>

</style>