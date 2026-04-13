<template>
  <div class="chart__container">
    <line-chart :data="chartData" :options="chartOptions" :plugins="[themeColorPlugin, vueLegendPlugin]"/>
    <chart-tooltip :data="tooltipData" :edges="edges" :tick-unit="tickUnit"
                   :format-value="(y) => (y != null ? new Intl.NumberFormat(undefined, {}).format(y) + (unit ? ' ' + unit : '') : '—')"
                   :filter="(dp) => dp.dataset.yAxisID !== 'y_occ'"/>
  </div>
</template>

<script setup>
import {useDateScale} from '@/components/charts/date.js';
import {useExternalTooltip, useThemeColorPlugin, useVueLegendPlugin} from '@/components/charts/plugins.js';
import ChartTooltip from '@/components/charts/ChartTooltip.vue';
import {defineChartOptions} from '@/components/charts/util.js';
import {timestampToDate} from '@/api/convpb.js';
import {listOccupancySensorHistory} from '@/api/sc/traits/occupancy.js';
import {metrics} from '@/traits/airQuality/airQuality.js';
import {useAirQualityHistoryMetrics} from '@/dynamic/widgets/environmental/airQuality.js';
import {shiftFnFromStr} from '@/dynamic/widgets/occupancy/baseline.js';
import {asyncWatch, useLocalProp} from '@/util/vue.js';
import binarySearch from 'binary-search';
import {Occupancy} from '@smart-core-os/sc-bos-ui-gen/proto/smartcore/bos/occupancysensor/v1/occupancy_sensor_pb';
import {usePullMetadata} from '@/traits/metadata/metadata.js';
import {usePullOccupancy} from '@/traits/occupancy/occupancy.js';
import {effectScope, reactive, watch as vWatch} from 'vue';
import {sentenceCase} from 'change-case';
import {Chart as ChartJS, Legend, LinearScale, LineElement, PointElement, TimeScale, Title, Tooltip} from 'chart.js';
import {startOfDay, startOfYear} from 'date-fns';
import {computed, toRef, toValue} from 'vue';
import {Line as LineChart} from 'vue-chartjs';
import * as vColors from 'vuetify/util/colors';
import 'chartjs-adapter-date-fns'; // imported for side effects

const datasetSourceName = Symbol('datasetSourceName');

ChartJS.register(Title, Tooltip, LineElement, LinearScale, PointElement, TimeScale, Legend);

const findIdx = (edges, at) => {
  let i = binarySearch(edges, at, (a, b) => a.getTime() - b.getTime());
  if (i < 0) i = ~i - 1;
  return i;
};

const props = defineProps({
  source: {
    type: [String, Array],
    default: null
  },
  metric: {
    type: String,
    default: 'score'
  },
  unit: {
    type: String,
    default: null, // default calculated based on metric
  },
  start: {
    type: [String, Number, Date],
    default: 'day', // 'month', 'day', etc. meaning 'start of <day>' or a Date-like object
  },
  end: {
    type: [String, Number, Date],
    default: 'day', // 'month', 'day', etc. meaning 'end of <day>' or a Date-like object
  },
  offset: {
    type: [String, Number],
    default: 0, // when start/End is 'month', 'day', etc. offset that value into the past, like 'last month'
  },
  // Optional: one or more occupancy sensor names. When set, each sensor contributes a shaded
  // step-line (0 = unoccupied, 1 = occupied) overlaid on a secondary hidden y-axis so users can
  // visually correlate CO₂ spikes with occupancy transitions.
  occupancy: {
    type: [String, Array],
    default: null,
  },
  // When true, overlays a dashed line per source showing the same metric for the prior period.
  showBaseline: {
    type: Boolean,
    default: false,
  },
  // 'day', 'week', 'month' — how far back to shift the baseline window.
  baselineShift: {
    type: String,
    default: 'week',
  },
});

const _start = useLocalProp(toRef(props, 'start'));
const _end = useLocalProp(toRef(props, 'end'));
const _offset = useLocalProp(toRef(props, 'offset'));

const metricInfo = computed(() => metrics[props.metric] || {});
const unit = computed(() => props.unit || metricInfo.value.unit || '');

const {edges, pastEdges, tickUnit, startDate, endDate} = useDateScale(_start, _end, _offset);

