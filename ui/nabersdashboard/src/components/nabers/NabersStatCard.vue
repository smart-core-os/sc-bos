<template>
  <div class="stat-card">
    <div class="stat-label">{{ title }}</div>
    <div class="stat-value" :style="{ color: valueColor }">
      <span v-if="value !== null">{{ value }}</span>
      <span v-else class="stat-dash">&mdash;</span>
      <span v-if="unit && value !== null" class="stat-unit">{{ unit }}</span>
    </div>
    <!-- With no value, why there is no value is more use than what the metric
         would have meant, so the reason takes the subtitle's place rather than
         stacking a third line into the card. -->
    <div v-if="value === null && emptyReason" class="stat-reason">{{ emptyReason }}</div>
    <div v-else-if="subtitle" class="stat-subtitle">{{ subtitle }}</div>
  </div>
</template>

<script setup>
defineProps({
  title:      {type: String, required: true},
  value:      {type: [String, Number], default: null},
  unit:       {type: String, default: ''},
  subtitle:   {type: String, default: ''},
  valueColor: {type: String, default: '#ffffff'},
  /**
   * Why there is no value, shown in place of the subtitle when `value` is null.
   * A bare em dash tells the reader the figure is absent but not whether that is
   * a config omission, a dead meter or simply too early in the rating period.
   */
  emptyReason: {type: String, default: ''}
});
</script>

<style scoped>
/* `flex-start`, not `space-between`. These cards stretch to whatever the tallest
   thing in the row needs, and with `space-between` the slack was split into two
   gaps — one of them between the label and its own figure, which pushed them
   apart and left the card looking mostly empty. Now the label and figure stay
   together at the top and all the slack collects above the bottom-anchored
   subtitle. */
.stat-card {
  background:      rgba(255, 255, 255, 0.05);
  border:          1px solid rgba(255, 255, 255, 0.08);
  border-radius:   12px;
  padding:         24px 28px;
  display:         flex;
  flex-direction:  column;
  gap:             6px;
  flex:            1;
  justify-content: flex-start;
}

.stat-label {
  font-size:      12px;
  font-weight:    500;
  color:          rgba(255, 255, 255, 0.45);
  text-transform: uppercase;
  letter-spacing: 0.08em;
}

.stat-value {
  font-size:   48px;
  font-weight: 300;
  line-height: 1.1;
  color:       #ffffff;
}

.stat-unit {
  font-size:   16px;
  font-weight: 400;
  color:       rgba(255, 255, 255, 0.45);
  margin-left: 6px;
}

.stat-dash {
  color: rgba(255, 255, 255, 0.3);
}

/* `margin-top: auto` keeps the subtitle on the card's floor, so three cards of
   differing text length still line their subtitles up with each other. */
.stat-subtitle {
  font-size:   13px;
  color:       rgba(255, 255, 255, 0.4);
  margin-top:  auto;
  padding-top: 8px;
  line-height: 1.4;
}

/* Amber, matching the attention colour used elsewhere on the dashboard, so a
   card that cannot show its figure reads as needing a look rather than as
   decoration. */
.stat-reason {
  font-size:   13px;
  color:       #f7a12c;
  margin-top:  auto;
  padding-top: 8px;
  line-height: 1.4;
}
</style>
