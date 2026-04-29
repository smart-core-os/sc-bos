<template>
  <div class="container">
    <time class="time" :datetime="timeDatetimeStr">{{ timeStr }}</time>
    <time class="date" :datetime="dateDatetimeStr">{{ dateStr }}</time>
  </div>
</template>

<script setup>
import {useNow} from '@/components/now.js';
import {SECOND} from '@/util/date.js';
import {computed} from 'vue';

const {now} = useNow(SECOND);
const timeStr = computed(() => now.value.toLocaleTimeString(undefined, {
  hour: 'numeric',
  minute: 'numeric'
}));
const dateStr = computed(() => now.value.toLocaleDateString(undefined, {
  weekday: 'long',
  day: 'numeric',
  month: 'long',
  year: 'numeric'
}));
const timeDatetimeStr = computed(() => now.value.toLocaleTimeString(undefined, {
  hour: '2-digit',
  minute: '2-digit',
  second: '2-digit'
}));
const dateDatetimeStr = computed(() => now.value.toLocaleDateString(undefined, {
  year: 'numeric',
  month: '2-digit',
  day: '2-digit'
}));
</script>

<style scoped>
.container {
  display: flex;
  flex-direction: column;
  justify-content: center;
  align-items: end;
}

.time {
  font-size: 160%;
  line-height: 1;
}
.date {
  font-weight: lighter;
}
</style>