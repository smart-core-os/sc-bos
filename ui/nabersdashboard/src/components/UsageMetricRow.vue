<template>
  <div class="metric-row">
    <div class="metric-label">
      <!-- eslint-disable-next-line vue/no-v-html -->
      <span class="metric-title" v-html="title"/>
      <span class="metric-unit">{{ unit }}</span>
    </div>
    <div class="usage-value">
      <div class="usage-primary">
        <span class="usage-number">{{ displayValue }}</span>
        <span class="usage-unit">{{ unit }}</span>
      </div>
      <div v-if="equivalent !== null" class="usage-equivalent">
        <span class="equiv-number">≈ {{ Math.round(equivalent) }}</span>
        <span class="equiv-label">{{ equivalentLabel }}</span>
      </div>
    </div>
  </div>
</template>

<script setup>
import {computed} from 'vue';

const props = defineProps({
  title:           {type: String, required: true},
  unit:            {type: String, required: true},
  value:           {type: Number, default: null},
  decimals:        {type: Number, default: 1},
  equivalent:      {type: Number, default: null},
  equivalentLabel: {type: String, default: ''}
});

const displayValue = computed(() => {
  if (props.value === null) return '—';
  return props.value.toFixed(props.decimals);
});
</script>

<style scoped>
.metric-row {
  display: flex;
  align-items: center;
  padding: 8px 24px;
  border-bottom: 1px solid var(--sc-navy);
}

.metric-label {
  width: 280px;
  flex-shrink: 0;
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.metric-title {
  font-size: 28px;
  font-weight: 600;
  line-height: 1.2;
}

.metric-unit {
  font-size: 18px;
  opacity: 0.45;
  font-weight: 400;
}

.usage-value {
  flex: 1;
  display: flex;
  align-items: center;
  justify-content: space-around;
}

.usage-primary,
.usage-equivalent {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 8px;
}

.usage-number {
  font-size: 96px;
  font-weight: 300;
  color: var(--sc-white);
  line-height: 1;
}

.usage-unit {
  font-size: 28px;
  opacity: 0.55;
}

.equiv-number {
  font-size: 96px;
  font-weight: 300;
  color: var(--sc-violet);
  line-height: 1;
}

.equiv-label {
  font-size: 28px;
  opacity: 0.55;
}
</style>
