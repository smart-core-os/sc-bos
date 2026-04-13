<template>
  <v-menu :target="target" :model-value="visible"
          location="end" :offset="20" transition="slide-x-transition"
          content-class="no-pointer-events">
    <v-card v-if="data">
      <v-card-title>{{ titleStr }}</v-card-title>
      <v-defaults-provider :defaults="{VListItem: {minHeight: '1.5em'}}">
        <v-list density="compact">
          <v-list-item v-for="(row, index) in rows" :key="index" :title="row.title">
            <template #prepend>
              <span v-if="row.isDashed" class="dashed-swatch" :style="{borderColor: row.color}"/>
              <v-avatar v-else :color="row.color" size="1.5em"/>
            </template>
            <template #append>
              <span class="ml-4">{{ row.valueStr }}</span>
            </template>
          </v-list-item>
          <v-list-item v-if="props.showTotal && totalRow && rows.length > 1"
                       :title="totalRow.title"
                       active>
            <template #prepend>
              <v-avatar icon="mdi-sigma" size="1.5em"/>
            </template>
            <template #append>
              <span class="ml-4">{{ totalRow.valueStr }}</span>
            </template>
          </v-list-item>
        </v-list>
      </v-defaults-provider>
    </v-card>
  </v-menu>
</template>

<script setup>
import {format} from 'date-fns';
import {computed, toRef} from 'vue';

const props = defineProps({
  data: {
    type: Object, // of type TooltipData
    default: null
  },
  edges: {
    type: Array, // of Date
    default: () => []
  },
  tickUnit: {
    type: String,
    default: 'hour'
  },
  showTotal: {
    type: Boolean,
    default: false,
  },
  totalTitle: {
    type: String,
    default: 'Total'
  },
  // Function to format the y-value: (y, dataset) => string
  formatValue: {
    type: Function,
    default: (y) => new Intl.NumberFormat(undefined, {}).format(y)
  },
  // Optional filter for data points
  filter: {
    type: Function,
    default: (dp) => !dp.dataset._inverted
  }
});

const data = computed(() => props.data);
const edges = computed(() => props.edges);
const tickUnit = toRef(props, 'tickUnit');

const visible = computed(() => data.value?.opacity > 0);
const target = computed(() => {
  const tt = data.value;
  if (!tt) return [0, 0];
  return [tt.x, tt.y];
});

const titleStr = computed(() => {
  const tt = data.value;
  if (!tt || tt.dataPoints.length === 0) return '';

  const index = tt.dataPoints[0].dataIndex;

  if (edges.value && edges.value[index]) {
    const formatStr = tt.displayFormats?.[tickUnit.value] || 'HH:mm';
    switch (tickUnit.value) {
      case 'minute':
      case 'hour':
        if (edges.value[index + 1]) {
          return `${format(edges.value[index], formatStr)}—${format(edges.value[index + 1], formatStr)}`;
        }
        return format(edges.value[index], formatStr);
      default:
        return format(edges.value[index], formatStr);
    }
  }

  // Fallback to label for categorical charts
  return tt.dataPoints[0].label;
});

const rows = computed(() => {
  const tt = data.value;
  if (!tt) return [];

  return tt.dataPoints
    .filter(props.filter)
    .map(dp => {
      const colorSource = dp.dataset.borderColor;
      const color = Array.isArray(colorSource) ? colorSource[dp.dataIndex] : colorSource;
      return {
        title: dp.dataset.label,
        color: color,
        valueStr: props.formatValue(dp.parsed.y, dp.dataset),
        isDashed: (dp.dataset.borderDash?.length ?? 0) > 0 || dp.dataset._isPeak,
      };
    });
});

const totalRow = computed(() => {
  const tt = data.value;
  if (!tt) return null;

  const filteredPoints = tt.dataPoints.filter(dp => props.filter(dp) && !dp.dataset._isPeak);
  if (filteredPoints.length === 0) return null;

  const total = filteredPoints.reduce((acc, dp) => acc + (dp.parsed.y || 0), 0);

  return {
    title: props.totalTitle,
    valueStr: props.formatValue(total, null),
  };
});
</script>

<style scoped>
.dashed-swatch {
  display: inline-block;
  width: 1.5em;
  height: 0;
  border-top: 2px dashed;
  margin-right: 0;
  flex-shrink: 0;
  align-self: center;
}
</style>
