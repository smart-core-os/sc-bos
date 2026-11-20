<template>
  <div class="donut-wrapper">
    <div class="donut-title">Energy source split</div>
    <div v-if="hasData" class="donut-container">
      <Doughnut :data="chartData" :options="chartOptions"/>
      <div class="donut-centre">
        <div class="centre-pct">{{ pvPct }}%</div>
        <div class="centre-label">solar</div>
      </div>
    </div>
    <div v-else class="donut-empty">&mdash;</div>
    <div v-if="hasData" class="donut-legend">
      <span class="legend-dot" style="background:#7F00FF"/>
      <span class="legend-text">Grid {{ gridPct }}%</span>
      <span class="legend-dot legend-dot--pv"/>
      <span class="legend-text">PV {{ pvPct }}%</span>
    </div>

    <!--
      Two references, because they answer different questions and the obvious one
      answers the wrong question. The share moves with demand: a building over its
      design consumption reads below the design's solar share while its array
      performs perfectly. Only the generation figure compares like for like, so
      that is the one that carries the colour.
    -->
    <div v-if="hasData && dfpPvSharePct !== null" class="donut-target">
      <span>Design {{ dfpPvSharePct.toFixed(1) }}% solar</span>
      <span
          v-if="generationLabel"
          class="target-generation"
          :style="{color: generationColor}"
          :title="generationTitle">
        Generation {{ generationLabel }}
      </span>
    </div>
    <!--
      The window is stated because the split is seasonal and the figure is not
      self-describing. A week of August and a rolling twelve months both render
      as an annual intensity, and solar's share of the first is nothing like its
      share of the second.
    -->
    <div class="donut-desc">
      Net imported electricity vs on-site PV generation as a proportion of gross energy demand.
      <template v-if="hasData">Measured over {{ measuredOver.label }}.</template>
    </div>
  </div>
</template>

<script setup>
import {computed} from 'vue';
import {Doughnut} from 'vue-chartjs';
import {Chart as ChartJS, ArcElement, DoughnutController, Tooltip, Legend} from 'chart.js';
import {measuredWindow} from '@/util/measuredWindow.js';
import {DFP_SEVERITY_COLOR} from '@/util/dfpSeverity.js';

ChartJS.register(ArcElement, DoughnutController, Tooltip, Legend);

const props = defineProps({
  grossKwh: {type: Number, default: null},
  pvKwh:    {type: Number, default: null},
  /**
   * Which window the two figures annualise. The store picks it once for the
   * whole dashboard, so this widget, the breakdown chart and the rating gauge
   * describe the same year.
   *
   * @type {'months'|'period'|null}
   */
  intensityBasis:      {type: String, default: 'period'},
  monthsOfData:        {type: Number, default: 0},
  trailingDaysCovered: {type: Number, default: 0},
  elapsedDays:         {type: Number, default: 0},
  /** The generation the DfP design assumed, kWh/m²·pa. */
  dfpPvIntensity:      {type: Number, default: null},
  /** That generation as a share of the design's own gross demand, percent. */
  dfpPvSharePct:       {type: Number, default: null}
});

const measuredOver = computed(() => measuredWindow(props));

const hasData = computed(() => props.grossKwh !== null && props.grossKwh > 0);

const gridKwh  = computed(() => Math.max(0, (props.grossKwh ?? 0) - (props.pvKwh ?? 0)));
const total    = computed(() => props.grossKwh ?? 0);
/** The exact share. `pvPct` rounds it for display; comparisons use this. */
const pvSharePct = computed(() => total.value > 0 ? ((props.pvKwh ?? 0) / total.value) * 100 : 0);
const pvPct    = computed(() => Math.round(pvSharePct.value));
const gridPct  = computed(() => 100 - pvPct.value);

/** Measured generation against the design's, both kWh/m²·pa. */
const generationLabel = computed(() => {
  if (props.dfpPvIntensity === null || props.pvKwh === null) return '';
  return `${props.pvKwh.toFixed(2)} of ${props.dfpPvIntensity.toFixed(2)} kWh/m²`;
});

/**
 * Generation at or above the design is good news; below it is worth chasing —
 * soiling, shading, a string down, an inverter that has been off since a power
 * cut. Unlike the share, this cannot be moved by the building's demand, which is
 * why the colour hangs off it rather than off the percentage beside it.
 */
const generationColor = computed(() => {
  if (props.dfpPvIntensity === null || props.pvKwh === null) return DFP_SEVERITY_COLOR.unknown;
  return props.pvKwh >= props.dfpPvIntensity ? DFP_SEVERITY_COLOR.good : DFP_SEVERITY_COLOR.watch;
});

const generationTitle = computed(() =>
  'On-site generation measured over this window, annualised, against what the ' +
  'Design for Performance case assumed. Compare this rather than the percentages: ' +
  'the solar share falls when the building consumes more than its design, so an ' +
  'array meeting its target still reads below the design share.'
);

const chartData = computed(() => ({
  labels:   ['Grid electricity', 'PV generation'],
  datasets: [{
    data:            [gridKwh.value, props.pvKwh ?? 0],
    backgroundColor: ['#7F00FF', '#4caf50'],
    borderColor:     ['#7F00FF', '#4caf50'],
    borderWidth:     0,
    hoverOffset:     4
  }]
}));

const chartOptions = {
  responsive:          true,
  maintainAspectRatio: true,
  cutout:              '68%',
  plugins: {
    legend:  {display: false},
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

/* Stacked, not inline: the card is one of three in a row and the two references
   answer different questions, so running them together invites reading one as a
   restatement of the other. */
.donut-target {
  display:        flex;
  flex-direction: column;
  gap:            2px;
  font-size:      12px;
  color:          rgba(255, 255, 255, 0.4);
  text-align:     center;
}

.target-generation {
  cursor:               help;
  font-variant-numeric: tabular-nums;
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
  width:    200px;
  height:   200px;
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

.legend-dot--pv {
  background: #4caf50;
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
