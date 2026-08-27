<template>
  <div class="donut-wrapper">
    <div class="donut-title">Energy split</div>
    <div v-if="hasData" class="donut-container">
      <Doughnut :data="chartData" :options="chartOptions"/>
      <div class="donut-centre">
        <div class="centre-pct">{{ lightingPct }}%</div>
        <div class="centre-label">lighting</div>
      </div>
    </div>
    <div v-else class="donut-empty">&mdash;</div>
    <div v-if="hasData" class="donut-legend">
      <span class="legend-dot" style="background:#7F00FF"/>
      <span class="legend-text">Lighting {{ lightingPct }}%</span>
      <span class="legend-dot legend-dot--equip"/>
      <span class="legend-text">Equipment {{ equipmentPct }}%</span>
    </div>
    <div class="donut-desc">
      Proportional breakdown of total energy intensity between lighting and equipment circuits.
    </div>
  </div>
</template>

<script setup>
import {computed} from 'vue';
import {Doughnut} from 'vue-chartjs';
import {Chart as ChartJS, ArcElement, DoughnutController, Tooltip, Legend} from 'chart.js';

ChartJS.register(ArcElement, DoughnutController, Tooltip, Legend);

const props = defineProps({
  lightingKwh:  {type: Number, default: null},
  equipmentKwh: {type: Number, default: null}
});

const hasData = computed(() => props.lightingKwh !== null && props.equipmentKwh !== null);

const total = computed(() => (props.lightingKwh ?? 0) + (props.equipmentKwh ?? 0));

const lightingPct = computed(() =>
  total.value > 0 ? Math.round((props.lightingKwh / total.value) * 100) : 0
);
const equipmentPct = computed(() => 100 - lightingPct.value);

const chartData = computed(() => ({
  labels:   ['Lighting', 'Equipment'],
  datasets: [{
    data:            [props.lightingKwh ?? 0, props.equipmentKwh ?? 0],
    backgroundColor: ['#7F00FF', '#3d0080'],
    borderColor:     ['#7F00FF', '#3d0080'],
    borderWidth:     0,
    hoverOffset:     4
  }]
}));

const chartOptions = {
  responsive:          true,
  maintainAspectRatio: true,
  cutout:              '68%',
  plugins: {
    legend: {display: false},
    tooltip: {
      callbacks: {
        label: ctx => ` ${ctx.label}: ${ctx.parsed.toFixed(1)} kWh/m² NIA/yr`
      }
    }
  }
};
</script>

<style scoped>
.donut-wrapper {
  background:     rgba(255, 255, 255, 0.05);
  border:         1px solid rgba(255, 255, 255, 0.08);
  border-radius:  12px;
  padding:        20px 24px;
  display:        flex;
  flex-direction: column;
  gap:            12px;
  flex:           1;
}

.donut-title {
  font-size:      11px;
  font-weight:    500;
  color:          rgba(255, 255, 255, 0.45);
  text-transform: uppercase;
  letter-spacing: 0.08em;
}

.donut-container {
  position: relative;
  width:    220px;
  height:   220px;
  margin:   0 auto;
}

.donut-centre {
  position:        absolute;
  inset:           0;
  display:         flex;
  flex-direction:  column;
  align-items:     center;
  justify-content: center;
  pointer-events:  none;
}

.centre-pct {
  font-size:   36px;
  font-weight: 300;
  color:       #ffffff;
  line-height: 1;
}

.centre-label {
  font-size: 13px;
  color:     rgba(255, 255, 255, 0.45);
}

.donut-empty {
  font-size:   36px;
  font-weight: 300;
  color:       rgba(255, 255, 255, 0.25);
  text-align:  center;
  padding:     32px 0;
}

.donut-legend {
  display:     flex;
  align-items: center;
  gap:         8px;
  flex-wrap:   wrap;
}

.legend-dot {
  width:         10px;
  height:        10px;
  border-radius: 50%;
  background:    #7F00FF;
  flex-shrink:   0;
}

.legend-dot--equip {
  background: #3d0080;
}

.legend-text {
  font-size:    13px;
  color:        rgba(255, 255, 255, 0.55);
  margin-right: 8px;
}

.donut-desc {
  font-size:   13px;
  color:       rgba(255, 255, 255, 0.35);
  margin-top:  auto;
  line-height: 1.4;
}
</style>