// Support both single source (string) and multiple sources (array)
const sources = computed(() => {
  if (Array.isArray(props.source)) {
    return props.source;
  } else if (props.source) {
    return [props.source];
  }
  return [];
});

const datasetNames = computed(() => {
  return chartData.value.datasets.map(item => {
    return item[datasetSourceName];
  });
});

// The same colour palette the themeColorPlugin assigns in dataset order
const paletteColors = [
  vColors.blue.base,
  vColors.green.base,
  vColors.orange.base,
  vColors.yellow.base,
  vColors.red.base,
];

// Current period
const devices = useAirQualityHistoryMetrics(sources, toRef(props, 'metric'), pastEdges);

// Prior period — edges shifted back; empty when showBaseline is off so no fetches are made
const shiftFn = computed(() => shiftFnFromStr(props.baselineShift));
const baselineEdges = computed(() =>
  props.showBaseline ? edges.value.map(shiftFn.value) : []
);
const baselineDevices = useAirQualityHistoryMetrics(
  computed(() => props.showBaseline ? sources.value : []),
  toRef(props, 'metric'),
  baselineEdges
);

// --- Occupancy overlay ---
// For each occupancy source, fetch state history and convert to 0/1 step values per edge.

const occupancySources = computed(() => {
  if (!props.occupancy) return [];
  return Array.isArray(props.occupancy) ? props.occupancy : [props.occupancy];
});

// occupancyEdges: includes all buckets so the shading extends until the end of the chart.
const occupancyEdges = computed(() => edges.value);

// occupancyDatasets: one dataset per occupancy sensor, each with 0/1 values per edge.
const occupancyStoreByName = reactive(/** @type {Record<string, {title: string, states: (0|1)[]}>} */ {});
const occupancyScopes = {};

vWatch(occupancySources, (names) => {
  const toStop = new Set(Object.keys(occupancyScopes));
  for (const name of names) {
    if (occupancyScopes[name]) { toStop.delete(name); continue; }
    const scope = effectScope();
    occupancyScopes[name] = scope;
    scope.run(() => {
      const {value: md} = usePullMetadata(name);
      const {value: liveOcc} = usePullOccupancy(name);
      const statesRef = reactive({values: []});

      asyncWatch([() => toValue(occupancyEdges)], async ([edges]) => {
        if (!edges || edges.length < 2) return;

        const states = Array(edges.length - 1).fill(null);

        try {
          const res = await listOccupancySensorHistory({
            name,
            period: {endTime: edges[0]},
            orderBy: 'record_time desc',
            pageSize: 1,
          }, {});
          if (res.occupancyRecordsList.length > 0) {
            states[0] = res.occupancyRecordsList[0].occupancy.state === Occupancy.State.OCCUPIED ? 1 : 0;
          }
        } catch { /* no pre-period record */ }

        const req = {
          name,
          period: {startTime: edges[0], endTime: edges[edges.length - 1]},
          pageSize: 500,
        };
        do {
          const res = await listOccupancySensorHistory(req, {});
          if (res.occupancyRecordsList.length === 0) break;
          for (const record of res.occupancyRecordsList) {
            const d = timestampToDate(record.recordTime);
            const idx = findIdx(edges, d);
            if (idx < 0 || idx >= edges.length - 1) continue;
            states[idx] = record.occupancy.state === Occupancy.State.OCCUPIED ? 1 : 0;
          }
          req.pageToken = res.nextPageToken;
        } while (req.pageToken);

        // carry forward
        if (states[0] === null) states[0] = 0;
        for (let i = 1; i < states.length; i++) {
          if (states[i] === null) states[i] = states[i - 1];
        }

        statesRef.values = states;
      }, {immediate: true});

      // Update current bucket state in real-time as events arrive
      vWatch(liveOcc, (occ) => {
        if (!occ) return;
        const edges = toValue(occupancyEdges);
        if (!edges || edges.length < 2) return;
        const d = timestampToDate(occ.stateChangeTime) || new Date();
        const idx = findIdx(edges, d);
        if (idx >= 0 && idx < statesRef.values.length) {
          const val = occ.state === Occupancy.State.OCCUPIED ? 1 : 0;
          if (statesRef.values[idx] !== val) {
            statesRef.values[idx] = val;
            // carry forward to any subsequent buckets already in statesRef
            for (let i = idx + 1; i < statesRef.values.length; i++) {
              statesRef.values[i] = val;
            }
          }
        }
      });

      vWatch([statesRef, md], () => {
        occupancyStoreByName[name] = {
          title: md.value?.appearance?.title ?? name,
          states: statesRef.values,
        };
      }, {immediate: true, deep: true});
    });
  }
  for (const name of toStop) {
    occupancyScopes[name].stop();
    delete occupancyScopes[name];
    delete occupancyStoreByName[name];
  }
}, {immediate: true});

