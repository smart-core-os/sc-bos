<template>
  <div class="breakdown-wrapper">
    <div class="breakdown-header">
      <span class="breakdown-title">Base building breakdown</span>
      <div class="header-right">
        <span class="breakdown-unit">kWh/m² NIA/yr</span>
        <!--
          A way out to the fuller energy view, rather than more detail crammed
          onto a screen that is already full. Absent unless a site configures it,
          and `target`/`rel` because it leaves the app: this is a fullscreen
          dashboard with no in-app way back, so replacing it in place would strand
          whoever followed the link.
        -->
        <v-btn
            v-if="opsHref"
            :href="opsHref"
            target="_blank"
            rel="noopener noreferrer"
            size="x-small"
            variant="outlined"
            color="rgba(255,255,255,0.5)"
            append-icon="mdi-open-in-new">
          {{ opsLinkLabel }}
        </v-btn>
        <v-btn
            size="x-small"
            variant="outlined"
            color="rgba(255,255,255,0.5)"
            :disabled="!hasData"
            @click="exportCsv">
          Export CSV
        </v-btn>
      </div>
    </div>
    <div v-if="hasData" class="breakdown-chart">
      <Bar :data="chartData" :options="chartOptions"/>
    </div>
    <div v-else class="breakdown-empty">Waiting for meter data...</div>
    <div class="breakdown-desc">
      Annualised base building energy intensity per end-use vs {{ targetLabel }}.
      <template v-if="hasData">Measured over {{ measuredOver.label }}.</template>
      <span v-if="hasEstimated" class="estimated-note">Amber-outlined bars include projected meter data.</span>
    </div>
  </div>
</template>

<script setup>
import {computed} from 'vue';
import {Bar} from 'vue-chartjs';
import {Chart as ChartJS, CategoryScale, LinearScale, BarElement, Title, Tooltip, Legend} from 'chart.js';
import {NABERS_MODEL_VERSION} from '@/util/nabersRating.js';
import {downloadCsv} from '@/util/csv.js';
import {safeHttpUrl} from '@/util/externalLink.js';
import {measuredWindow} from '@/util/measuredWindow.js';

ChartJS.register(CategoryScale, LinearScale, BarElement, Title, Tooltip, Legend);

/**
 * Past this many end uses, vertical bars become unreadable — Chart.js rotates
 * or silently drops tick labels, and two datasets across eleven groups leaves
 * a few pixels per bar — so the chart flips to horizontal.
 */
const HORIZONTAL_THRESHOLD = 8;

const props = defineProps({
  categoryIntensities: {type: Object, default: () => ({})},
  dfpTargets:          {type: Object, default: () => ({})},
  categories:          {type: Array,  default: () => []},
  categoryLabels:      {type: Object, default: () => ({})},
  targetLabel:         {type: String, default: 'DfP Stage 4 Target'},
  /** `{[category]: kWh}` estimated over the period, keyed as `categories`. */
  categoryEstimatedKwh: {type: Object, default: () => ({})},
  /**
   * The measured kWh behind each intensity, and the area and elapsed days that
   * turned it into one. Not used to draw anything — the export carries them so a
   * reader can reproduce the chart's figures rather than take them on trust.
   */
  categoryMeasuredKwh: {type: Object, default: () => ({})},
  nia:                 {type: Number, default: null},
  elapsedDays:         {type: Number, default: null},
  /**
   * Which window the intensities annualise: `period` for the rating period to
   * date, `months` for the measured months behind it. The store picks whichever
   * the rating itself uses, so the chart and the gauge cannot disagree.
   *
   * @type {'months'|'period'|null}
   */
  intensityBasis:      {type: String, default: 'period'},
  monthsOfData:        {type: Number, default: 0},
  trailingDaysCovered: {type: Number, default: 0},
  /**
   * Where to send a reader who wants the detailed energy view — an ops UI page,
   * typically. Empty by default: a link is only worth offering where one has been
   * configured, and a wrong destination is worse than none.
   */
  opsUrl:              {type: String, default: ''},
  opsLinkLabel:        {type: String, default: 'Detailed energy view'}
});

