<template>
  <v-card class="d-flex flex-column">
    <v-toolbar color="transparent">
      <v-toolbar-title class="text-h4" style="overflow-wrap: break-word">{{ props.title }}</v-toolbar-title>
      <v-spacer/>
      <v-chip
          v-if="summaryPct !== null"
          :color="summaryPct > 0 ? 'error-lighten-1' : 'success-lighten-1'"
          size="small"
          label
          class="flex-shrink-0">
        <v-icon start :icon="summaryPct > 0 ? 'mdi-trending-up' : 'mdi-trending-down'"/>
        {{ Math.abs(summaryPct).toFixed(1) }}% vs prior period
      </v-chip>
    </v-toolbar>
    <v-card-text class="flex-grow-1 d-flex pt-0">
      <div class="chart__container flex-grow-1">
        <bar :data="chartData" :options="chartOptions" :plugins="[]"/>
      </div>
    </v-card-text>
    <div class="d-flex ga-8 justify-center pb-3 text-caption opacity-70">
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
import {useDateScale, getTooltipDateFormat} from '@/components/charts/date.js';
import {defineChartOptions} from '@/components/charts/util.js';
import {usePeriod} from '@/composables/time.js';
import {useEnergyNormalized, subDays, subWeeks, subMonths} from '@/dynamic/widgets/meter/baseline.js';
import {useLocalProp} from '@/util/vue.js';
import {BarElement, CategoryScale, Chart as ChartJS, Legend, LinearScale, LineElement, PointElement, Title, Tooltip} from 'chart.js';
import {format as fmtDate} from 'date-fns';
import {format} from '@/util/number.js';
import {computed, toRef} from 'vue';
import {Bar} from 'vue-chartjs';
import * as vColors from 'vuetify/util/colors';

ChartJS.register(Title, Tooltip, Legend, BarElement, CategoryScale, LinearScale, LineElement, PointElement);

const props = defineProps({
  title: {type: String, default: 'Energy vs Baseline'},
  name: {type: String, default: ''},
  unit: {type: String, default: 'kWh'},
  period: {type: String, default: 'week'},
  offset: {type: [Number, String], default: 0},
  // 'day', 'week', 'month' — how far back to shift the baseline
  baselineShift: {type: String, default: 'week'},
});

const shiftFn = computed(() => {
  switch (props.baselineShift) {
    case 'month': return (d) => subMonths(d, 1);
    case 'week': return (d) => subWeeks(d, 1);
    default: return (d) => subDays(d, 7);
  }
});

const _offset = computed(() => -Math.abs(parseInt(props.offset)));
const {start, end} = usePeriod(toRef(props, 'period'), toRef(props, 'period'), _offset);
const {edges, pastEdges, tickUnit} = useDateScale(start, end, useLocalProp(toRef(props, 'offset')));

const {currentConsumption, baselineConsumption, summaryPct} = useEnergyNormalized(
  toRef(props, 'name'),
  pastEdges,
  shiftFn.value
);

const currentColor = vColors.blue.base;
const baselineColor = '#aaaaaa';

const tickFmt = computed(() => {
  switch (tickUnit.value) {
    case 'minute':
    case 'hour':   return 'H:mm';
    case 'month':  return 'MMM';
    case 'year':   return 'yyyy';
    default:       return 'd MMM'; // day
  }
});

const chartLabels = computed(() =>
  edges.value.slice(0, -1).map(d => fmtDate(d, tickFmt.value))
);

const chartData = computed(() => ({
  labels: chartLabels.value,
  datasets: [
    {
      type: 'line',
      label: 'Prior period',
      data: baselineConsumption.value.map(pt => pt.y),
      dates: baselineConsumption.value.map(pt => pt.x),
      backgroundColor: 'transparent',
      borderColor: baselineColor,
      borderWidth: 2,
      borderDash: [5, 5], // Dotted line
      pointRadius: 0, // No points on the line
      pointHoverRadius: 4,
      fill: false,
      tension: 0.1, // Slight curve
    },
    {
      type: 'bar',
      label: 'Current',
      data: currentConsumption.value.map(pt => pt.y),
      dates: currentConsumption.value.map(pt => pt.x),
      backgroundColor: currentColor + '99',
      borderColor: currentColor,
      borderWidth: 1,
    },
  ],
}));

const chartOptions = computed(() => defineChartOptions({
  maintainAspectRatio: false,
  interaction: {
    mode: 'index',
    intersect: false,
  },
  plugins: {
    legend: {display: false},
    tooltip: {
      callbacks: {
        title: () => {
          return '';
        },
        label: (ctx) => {
          const dates = ctx.dataset.dates;
          const date = dates ? dates[ctx.dataIndex] : null;
          const fmt = getTooltipDateFormat(tickUnit.value);
          const dateStr = date ? ` (${fmtDate(date, fmt)})` : '';
          return `${ctx.dataset.label}${dateStr}: ${ctx.parsed.y != null ? format(ctx.parsed.y) : '—'} ${props.unit}`;
        },
      },
    },
  },
  scales: {
    y: {
      beginAtZero: true,
      title: {display: true, text: props.unit, color: '#fff8'},
      border: {color: 'transparent'},
      grid: {
        color(ctx) {
          return ctx.tick.value === 0 ? '#fff4' : '#fff1';
        },
        drawTicks: false,
      },
      ticks: {color: '#fff', padding: 8},
    },
    x: {
      grid: {color: '#fff1'},
      ticks: {color: '#fff', padding: 8, maxRotation: 0},
    },
  },
}));
</script>

<style scoped>
.chart__container {
  min-height: 160px;
}

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
