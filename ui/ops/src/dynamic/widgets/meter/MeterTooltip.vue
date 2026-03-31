<template>
  <v-menu :target="target" :model-value="visible"
          location="end" :offset="20" transition="slide-x-transition"
          content-class="no-pointer-events">
    <v-card>
      <v-card-title>{{ titleStr }}</v-card-title>
      <v-defaults-provider :defaults="{VListItem: {minHeight: '1.5em'}}">
        <template v-if="consumptionRows.length > 0">
          <v-card-subtitle>Consumption</v-card-subtitle>
          <v-list density="compact">
            <v-list-item v-for="(row, index) in consumptionRows" :key="index" :title="row.title">
              <template #prepend>
                <v-avatar :color="row.prependColor" size="1.5em"/>
              </template>
              <template #append>
                <span class="ml-4">{{ row.append }}</span>
              </template>
            </v-list-item>
            <v-list-item v-if="totalConsumptionRow && consumptionRows.length > 1 && !props.hideTotalConsumption"
                         :title="totalConsumptionRow.title"
                         active>
              <template #prepend>
                <v-avatar icon="mdi-sigma" size="1.5em"/>
              </template>
              <template #append>
                <span class="ml-4">{{ totalConsumptionRow.append }}</span>
              </template>
            </v-list-item>
          </v-list>
        </template>
        <template v-if="productionRows.length > 0">
          <v-card-subtitle>Production</v-card-subtitle>
          <v-list density="compact">
            <v-list-item v-for="(row, index) in productionRows" :key="index" :title="row.title">
              <template #prepend>
                <v-avatar :color="row.prependColor" size="1.5em"/>
              </template>
              <template #append>
                <span class="ml-4">{{ row.append }}</span>
              </template>
            </v-list-item>
            <v-list-item v-if="totalProductionRow && productionRows.length > 1"
                         :title="totalProductionRow.title"
                         active>
              <template #prepend>
                <v-avatar icon="mdi-sigma" size="1.5em"/>
              </template>
              <template #append>
                <span class="ml-4">{{ totalProductionRow.append }}</span>
              </template>
            </v-list-item>
          </v-list>
        </template>
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
  hideTotalConsumption: {
    type: Boolean,
    default: false
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

const consumptionData = computed(() => {
  const tt = data.value;
  if (!tt) return null;
  return tt.dataPoints.filter((dp) => !dp.dataset._inverted);
})
const productionData = computed(() => {
  const tt = data.value;
  if (!tt) return null;
  return tt.dataPoints.filter((dp) => dp.dataset._inverted);
})

const consumptionRows = computed(() => {
  return (consumptionData.value ?? []).map((dp) => {
    return {
      title: dp.dataset.label,
      prependColor: dp.dataset.borderColor,
      append: usageToString(dp.parsed.y, unit.value),
    };
  });
});
const productionRows = computed(() => {
  return (productionData.value ?? []).map((dp) => {
    return {
      title: dp.dataset.label,
      prependColor: dp.dataset.borderColor,
      append: usageToString(-dp.parsed.y, unit.value),
    };
  });
});

const totalConsumptionRow = computed(() => {
  const total = consumptionData.value?.reduce((acc, dp) => acc + dp.parsed.y, 0);
  return {
    title: 'Total Consumption',
    append: usageToString(total, unit.value),
  };
});
const totalProductionRow = computed(() => {
  const total = productionData.value?.reduce((acc, dp) => acc + dp.parsed.y, 0);
  return {
    title: 'Total Production',
    append: usageToString(-total, unit.value),
  };
});

</script>

<style scoped>

</style>