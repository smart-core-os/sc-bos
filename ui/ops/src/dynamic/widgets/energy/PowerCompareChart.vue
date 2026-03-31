<template>
  <div class="root">
    <div class="chart__legend">
      <v-list bg-color="transparent" density="compact" class="py-0">
        <v-list-item v-for="(item, index) in legendItems"
                     :key="index"
                     :title="item.text"
                     @click="item.onClick(item.hidden)">
          <template #prepend>
            <v-list-item-action start>
              <v-checkbox-btn :model-value="!item.hidden" readonly :color="item.bgColor" density="compact"/>
            </v-list-item-action>
          </template>
        </v-list-item>
      </v-list>
    </div>
    <div class="chart__container">
      <div class="chart__parent">
        <pie :data="chartData" :options="chartOptions" :plugins="[vueLegendPlugin]"/>
      </div>
      <div class="chart__total">
        <span class="value">{{ totalStr }}</span><span class="units">{{ totalUnits }}</span>
      </div>
    </div>
  </div>
</template>

<script setup>
import {useVueLegendPlugin} from '@/components/charts/plugins.js';
import {defineChartOptions} from '@/components/charts/util.js';
import {
  useChartTotalDataset,
  usePullElectricDemandRecord,
  usePullElectricDemands
} from '@/dynamic/widgets/energy/electric.js';
import {ArcElement, Chart as ChartJS, Legend, Title, Tooltip} from 'chart.js';
import {computed, ref, toRef, watch} from 'vue';
import {Pie} from 'vue-chartjs';

ChartJS.register(Title, Tooltip, Legend, ArcElement);

const props = defineProps({
  totalSource: {
    type: String,
    default: undefined,
  },
  sources: {
    type: Array, // of String or {title: String, name: String}
    default: () => [],
  },
  metric: {
    type: String,
    default: 'realPower',
  },
  unit: {
    type: String,
    default: undefined,
  }
});

const totalUsage = usePullElectricDemandRecord(toRef(props, 'totalSource'), toRef(props, 'metric'));
const otherUsages = usePullElectricDemands(toRef(props, 'sources'), toRef(props, 'metric'));

const records = useChartTotalDataset(totalUsage, otherUsages);
const recordValues = computed(() => records.value.datasets[0].data);

const {legendItems, vueLegendPlugin} = useVueLegendPlugin()
// not computed to avoid infinite loop with legendItems updates triggering chartData updates
const visibleRecords = ref(recordValues.value);
watch(legendItems, (legendItems) => {
  if (!legendItems) {
    return;
  }
  const updated = legendItems.reduce((acc, item, i) => {
    if (!item.hidden) {
      acc.push(recordValues.value[i]);
    }
    return acc;
  }, []);
  const old = visibleRecords.value;
  if (updated.length !== old.length || updated.some((v, i) => v !== old[i])) {
    visibleRecords.value = updated;
  }
}, {deep: true});

const totalVisible = computed(() => visibleRecords.value.reduce((a, b) => a + b, 0));
const totalStr = computed(() => {
  return `${totalVisible.value.toLocaleString(undefined, {
    maximumFractionDigits: 1,
  })}`;
});
const totalUnits = computed(() => {
  if (props.unit) {
    return props.unit;
  }
  switch (props.metric) {
    case 'realPower':
      return 'kW';
    case 'current':
      return 'A';
    default:
      return '';
  }
});

const chartOptions = computed(() => {
  return defineChartOptions({
    cutout: '75%',
    circumference: 230,
    borderWidth: 0,
    spacing: 1,
    plugins: {
      legend: {
        display: false
      }
    }
  });
});
const chartData = computed(() => records.value);
</script>

<style scoped lang="scss">
.root {
  display: grid;
  grid-template-columns: repeat(3, 1fr);
}

.chart__legend {
  grid-row: 1 / span 1;
  grid-column: 1 / span 2;
}

.chart__container {
  grid-row: 1 / span 1;
  grid-column: 2 / span 2;
  min-height: 0;
  min-width: 0;
  max-height: 100%;
  display: grid;
  grid-template: minmax(0, 1fr) / minmax(0, 1fr);
  align-items: stretch;
  justify-self: end;
  padding-top: 10px;
  aspect-ratio: 1;

  > * {
    grid-column: 1 / span 1;
    grid-row: 1 / span 1;
  }

  // fix alignment issues caused by the donut chart not having an option to lay out as if it were a full circle.
  & {
    overflow: hidden;
  }
  .chart__parent {
    position: relative;
    left: 10%; // this should be ok as we have aspect-ratio 1
  }
}

.chart__total {
  align-self: center;
  justify-self: center;

  font-size: 2.2rem;
  letter-spacing: -.02em;
  line-height: 1;

  position: relative;

  .units {
    font-size: 50%;
    opacity: 0.8;
    font-weight: lighter;
    // force the value to be the sole source of layout
    position: absolute;
    top: 100%;
    right: 0;
  }
}
</style>