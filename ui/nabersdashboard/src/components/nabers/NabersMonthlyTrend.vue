<template>
  <div class="trend-wrapper">
    <div class="trend-header">
      <span class="trend-title">Monthly energy intensity</span>
      <span class="trend-unit">kWh/m² NIA / month</span>
    </div>
    <div v-if="hasData" class="trend-chart">
      <Line :data="chartData" :options="chartOptions"/>
    </div>
    <div v-else class="trend-empty">
      Insufficient meter history — data will appear as months accumulate.
    </div>
    <div class="trend-desc">
      Rolling 12-month energy intensity for lighting and equipment. Dashed white and green lines show the monthly equivalent of DfP Stage 4 annual targets.
      <span v-if="hasEstimated" class="estimated-note">Amber triangles mark months resting on projected meter data.</span>
    </div>
  </div>
</template>

<script setup>
import {computed} from 'vue';
import {Line} from 'vue-chartjs';
import {
  Chart as ChartJS,
  CategoryScale,
  LinearScale,
  LineElement,
  PointElement,
  Title,
  Tooltip,
  Legend,
  Filler
} from 'chart.js';

ChartJS.register(CategoryScale, LinearScale, LineElement, PointElement, Title, Tooltip, Legend, Filler);

const props = defineProps({
  months:          {type: Array, default: () => []},
  lightingTarget:  {type: Number, default: null},
  equipmentTarget: {type: Number, default: null}
});

const hasData = computed(() =>
  props.months.some(m => m.lightingIntensity !== null || m.equipmentIntensity !== null)
);

/**
 * @param {number} i
 * @return {boolean}
 */
const isEstimated = (i) => props.months[i]?.quality === 'estimated';

const hasEstimated = computed(() => props.months.some(m => m.quality === 'estimated'));

/**
 * Marks for months resting on a projected reading.
 *
 * The series stay their own colours and only take a dash: both DfP target lines
 * are already dashed in other colours, so recolouring a data segment would read
 * as a third reference line.
 *
 * @param {string} colour the series' own border colour
 * @return {Object} chart.js dataset fragment
 */
function estimatedMarks(colour) {
  return {
    pointRadius:          ctx => (isEstimated(ctx.dataIndex) ? 5 : 3),
    pointStyle:           ctx => (isEstimated(ctx.dataIndex) ? 'triangle' : 'circle'),
    pointBackgroundColor: ctx => (isEstimated(ctx.dataIndex) ? '#f7a12c' : colour),
    pointBorderColor:     ctx => (isEstimated(ctx.dataIndex) ? '#f7a12c' : colour),
    segment: {
      borderDash: ctx => (isEstimated(ctx.p0DataIndex) || isEstimated(ctx.p1DataIndex))
        ? [6, 4]
        : undefined
    }
  };
}

// Monthly targets are the annual figure ÷ 12
const lightingMonthlyTarget  = computed(() => props.lightingTarget  !== null ? props.lightingTarget  / 12 : null);
const equipmentMonthlyTarget = computed(() => props.equipmentTarget !== null ? props.equipmentTarget / 12 : null);

const chartData = computed(() => {
  const labels    = props.months.map(m => m.label);
  const datasets  = [
    {
      label:           'Lighting',
      data:            props.months.map(m => m.lightingIntensity),
      borderColor:     '#9c40ff',
      backgroundColor: 'rgba(127,0,255,0.12)',
      borderWidth:     2,
      tension:         0.3,
      fill:            true,
      spanGaps:        false,
      ...estimatedMarks('#9c40ff')
    },
    {
      label:           'Equipment',
      data:            props.months.map(m => m.equipmentIntensity),
      borderColor:     '#5c0099',
      backgroundColor: 'rgba(61,0,128,0.12)',
      borderWidth:     2,
      tension:         0.3,
      fill:            true,
      spanGaps:        false,
      ...estimatedMarks('#5c0099')
    }
  ];

  if (lightingMonthlyTarget.value !== null) {
    datasets.push({
      label:       'Lighting DfP target',
      data:        props.months.map(() => lightingMonthlyTarget.value),
      borderColor: 'rgba(255,255,255,0.75)',
      borderWidth: 2,
      borderDash:  [5, 4],
      pointRadius: 0,
      fill:        false
    });
  }
  if (equipmentMonthlyTarget.value !== null) {
    datasets.push({
      label:       'Equipment DfP target',
      data:        props.months.map(() => equipmentMonthlyTarget.value),
      borderColor: 'rgba(76,175,80,0.85)',
      borderWidth: 2,
      borderDash:  [5, 4],
      pointRadius: 0,
      fill:        false
    });
  }

  return {labels, datasets};
});

const chartOptions = {
  responsive:          true,
  maintainAspectRatio: false,
  interaction:         {mode: 'index', intersect: false},
  plugins: {
    legend: {
      labels: {color: 'rgba(255,255,255,0.7)', font: {family: 'Poppins', size: 12}}
    },
    tooltip: {
      callbacks: {
        label: ctx => ctx.parsed.y !== null
          ? ` ${ctx.dataset.label}: ${ctx.parsed.y.toFixed(2)} kWh/m²`
          : null
      }
    }
  },
  scales: {
    x: {
      ticks: {color: 'rgba(255,255,255,0.6)', font: {family: 'Poppins', size: 11}},
      grid:  {color: 'rgba(255,255,255,0.06)'}
    },
    y: {
      ticks:       {color: 'rgba(255,255,255,0.6)', font: {family: 'Poppins', size: 11}},
      grid:        {color: 'rgba(255,255,255,0.06)'},
      beginAtZero: true
    }
  }
};
</script>

<style scoped>
.trend-wrapper {
  background:     rgba(255, 255, 255, 0.05);
  border:         1px solid rgba(255, 255, 255, 0.08);
  border-radius:  12px;
  padding:        20px 24px;
  display:        flex;
  flex-direction: column;
  gap:            12px;
  height:         100%;
  box-sizing:     border-box;
}

.trend-header {
  display:     flex;
  align-items: baseline;
  gap:         12px;
  flex-shrink: 0;
}

.trend-title {
  font-size:      12px;
  font-weight:    500;
  color:          rgba(255, 255, 255, 0.45);
  text-transform: uppercase;
  letter-spacing: 0.08em;
}

.trend-unit {
  font-size: 12px;
  color:     rgba(255, 255, 255, 0.25);
}

.trend-chart {
  flex:       1;
  min-height: 0;
}

.trend-empty {
  flex:       1;
  font-size:  14px;
  color:      rgba(255, 255, 255, 0.3);
  text-align: center;
  padding:    32px 0;
  font-style: italic;
}

.trend-desc {
  font-size:   13px;
  color:       rgba(255, 255, 255, 0.35);
  line-height: 1.4;
  flex-shrink: 0;
}

.estimated-note {
  color: rgba(247, 161, 44, 0.7);
}
</style>