/**
 * The link target, or null when unconfigured or unusable.
 *
 * Resolved against the current document so a deployment serving the ops UI from
 * this same origin can configure a bare path.
 */
const opsHref = computed(() => safeHttpUrl(props.opsUrl, window.location.href));

/**
 * The window the intensities cover, and its length.
 *
 * Not named `window`: that shadows the global inside an SFC.
 */
const measuredOver = computed(() => measuredWindow(props));

const annualisationFactor = computed(() =>
  measuredOver.value.days ? 365 / measuredOver.value.days : null
);

const hasData = computed(() => props.categories.some(c => props.categoryIntensities[c] !== null));

const horizontal = computed(() => props.categories.length > HORIZONTAL_THRESHOLD);

/**
 * @param {string} cat
 * @return {boolean}
 */
const isEstimated = (cat) => (props.categoryEstimatedKwh[cat] ?? 0) > 0;

const hasEstimated = computed(() => props.categories.some(isEstimated));

/**
 * Export what the chart draws.
 *
 * The bars are annualised intensities, which is the right shape for comparing
 * against a design target but hides how they were derived — so the export carries
 * the measured kWh, the rated area and the annualisation alongside, and states the
 * arithmetic in the preamble. Someone checking a figure can then follow it from
 * the meter reading to the bar without opening the code.
 */
function exportCsv() {
  const generatedAt = new Date();
  const factor = annualisationFactor.value;

  const preamble = [
    ['NABERS UK base building — energy breakdown by end use'],
    ['Generated', generatedAt.toISOString()],
    ['Model version', NABERS_MODEL_VERSION],
    ['Rated area (m² NIA)', props.nia ?? ''],
    ['Measured over', measuredOver.value.label],
    ['Days in window', measuredOver.value.days ?? ''],
    ['Annualisation factor', factor === null ? '' : factor.toFixed(4)],
    ['Intensity', `Measured kWh ÷ rated area × annualisation factor, over ${measuredOver.value.label}. ` +
      'The window is whichever the rating itself uses: complete months where twelve exist, else ' +
      'the period to date, else the measured months behind it. Annualising ignores seasonality.'],
    ['Target', 'The modelled figure from config for comparison. Blank where no target was ' +
      'transcribed for that end use — deliberately absent rather than estimated.'],
    ['Estimated kWh', 'Energy within the measured figure that was projected forward past the last ' +
      'reading of an unreachable meter. These are the amber-outlined bars on screen.'],
    ['Blank intensity', 'The end use could not be read — one unreadable meter makes its whole ' +
      'end use unknown. Not a zero.'],
    []
  ];

  const header = [
    'End use', 'Intensity (kWh/m²/yr)', `${props.targetLabel} (kWh/m²/yr)`,
    'Variance vs target (kWh/m²/yr)', 'Variance vs target (%)',
    'Measured (kWh)', 'Estimated (kWh)', 'Estimated (%)', 'Includes estimated data'
  ];

  const body = props.categories.map(cat => {
    const intensity = props.categoryIntensities[cat] ?? null;
    const target = props.dfpTargets[cat] ?? null;
    const kwh = props.categoryMeasuredKwh[cat] ?? null;
    const estimated = props.categoryEstimatedKwh[cat] ?? 0;
    const variance = (intensity !== null && target !== null) ? intensity - target : null;
    return [
      props.categoryLabels[cat] ?? cat,
      intensity === null ? '' : intensity.toFixed(2),
      target === null ? '' : target.toFixed(2),
      variance === null ? '' : variance.toFixed(2),
      (variance === null || !target) ? '' : ((variance / target) * 100).toFixed(1),
      kwh === null ? '' : Math.round(kwh),
      estimated > 0 ? Math.round(estimated) : '',
      (estimated > 0 && kwh) ? ((estimated / kwh) * 100).toFixed(1) : '',
      isEstimated(cat) ? 'yes' : ''
    ];
  });

  // A total only over the end uses that could be read, so it never presents a
  // partial sum as if it were the building's whole demand.
  const readable = props.categories.filter(c => (props.categoryIntensities[c] ?? null) !== null);
  const total = [
    `Total (${readable.length} of ${props.categories.length} end uses)`,
    readable.reduce((a, c) => a + props.categoryIntensities[c], 0).toFixed(2),
    '', '', '',
    Math.round(readable.reduce((a, c) => a + (props.categoryMeasuredKwh[c] ?? 0), 0)),
    Math.round(readable.reduce((a, c) => a + (props.categoryEstimatedKwh[c] ?? 0), 0)),
    '', ''
  ];

  downloadCsv([...preamble, header, ...body, total],
    `nabers-base-building-breakdown-${generatedAt.toISOString().slice(0, 10)}.csv`);
}

