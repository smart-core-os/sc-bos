<template>
  <time :datetime="datetime">{{ str }}</time>
</template>

<script setup>
import {useNow} from '@/components/now.js';
import {DAY} from '@/util/date.js';
import {format, milliseconds} from 'date-fns';
import {computed} from 'vue';

const props = defineProps({
  date: {
    type: [String, Number, Date],
    default: ''
  },
  noRelative: Boolean
});
const relativeFormatter = new Intl.RelativeTimeFormat(undefined, {numeric: 'auto'});
const absoluteFormatter = new Intl.DateTimeFormat(undefined, {weekday: 'short', day: 'numeric', month: 'short'});

const dateObj = computed(() => {
  if (typeof props.date === 'number') return new Date(props.date);
  if (typeof props.date === 'string') return Date.parse(/** @type {string} */ props.date);
  return /** @type {Date} */ props.date;
});

const datetime = computed(() => {
  return format(dateObj.value, 'yyyy-MM-dd');
});

const days = milliseconds({days: 1});
const relativeDateThreshold = 7 * days;
const {now} = useNow(DAY);
const str = computed(() => {
  const dur = dateObj.value - now.value;
  if (!props.noRelative && Math.abs(dur) < relativeDateThreshold) {
    return relativeFormatter.format(Math.round(dur / days), 'days');
  }

  return `on ${absoluteFormatter.format(dateObj.value)}`;
});
</script>
