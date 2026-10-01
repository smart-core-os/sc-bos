<template>
  <div class="chart-outer">
    <div class="chart-container">
      <Bar :data="chartData" :options="chartOptions"/>
    </div>
    <div class="chart-desc">
      Actual annualised lighting and equipment energy intensity compared against the
      configured NABERS DfP benchmarks{{ targetSummary }}.
    </div>
  </div>
</template>

<script setup>
import {computed} from 'vue';
import {Bar} from 'vue-chartjs';
import {
  Chart as ChartJS,
  CategoryScale,
  LinearScale,
  BarElement,
  Title,
  Tooltip,
  Legend
} from 'chart.js';

ChartJS.register(CategoryScale, LinearScale, BarElement, Title, Tooltip, Legend);

const props = defineProps({
  lightingActual:  {type: Number, default: null},
  equipmentActual: {type: Number, default: null},
  // Per-building figures from the DfP assessment, so there is no default to
  // fall back on: an unconfigured benchmark draws no reference bar.
  lightingTarget:  {type: Number, default: null},
  equipmentTarget: {type: Number, default: null}
});

// The caption quotes whatever was configured rather than naming fixed figures.
const targetSummary = computed(() => {
  const parts = [];
  if (props.lightingTarget !== null) parts.push(`lighting ${props.lightingTarget}`);
  if (props.equipmentTarget !== null) parts.push(`equipment ${props.equipmentTarget}`);
  return parts.length ? ` (${parts.join(', ')} kWh/m²/yr)` : '';
});

const chartData = computed(() => ({
  labels:   ['Lighting', 'Equipment'],
  datasets: [
    {
      label:           'Actual (annualised)',
      data:            [props.lightingActual, props.equipmentActual],
      backgroundColor: '#7F00FF',
      borderRadius:    4
    },
    {
      label:           'NABERS DfP Target',
      data:            [props.lightingTarget, props.equipmentTarget],
      backgroundColor: 'rgba(127, 0, 255, 0.2)',
      borderColor:     '#7F00FF',
      borderWidth:     2,
      borderRadius:    4
    }
  ]
}));

const chartOptions = {
  responsive:          true,
  maintainAspectRatio: false,
  plugins: {
    legend: {
      labels: {color: 'rgba(255,255,255,0.85)', font: {family: 'Poppins', size: 13}}
    },
    tooltip: {
      callbacks: {
        label: ctx => ` ${ctx.dataset.label}: ${ctx.parsed.y !== null ? ctx.parsed.y.toFixed(1) : '—'} kWh/m² NIA/yr`
      }
    }
  },
  scales: {
    x: {
      ticks: {color: 'rgba(255,255,255,0.85)', font: {family: 'Poppins', size: 13}},
      grid:  {color: 'rgba(255,255,255,0.08)'}
    },
    y: {
      ticks: {color: 'rgba(255,255,255,0.85)', font: {family: 'Poppins', size: 13}},
      grid:  {color: 'rgba(255,255,255,0.08)'},
      title: {
        display: true,
        text:    'kWh/m² NIA/yr',
        color:   'rgba(255,255,255,0.5)',
        font:    {family: 'Poppins', size: 12}
      },
      beginAtZero: true
    }
  }
};
</script>

<style scoped>
.chart-outer {
  display:        flex;
  flex-direction: column;
  height:         100%;
  min-height:     0;
}

.chart-container {
  position:   relative;
  flex:        1;
  min-height:  0;
  width:       100%;
}

.chart-desc {
  font-size:   13px;
  color:       rgba(255, 255, 255, 0.35);
  margin-top:  10px;
  line-height: 1.4;
  flex-shrink: 0;
}
</style>
