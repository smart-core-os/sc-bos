<template>
  <div class="glance-widget mt-2" @pointerup="emits('adminClick')">
    <div class="timedate">
      <h1 class="time">{{ timeStr }}</h1>
      <span class="text-h5">{{ dateStr }}</span>
      <span v-if="zoneName" class="text-h6 zone-name d-block">{{ zoneName }}</span>
    </div>
  </div>
</template>

<script setup>
import {SECOND, useNow} from '@/components/now';
import {useConfigStore} from '@/stores/config';
import {storeToRefs} from 'pinia';
import {computed} from 'vue';

const {now} = useNow(SECOND);
const timeStr = computed(() => now.value.toLocaleTimeString('en-GB', {
  hour: 'numeric',
  minute: 'numeric'
}));
const dateStr = computed(() => now.value.toLocaleDateString('en-GB', {
  weekday: 'long',
  day: 'numeric',
  month: 'long',
  year: 'numeric'
}));

const {zoneName} = storeToRefs(useConfigStore());

const emits = defineEmits(['adminClick']);
</script>

<style scoped>
.glance-widget {
  cursor: pointer;
  user-select: none;
}

.time {
  color: transparent;
  font-size: 4.5em;
  line-height: 1em;
  -webkit-text-stroke: 3px #fff;
  -webkit-font-smoothing: subpixel-antialiased;
  letter-spacing: 0.02em;
  font-weight: 900;
}
</style>