const yAxisLabel = computed(() => {
  const s = metricInfo.value.labelText || sentenceCase(props.metric);
  if (unit.value) {
    return `${s} (${unit.value})`;
  }
  return s;
});

// Always use both plugins
const {legendItems, vueLegendPlugin} = useVueLegendPlugin();
const {themeColorPlugin} = useThemeColorPlugin();
const {external: tooltipExternal, data: tooltipData} = useExternalTooltip();
const hasOccupancy = computed(() => occupancySources.value.length > 0);
const occupancyDataMap = computed(() => {
  const map = {};
  const sourcesArr = sources.value;
  const occSourcesArr = occupancySources.value;
  const stores = occupancyStoreByName;

  sourcesArr.forEach((source, i) => {
    let occName = null;
    if (occSourcesArr.length === sourcesArr.length) {
      occName = occSourcesArr[i];
    } else if (occSourcesArr.length === 1) {
      occName = occSourcesArr[0];
    }
    if (!occName && occSourcesArr.length > 0) {
      occName = occSourcesArr.find(o => o.includes(source) || source.includes(o));
    }

    if (occName && stores[occName]) {
      map[source] = stores[occName].states;
    }
  });
  return map;
});

const chartOptions = computed(() => {
  return defineChartOptions({
    responsive: true,
    maintainAspectRatio: false,
    borderRadius: 3,
    interaction: {
      mode: 'index', // a single tooltip with all stacked datasets at the same x location in it
      intersect: false,
    },
    plugins: {
      legend: {
        display: false, // we use a custom legend plugin and vue for this
        labels: {
          filter: (item, data) => {
            const ds = data?.datasets?.[item.datasetIndex];
            return ds && ds.yAxisID !== 'y_occ';
          }
        }
      },
      tooltip: {
        enabled: false,
        external: tooltipExternal,
      }
    },
    scales: {
      y: {
        stacked: false,
        beginAtZero: true,
        title: {
          display: true,
          text: yAxisLabel.value,
          color: 'rgba(255, 255, 255, 0.7)',
        },
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
          callback(value) {
            return new Intl.NumberFormat(undefined, {}).format(Math.abs(value));
          },
          color: 'rgba(255, 255, 255, 0.9)',
          padding: 8
        },
      },
      // Hidden secondary axis for occupancy 0/1 overlay
      ...(hasOccupancy.value ? {
        y_occ: {
          display: false,
          min: 0,
          max: 1.3, // leave headroom so occupied line doesn't touch the top
          position: 'right',
        }
      } : {}),
      x: {
        type: 'time',
        min: startDate.value,
        max: endDate.value,
        stacked: false,
        border: {
          display: false
        },
        grid: {
          display: false,
        },
        ticks: {
          maxTicksLimit: 11,
          includeBounds: true,
          callback(value) {
            const unit = tickUnit.value;
            if (unit === 'month' && value === startOfYear(value).getTime()) return this.format(value, this.options.time.displayFormats['year']);
            if (unit === 'hour' && value === startOfDay(value).getTime()) return this.format(value, this.options.time.displayFormats['day']);
            return this.format(value);
          },
          color: 'rgba(255, 255, 255, 0.9)',
          padding: 8,
          maxRotation: 0
        },
        time: {
          unit: tickUnit.value,
          displayFormats: {
            hour: 'H:mm', // default: 4:00 AM
            day: 'd MMM', // default: Feb 10
            month: 'MMM', // default: Feb 2025 - we fix the ambiguity in ticks.callback
          }
        }
      }
    }
  });
});

