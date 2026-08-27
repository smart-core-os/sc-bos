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
      Rolling 12-month base building energy intensity. Green and amber dashed lines show monthly equivalents of the 5-star and 4-star thresholds.
      <template v-if="hasEstimated">
        <span class="estimated-note">Amber triangles, joined by a dashed purple line, mark months resting on projected meter data.</span>
      </template>
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
  months:      {type: Array,  default: () => []},
  fiveStarMax: {type: Number, default: null},
  fourStarMax: {type: Number, default: null}
});

const hasData = computed(() => props.months.some(m => m.totalIntensity !== null));

const hasEstimated = computed(() => props.months.some(m => m.quality === 'estimated'));

const fiveStarMonthly = computed(() => props.fiveStarMax !== null ? props.fiveStarMax / 12 : null);
const fourStarMonthly = computed(() => props.fourStarMax !== null ? props.fourStarMax / 12 : null);

/**
 * Whether the month at a data index rests on a projected meter reading.
 *
 * @param {number} i
 * @return {boolean}
 */
const isEstimated = (i) => props.months[i]?.quality === 'estimated';

const chartData = computed(() => {
  const labels   = props.months.map(m => m.label);
  const datasets = [
    {
      label:           'Base Building',
      data:            props.months.map(m => m.totalIntensity),
      borderColor:     '#7F00FF',
      backgroundColor: 'rgba(127,0,255,0.12)',
      borderWidth:     2,
      // Estimated months are drawn amber and larger, so a reader scanning the
      // line sees which points are not measurements without reading a legend.
      pointRadius:     ctx => (isEstimated(ctx.dataIndex) ? 5 : 3),
      pointBackgroundColor: ctx => (isEstimated(ctx.dataIndex) ? '#f7a12c' : '#7F00FF'),
      pointBorderColor: ctx => (isEstimated(ctx.dataIndex) ? '#f7a12c' : '#7F00FF'),
      pointStyle:      ctx => (isEstimated(ctx.dataIndex) ? 'triangle' : 'circle'),
      // A segment is dashed when either month it joins was estimated: the
      // segment's own slope is only as trustworthy as its worse end.
      //
      // Dashed but still purple, deliberately. The 4-star threshold below is
      // already amber and dashed, so an amber dashed data segment would read as
      // a second threshold line. Keeping the series colour and marking the
      // points amber separates "estimated data" from "reference line".
      segment: {
        borderDash: ctx => (isEstimated(ctx.p0DataIndex) || isEstimated(ctx.p1DataIndex))
          ? [6, 4]
          : undefined
      },
      tension:         0.3,
      fill:            true,
      // Still false: a month with no data at all stays a break in the line.
      // Estimation fills what it honestly can, and what it cannot stays absent.
      spanGaps:        false
    }
  ];
  if (fiveStarMonthly.value !== null) {
    datasets.push({
      label:       '5-star threshold',
      data:        props.months.map(() => fiveStarMonthly.value),
      borderColor: 'rgba(76,175,80,0.8)',
      borderWidth: 2,
      borderDash:  [5, 4],
      pointRadius: 0,
      fill:        false
    });
  }
  if (fourStarMonthly.value !== null) {
    datasets.push({
      label:       '4-star threshold',
      data:        props.months.map(() => fourStarMonthly.value),
      borderColor: 'rgba(247,161,44,0.7)',
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
        label: ctx => {
          if (ctx.parsed.y === null) return null;
          const base = ` ${ctx.dataset.label}: ${ctx.parsed.y.toFixed(2)} kWh/m²`;
          // Only the measured series can be estimated; the threshold lines are
          // computed from the benchmark and are never meter-derived.
          if (ctx.datasetIndex !== 0 || !isEstimated(ctx.dataIndex)) return base;
          const pct = props.months[ctx.dataIndex]?.estimatedPct ?? 0;
          return `${base} (estimated — ${pct < 0.1 ? '<0.1' : pct.toFixed(0)}% of this month)`;
        }
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
  flex:        1;
  font-size:   14px;
  color:       rgba(255, 255, 255, 0.3);
  text-align:  center;
  padding:     32px 0;
  font-style:  italic;
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
