<template>
  <div class="d-flex flex-column">
    <div class="d-flex flex-row flex-nowrap justify-end align-center mb-6">
      <v-card-title class="text-h4 pa-0 mr-auto pl-4">{{ props.chartTitle }}</v-card-title>
      <div v-if="!props.hideLegends" class="legend">
        <v-checkbox
            v-for="(item, index) in legendItems"
            :key="index"
            :model-value="!item.hidden"
            @update:model-value="item.onClick"
            :label="item.text"
            :color="item.bgColor"
            hide-details
            class="mt-0"/>
      </div>
      <template v-if="$slots.options">
        <v-divider vertical class="ml-6 mr-2"/>
        <span>
          <slot name="options"/>
        </span>
      </template>
    </div>
    <div class="flex-grow-1">
      <line-chart-generator
          :options="props.chartOptions"
          :data="props.chartData"
          :plugins="[vueLegendPlugin, ...props.plugins]"
          :dataset-id-key="props.datasetIdKey"/>
    </div>
    <slot name="tooltip"/>
  </div>
</template>

<script setup>
import {
  CategoryScale,
  Chart as ChartJS,
  Filler,
  Legend,
  LinearScale,
  LineElement,
  PointElement,
  TimeScale,
  Title,
  Tooltip
} from 'chart.js';
import {ref} from 'vue';
import {Line as LineChartGenerator} from 'vue-chartjs';
import 'chartjs-adapter-date-fns'; // imported for side effects

ChartJS.register(Title, Tooltip, Legend, LineElement, LinearScale, TimeScale, Filler, CategoryScale, PointElement);

const props = defineProps({
  datasetIdKey: {
    type: String,
    default: 'x'
  },
  hideLegends: {
    type: Boolean,
    default: false
  },
  chartTitle: {
    type: String,
    default: ''
  },
  chartData: {
    type: Object,
    default: () => {
      return {};
    }
  },
  chartOptions: {
    type: Object,
    default: () => {
      return {};
    }
  },
  plugins: {
    type: Array,
    default: () => []
  }
});

/**
 * Helper to give type assistance to chart.js plugins.
 *
 * @template {import('chart.js').Plugin} T
 * @param {T} plugin
 * @return {T}
 */
const definePlugin = (plugin) => plugin;

const legendItems = ref([]);
const vueLegendPlugin = definePlugin({
  id: 'vueLegend',
  afterUpdate(chart) {
    const items = chart.options.plugins.legend.labels.generateLabels(chart);
    legendItems.value = items.map((item) => {
      return {
        text: item.text,
        hidden: item.hidden,
        bgColor: item.strokeStyle,
        onClick: (e) => {
          const {type} = chart.config;
          if (type === 'pie' || type === 'doughnut') {
            // Pie and doughnut charts only have a single dataset and visibility is per item
            chart.setDatasetVisibility(item.index, e);
          } else {
            chart.setDatasetVisibility(item.datasetIndex, e);
          }
          chart.update();
        }
      };
    });
  }
});
</script>

<style scoped>
.legend {
  display: inline-flex;
  gap: 24px;
}
</style>
