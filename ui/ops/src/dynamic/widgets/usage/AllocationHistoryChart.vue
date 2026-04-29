<template>
  <div class="chart__container">
    <bar :data="chartData" :options="chartOptions" :plugins="[themeColorPlugin, vueLegendPlugin]"/>
    <chart-tooltip :data="tooltipData" :edges="edges" :tick-unit="tickUnit"/>
  </div>
</template>
<script setup>

import {useDateScale} from '@/components/charts/date.js';
import {useExternalTooltip, useThemeColorPlugin, useVueLegendPlugin} from '@/components/charts/plugins.js';
import ChartTooltip from '@/components/charts/ChartTooltip.vue';
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
  title: {
    type: String,
    default: 'Resource Allocations'
  },
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

const {edges, pastEdges, tickUnit, startDate, endDate} = useDateScale(_start, _end, _offset);
const usageCount = useUsageCount(toRef(props, 'source'), pastEdges);
const totalUsageCounts = computed(() => usageCount?.value.results);
const nameMapping = computed(() => usageCount?.value.nameMapping);

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
        stacked: true,
        title: {
          display: true,
          text: props.title || 'Allocations',
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
        stacked: true,
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
  const datasets = [];
  const groupIds = new Set();

  for (const time of totalUsageCounts.value) {
    Object.keys(time.groups).forEach(id => groupIds.add(id));
  }

  for (const groupId of groupIds) {
    const data = [];
    for (let i = 0; i < totalUsageCounts.value.length; i++) {
      const time = totalUsageCounts.value[i];
      const count = time.groups[groupId] ?? 0;
      data.push({x: chartLabels.value[i], y: count});
    }
    datasets.push({
      label: groupId,
      data: data,
      [datasetSourceName]: groupId,
    });
  }

  return {
    labels: chartLabels.value,
    datasets: datasets,
  };
});

// Always use both plugins
const {legendItems, vueLegendPlugin} = useVueLegendPlugin();
const {themeColorPlugin} = useThemeColorPlugin();
const {external: tooltipExternal, data: tooltipData} = useExternalTooltip();

// Expose chart reference for parent component
defineExpose({
  legendItems,
  datasetNames,
  nameMapping,
});
</script>

<style scoped>

</style>