const chartData = computed(() => ({
  labels: props.categories.map(c => props.categoryLabels[c] ?? c),
  datasets: [
    {
      label:           'Actual (annualised)',
      data:            props.categories.map(c => props.categoryIntensities[c]),
      backgroundColor: '#7F00FF',
      // An amber outline on the bars that rest on a projected reading. A
      // missing bar and a genuine zero already look alike here; an estimated bar
      // must not silently join them.
      borderColor:     props.categories.map(c => (isEstimated(c) ? '#f7a12c' : 'transparent')),
      borderWidth:     props.categories.map(c => (isEstimated(c) ? 2 : 0)),
      borderRadius:    4
    },
    {
      label:           props.targetLabel,
      // `?? null`, not `|| null`: a genuine 0 target is a real value, not absence.
      data:            props.categories.map(c => props.dfpTargets[c] ?? null),
      backgroundColor: 'rgba(127,0,255,0.18)',
      borderColor:     '#7F00FF',
      borderWidth:     2,
      borderRadius:    4
    }
  ]
}));

const chartOptions = computed(() => {
  const categoryAxis = {
    ticks:    {color: 'rgba(255,255,255,0.7)', font: {family: 'Poppins', size: 12}, autoSkip: false},
    grid:     {color: 'rgba(255,255,255,0.06)'}
  };
  const valueAxis = {
    ticks:       {color: 'rgba(255,255,255,0.7)', font: {family: 'Poppins', size: 11}},
    grid:        {color: 'rgba(255,255,255,0.06)'},
    beginAtZero: true,
    title: {
      display: true,
      text:    'kWh/m² NIA/yr',
      color:   'rgba(255,255,255,0.4)',
      font:    {family: 'Poppins', size: 11}
    }
  };
  return {
    indexAxis:           horizontal.value ? 'y' : 'x',
    responsive:          true,
    maintainAspectRatio: false,
    plugins: {
      legend: {
        labels: {color: 'rgba(255,255,255,0.7)', font: {family: 'Poppins', size: 12}}
      },
      tooltip: {
        callbacks: {
          // The value sits on whichever axis is not the category axis.
          label: ctx => {
            const v = horizontal.value ? ctx.parsed.x : ctx.parsed.y;
            return v !== null && v !== undefined
              ? ` ${ctx.dataset.label}: ${v.toFixed(1)} kWh/m²`
              : null;
          }
        }
      }
    },
    scales: horizontal.value
      ? {x: valueAxis, y: categoryAxis}
      : {x: categoryAxis, y: valueAxis}
  };
});
</script>

<style scoped>
.breakdown-wrapper {
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

.breakdown-header {
  display:         flex;
  align-items:     baseline;
  justify-content: space-between;
  gap:             12px;
  flex-shrink:     0;
}

.header-right {
  display:     flex;
  align-items: baseline;
  gap:         12px;
}

.breakdown-title {
  font-size:      12px;
  font-weight:    500;
  color:          rgba(255, 255, 255, 0.45);
  text-transform: uppercase;
  letter-spacing: 0.08em;
}

.breakdown-unit {
  font-size: 12px;
  color:     rgba(255, 255, 255, 0.25);
}

.breakdown-chart {
  flex:       1;
  min-height: 0;
}

.breakdown-empty {
  flex:        1;
  font-size:   14px;
  color:       rgba(255, 255, 255, 0.3);
  text-align:  center;
  padding:     32px 0;
  font-style:  italic;
}

.breakdown-desc {
  font-size:   13px;
  color:       rgba(255, 255, 255, 0.35);
  line-height: 1.4;
  flex-shrink: 0;
}

.estimated-note {
  color: rgba(247, 161, 44, 0.7);
}
</style>
