<template>
  <v-card class="d-flex flex-column">
    <v-toolbar v-if="props.title" color="transparent">
      <v-toolbar-title class="text-h4">{{ props.title }}</v-toolbar-title>
    </v-toolbar>
    <v-card-text class="pt-0">
      <div class="d-flex flex-column ga-6">
        <div v-for="tower in props.towers" :key="tower.name">
          <h3 class="text-h6 mb-3">{{ tower.name }}</h3>
          <air-quality-tower-average
              :tower-name="tower.name"
              :sensor-names="tower.floors.map(f => f.sensorName)"
              class="mb-4"/>
          <div class="floor-grid">
            <v-card
                v-for="floor in tower.floors"
                :key="floor.label"
                variant="outlined"
                class="floor-card pa-3">
              <div class="text-subtitle-2 font-weight-bold mb-2">{{ floor.label }}</div>
              <air-quality-cell :sensor-name="floor.sensorName"/>
            </v-card>
          </div>
        </div>
      </div>
    </v-card-text>
  </v-card>
</template>

<script setup>
import {defineAsyncComponent} from 'vue';

const AirQualityCell = defineAsyncComponent(() => import('./AirQualityDashboardCell.vue'));
const AirQualityTowerAverage = defineAsyncComponent(() => import('./AirQualityTowerAverage.vue'));

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
</script>

<style scoped>
.floor-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(200px, 1fr));
  gap: 12px;
}

.floor-card {
  min-height: 80px;
}
</style>
