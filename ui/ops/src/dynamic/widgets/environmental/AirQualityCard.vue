<template>
  <v-card :loading="loading" class="d-flex flex-column">
    <v-toolbar color="transparent">
      <v-toolbar-title class="text-h4" style="overflow-wrap: break-word">{{ props.title }}</v-toolbar-title>
    </v-toolbar>
    <v-card-text class="flex-1-1-100 pt-0 flex-grow-1 d-flex">
      <div class="chart__container mb-n2 flex-grow-1">
        <bar :data="metricData" :options="metricOptions" :plugins="[]"/>
        <chart-tooltip :data="tooltipData" :format-value="(y) => format(y) + '%'"/>
      </div>
    </v-card-text>
  </v-card>
</template>

<script setup>
import {defineChartOptions} from '@/components/charts/util.js';
import {useExternalTooltip} from '@/components/charts/plugins.js';
import ChartTooltip from '@/components/charts/ChartTooltip.vue';
import {metrics, statusToColor, useAirQuality, usePullAirQuality} from '@/traits/airQuality/airQuality.js';
import {cap, format, scale} from '@/util/number.js';
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

const {external: tooltipExternal, data: tooltipData} = useExternalTooltip();

const metricOptions = computed(() => {
  return defineChartOptions({
    maintainAspectRatio: false,
    borderRadius: 3,
    interaction: {
      mode: 'index', // a single tooltip with all stacked datasets at the same x location in it
      intersect: false,
    },
    plugins: {
      legend: {
        display: false, // no legend needed
      },
      tooltip: {
        enabled: false,
        external: tooltipExternal
      }
    },
    scales: {
      y: {
        min: 0,
        max: 100,
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
          color: 'rgba(255, 255, 255, 0.9)',
          padding: 8,
        },
      },
      x: {
        border: {
          display: false
        },
        grid: {
          display: false,
        },
        ticks: {
          color: 'rgba(255, 255, 255, 0.9)',
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