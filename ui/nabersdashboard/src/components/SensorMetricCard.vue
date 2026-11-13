<template>
  <div class="metric-row">
    <div class="metric-label">
      <!-- eslint-disable-next-line vue/no-v-html -->
      <span class="metric-title" v-html="title"/>
      <span class="metric-unit">{{ unit }}</span>
    </div>
    <div class="metric-gauges">
      <div v-for="g in gauges" :key="g.label" class="gauge-cell">
        <SensorGauge
            :value="g.value"
            :min="min"
            :max="max"
            :unit="unit"
            :label="g.label"
            :color="g.color"
            :size="100"/>
      </div>
    </div>
  </div>
</template>

<script setup>
import SensorGauge from './SensorGauge.vue';

defineProps({
  title:  {type: String, required: true},
  unit:   {type: String, required: true},
  gauges: {type: Array,  required: true},
  min:    {type: Number, required: true},
  max:    {type: Number, required: true}
});
</script>

<style scoped>
.metric-row {
  display: flex;
  align-items: center;
  padding: 0 24px;
  border-bottom: 1px solid var(--sc-navy);
}

.metric-label {
  width: 120px;
  flex-shrink: 0;
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.metric-title {
  font-size: 14px;
  font-weight: 600;
  line-height: 1.2;
}

.metric-unit {
  font-size: 11px;
  opacity: 0.45;
  font-weight: 400;
}

.metric-gauges {
  display: flex;
  flex: 1;
  justify-content: space-around;
  align-items: center;
}

.gauge-cell {
  flex: 1;
  display: flex;
  justify-content: center;
  align-items: center;
}
</style>
