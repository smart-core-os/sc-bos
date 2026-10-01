<template>
  <div class="stat-card">
    <div class="stat-label">After-hours equipment</div>
    <div class="status-row">
      <span class="status-dot" :style="{ background: statusColor }"/>
      <span class="status-text" :style="{ color: statusColor }">{{ statusText }}</span>
    </div>
    <div v-if="isAfterHours && kwh !== null" class="stat-sub">
      {{ kwh.toFixed(2) }} kWh since {{ endHourLabel }}
    </div>
    <div v-else-if="!isAfterHours" class="stat-sub">
      Operating hours end {{ endHourLabel }}
    </div>
    <div class="stat-desc">
      Equipment energy consumed after operating hours end. Standby loads indicate devices left on overnight.
    </div>
  </div>
</template>

<script setup>
import {computed} from 'vue';

const props = defineProps({
  kwh:        {type: Number, default: null},
  isAfterHours: {type: Boolean, default: false},
  operatingHoursEnd: {type: Number, default: 17}
});

const STANDBY_THRESHOLD_KWH = 0.5;

const endHourLabel = computed(() => `${String(props.operatingHoursEnd).padStart(2, '0')}:00`);

const statusColor = computed(() => {
  if (!props.isAfterHours) return '#4caf50';
  if (props.kwh === null)   return 'rgba(255,255,255,0.35)';
  return props.kwh >= STANDBY_THRESHOLD_KWH ? '#f7a12c' : '#4caf50';
});

const statusText = computed(() => {
  if (!props.isAfterHours)  return 'Within operating hours';
  if (props.kwh === null)   return 'No data';
  return props.kwh >= STANDBY_THRESHOLD_KWH ? 'Standby loads detected' : 'Equipment off';
});
</script>

<style scoped>
.stat-card {
  background:      rgba(255, 255, 255, 0.05);
  border:          1px solid rgba(255, 255, 255, 0.08);
  border-radius:   12px;
  padding:         24px 28px;
  display:         flex;
  flex-direction:  column;
  gap:             6px;
  flex:            1;
  justify-content: space-between;
}

.stat-label {
  font-size:      12px;
  font-weight:    500;
  color:          rgba(255, 255, 255, 0.45);
  text-transform: uppercase;
  letter-spacing: 0.08em;
}

.status-row {
  display:     flex;
  align-items: center;
  gap:         10px;
  margin-top:  8px;
}

.status-dot {
  width:         14px;
  height:        14px;
  border-radius: 50%;
  flex-shrink:   0;
}

.status-text {
  font-size:   28px;
  font-weight: 300;
}

.stat-sub {
  font-size: 14px;
  color:     rgba(255, 255, 255, 0.4);
}

.stat-desc {
  font-size:   13px;
  color:       rgba(255, 255, 255, 0.35);
  margin-top:  8px;
  line-height: 1.4;
}
</style>
