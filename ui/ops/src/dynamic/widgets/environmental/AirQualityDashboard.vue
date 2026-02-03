<template>
  <v-card class="d-flex flex-column">
    <v-toolbar v-if="props.title" color="transparent">
      <v-toolbar-title class="text-h4">{{ props.title }}</v-toolbar-title>
    </v-toolbar>
    <v-card-text class="pt-0">
      <v-table density="comfortable" class="bg-transparent">
        <thead>
          <tr>
            <th class="text-left">Floor</th>
            <th v-for="tower in props.towers" :key="tower.name" class="text-center">
              {{ tower.name }}
            </th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="(floor, floorIndex) in floorLabels" :key="floorIndex">
            <td class="text-left">{{ floor }}</td>
            <td v-for="tower in props.towers" :key="tower.name" class="text-center pa-1">
              <air-quality-cell
                  v-if="getSensorName(tower, floorIndex)"
                  :sensor-name="getSensorName(tower, floorIndex)"/>
              <span v-else class="text-disabled">-</span>
            </td>
          </tr>
        </tbody>
      </v-table>
    </v-card-text>
  </v-card>
</template>

<script setup>
import {computed, defineAsyncComponent} from 'vue';

const AirQualityCell = defineAsyncComponent(() => import('./AirQualityDashboardCell.vue'));

const props = defineProps({
  title: {
    type: String,
    default: ''
  },
  towers: {
    type: Array,
    required: true,
    validator: (value) => {
      return value.every(tower =>
          typeof tower.name === 'string' &&
          Array.isArray(tower.floors) &&
          tower.floors.every(floor =>
              typeof floor.label === 'string' &&
              typeof floor.sensorName === 'string'
          )
      );
    }
  }
});

// Compute a unified list of floor labels across all towers
const floorLabels = computed(() => {
  const labels = new Set();
  for (const tower of props.towers) {
    for (const floor of tower.floors) {
      labels.add(floor.label);
    }
  }
  return Array.from(labels);
});

// Get the sensor name for a given tower and floor index
const getSensorName = (tower, floorIndex) => {
  const floorLabel = floorLabels.value[floorIndex];
  const floor = tower.floors.find(f => f.label === floorLabel);
  return floor?.sensorName;
};
</script>

<style scoped>
.v-table {
  --v-table-header-height: 40px;
}
</style>