const chartLabels = computed(() => edges.value.slice(0, -1));
const chartData = computed(() => {
  const datasets = [];

  const sourceList = sources.value;

  // Current period — themeColorPlugin assigns palette colours in dataset order
  for (const name of sourceList) {
    const device = devices[name];
    if (!device) continue;
    const label = toValue(device.title) || name;
    const deviceData = toValue(device.data);
    const d = deviceData.map(p => ({x: p.x, y: p.y}));
    const dates = deviceData.map(p => p.x);
    if (d.length > 0) {
      // Add a padding point at the end of the chart to ensure the line renders fully across the x axis
      const lastChartEdge = edges.value[edges.value.length - 1];
      d.push({x: lastChartEdge, y: d[d.length - 1].y});
      dates.push(lastChartEdge);
    }

    datasets.push({
      label,
      data: d,
      dates,
      [datasetSourceName]: name,
      order: 0,
      segment: {
        borderColor: (ctx) => {
          const ds = ctx.chart.data.datasets[ctx.datasetIndex];
          const color = ds.borderColor;
          const occStates = occupancyDataMap.value[name];
          if (!occStates) return color;
          const idx = ctx.p0DataIndex;
          const occupied = occStates[idx] === 1;
          if (occupied) return color;

          // themeColorPlugin sets borderColor as a hex string (e.g. #3f51b5)
          if (typeof color === 'string' && color.startsWith('#')) {
            return color + '40'; // 25% opacity
          }
          return color;
        },
        borderWidth: (ctx) => {
          const occStates = occupancyDataMap.value[name];
          if (!occStates) return undefined;
          const idx = ctx.p0DataIndex;
          const occupied = occStates[idx] === 1;
          return occupied ? 3 : 1.5;
        }
      }
    });
  }

  // Prior period — dashed lines in the same palette colour as their current counterpart.
  // backgroundColor: 'transparent' causes themeColorPlugin to skip these datasets, so
  // we pre-set borderColor to match the current dataset at the same index.
  if (props.showBaseline) {
    sourceList.forEach((name, i) => {
      const device = baselineDevices[name];
      if (!device) return;
      const deviceData = toValue(device.data);
      const d = deviceData.map((p, j) => ({x: chartLabels.value[j], y: p.y}));
      const dates = deviceData.map(p => p.x);
      if (d.length > 0) {
        // Add a padding point at the end of the chart to ensure the line renders fully across the x axis
        const lastChartEdge = edges.value[edges.value.length - 1];
        d.push({x: lastChartEdge, y: d[d.length - 1].y});
        dates.push(lastChartEdge);
      }
      datasets.push({
        label: `${toValue(device.title) || name} (prior)`,
        data: d,
        dates,
        [datasetSourceName]: name,
        borderColor: paletteColors[i % paletteColors.length],
        backgroundColor: 'transparent',
        isDashed: true,
        borderDash: [5, 5],
        borderWidth: 1.5,
        pointRadius: 0,
        pointHoverRadius: 3,
        tension: 0.3,
        order: 5,
      });
    });
  }

  // Occupancy overlay: one semi-transparent filled step-line per occupancy source.
  // Rendered on the hidden y_occ axis so it doesn't interfere with the main scale.
  Object.entries(occupancyStoreByName).forEach(([, occ]) => {
    const data = occ.states.map((s, i) => ({x: chartLabels.value[i], y: s}));
    if (occ.states.length > 0) {
      // Add a point at the end of the last bucket to close the stepped line
      const lastBucketEnd = edges.value[occ.states.length];
      data.push({x: lastBucketEnd, y: occ.states[occ.states.length - 1]});
    }
    datasets.push({
      type: 'line',
      label: `${occ.title} occupied`,
      data,
      yAxisID: 'y_occ',
      stepped: 'after',
      fill: true,
      borderWidth: 0,
      pointRadius: 0,
      pointHitRadius: 0, // don't trigger tooltips for the background shade
      backgroundColor: 'rgba(255, 255, 255, 0.08)',
      borderColor: 'transparent',
      // Pre-setting backgroundColor prevents themeColorPlugin from overriding it
      _pluginColor: false,
      order: 10, // draw behind other datasets
    });
  });

  return {labels: chartLabels.value, datasets};
});

// Expose chart reference for parent component
defineExpose({
  legendItems,
  datasetNames,
});
</script>

<style scoped>

</style>