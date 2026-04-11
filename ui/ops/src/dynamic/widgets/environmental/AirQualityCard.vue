<template>
  <v-card :loading="loading" class="d-flex flex-column">
    <v-toolbar color="transparent">
      <v-toolbar-title class="text-h4" style="overflow-wrap: break-word">{{ props.title }}</v-toolbar-title>
    </v-toolbar>
    <v-card-text class="flex-1-1-100 pt-0 flex-grow-1 d-flex">
      <div class="chart__container mb-n2 flex-grow-1">
        <bar :data="metricData" :options="metricOptions" :plugins="[]"/>
      </div>
    </v-card-text>
  </v-card>
</template>

<script setup>
import {defineChartOptions} from '@/components/charts/util.js';
import {metrics, statusToColor, useAirQuality, usePullAirQuality} from '@/traits/airQuality/airQuality.js';
import {cap, scale} from '@/util/number.js';
import {BarElement, CategoryScale, Chart as ChartJS, Legend, LinearScale, Title, Tooltip} from 'chart.js';
import Color from 'colorjs.io';
import {computed} from 'vue';
import {Bar} from 'vue-chartjs';
import {useTheme} from 'vuetify';

ChartJS.register(Title, Tooltip, Legend, BarElement, CategoryScale, LinearScale);

const props = defineProps({
  title: {
    type: String,
    default: 'Air Quality'
  },
  name: {
    type: String,
    default: ''
  }
});

const orderedMetrics = [
  'volatileOrganicCompounds', 'carbonDioxideLevel',
  'particulateMatter25', 'particulateMatter10', 'particulateMatter1',
  'infectionRisk', 'airChangePerHour'
];
const {value: airQuality, loading} = usePullAirQuality(() => props.name);
const {presentMetrics} = useAirQuality(airQuality);

const metricOptions = computed(() => {
  return defineChartOptions({
    maintainAspectRatio: false,
    borderRadius: 3,
    borderWidth: 1,
    interaction: {
      mode: 'index', // a single tooltip with all stacked datasets at the same x location in it
      intersect: false,
    },
    plugins: {
      legend: {
        display: false, // no legend needed
      }
    },
    scales: {
      y: {
        min: 0,
        max: 100,
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
          display: false
        },
      },
      x: {
        grid: {
          offset: false, // bars default to true here, put ticks back inline with grid lines
          color: '#fff1'
        },
        ticks: {
          color: '#fff',
          padding: 8,
        },
      }
    }
  });
});

const theme = useTheme();
const metricLabels = computed(() => {
  const dst = [];
  const src = presentMetrics.value;
  for (const m of orderedMetrics) {
    if (!src[m]) continue;
    dst.push(metrics[m].labelText);
  }
  return dst;
});
const metricData = computed(() => {
  const data = [];
  const colors = [];
  const src = presentMetrics.value;
  for (const m of orderedMetrics) {
    if (!src[m]) continue;
    const mInfo = metrics[m];
    data.push(cap(scale(src[m].value, mInfo.min, mInfo.max, 0, 100), 0, 100));
    colors.push(statusToColor(src[m].status));
  }
  const backgroundColors = colors.map(color => {
    if (!color) return 'rgba(0,0,0,0.1)';
    const c = new Color(theme.current.value.colors[color] ?? color);
    c.alpha = 0.5;
    return c.toString();
  });
  const borderColors = colors.map(color => {
    return theme.current.value.colors[color];
  });
  return {
    labels: metricLabels.value,
    datasets: [
      {
        label: 'Air Quality',
        data: data,
        backgroundColor: backgroundColors,
        borderColor: borderColors,
        borderWidth: 1
      }
    ]
  };
});

</script>

<style scoped>
.chart__container {
  min-height: 100%;
}
</style>