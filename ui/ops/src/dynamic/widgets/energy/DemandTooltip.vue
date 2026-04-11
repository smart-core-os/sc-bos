<template>
  <v-menu :target="target" :model-value="visible"
          location="end" :offset="20" transition="slide-x-transition"
          content-class="no-pointer-events">
    <v-card>
      <v-card-title>{{ titleStr }}</v-card-title>
      <v-defaults-provider :defaults="{VListItem: {minHeight: '1.5em'}}">
        <v-list density="compact">
          <v-list-item v-for="(row, index) in demandRows" :key="index" :title="row.title">
            <template #prepend>
              <span v-if="row.isDashed" class="peak-swatch" :style="{borderColor: row.prependColor}"/>
              <v-avatar v-else :color="row.prependColor" size="1.5em"/>
            </template>
            <template #append>
              <span class="ml-4">{{ row.append }}</span>
            </template>
          </v-list-item>
          <v-list-item v-if="props.showTotal && totalDemandRow && demandRows.length > 1"
                       :title="totalDemandRow.title"
                       active>
            <template #prepend>
              <v-avatar icon="mdi-sigma" size="1.5em"/>
            </template>
            <template #append>
              <span class="ml-4">{{ totalDemandRow.append }}</span>
            </template>
          </v-list-item>
        </v-list>
      </v-defaults-provider>
    </v-card>
  </v-menu>
</template>

<script setup>
import {usageToString} from '@/traits/meter/meter.js';
import {format} from 'date-fns';
import {computed, toRef} from 'vue';

const props = defineProps({
  data: {
    type: Object, // of type TooltipData
    default: null
  },
  edges: {
    type: Array, // of Date
    required: true,
  },
  tickUnit: {
    type: String,
    default: 'hour'
  },
  unit: {
    type: String,
    default: undefined
  },
  showTotal: {
    type: Boolean,
    default: false,
  }
});
const data = computed(() => /** @type {TooltipData} */ props.data);
const edges = computed(() => /** @type {Date[]} */ props.edges);
const tickUnit = toRef(props, 'tickUnit');
const unit = toRef(props, 'unit');

const visible = computed(() => data.value?.opacity > 0);
const target = computed(() => {
  const tt = data.value;
  if (!tt) return [0, 0];
  return [tt.x, tt.y];
});
const titleStr = computed(() => {
  const tt = data.value;
  if (!tt || tt.dataPoints.length === 0) return '';

  // the tooltip title should match the tick label where possible.
  // For short timeUnits (minutes, hours) we explicitly show the range to make it more obvious.
  // For larger timeUnits this disambiguation isn't needed: Feb 10 or 2024 are clear enough
  const formatStr = tt.displayFormats[tickUnit.value];
  const index = tt.dataPoints[0].dataIndex;
  switch (tickUnit.value) {
    case 'minute':
    case 'hour':
      return `${format(edges.value[index], formatStr)}—${format(edges.value[index + 1], formatStr)}`
    default:
      return format(edges.value[index], formatStr);
  }
});

const demandData = computed(() => {
  const tt = data.value;
  if (!tt) return [];
  return tt.dataPoints.filter((dp) => !dp.dataset._inverted && !dp.dataset._isPeak);
})
const peakData = computed(() => {
  const tt = data.value;
  if (!tt) return [];
  return tt.dataPoints.filter((dp) => !dp.dataset._inverted && dp.dataset._isPeak);
})
const demandRows = computed(() => {
  const rows = (demandData.value ?? []).map((dp) => {
    return {
      title: dp.dataset.label,
      prependColor: dp.dataset.borderColor,
      append: usageToString(dp.parsed.y, unit.value),
    };
  });
  const peaks = (peakData.value ?? []).map((dp) => {
    return {
      title: dp.dataset.label,
      prependColor: dp.dataset.borderColor,
      append: usageToString(dp.parsed.y, unit.value),
      isDashed: true,
    };
  });
  return [...rows, ...peaks];
});
const totalDemandRow = computed(() => {
  const total = demandData.value?.reduce((acc, dp) => acc + dp.parsed.y, 0);
  return {
    title: 'Total Demand',
    append: usageToString(total, unit.value),
  };
});

</script>

<style scoped>
.peak-swatch {
  display: inline-block;
  width: 1.5em;
  height: 0;
  border-top: 2px dashed;
  margin-right: 0; /* aligns with v-avatar spacing */
  flex-shrink: 0;
  align-self: center;
}
</style>