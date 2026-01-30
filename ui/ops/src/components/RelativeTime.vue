<template>
  <time :datetime="datetime">{{ str }}</time>
</template>

<script setup>
import {DAY, HOUR, MINUTE, SECOND, useNow} from '@/components/now.js';
import {format} from 'date-fns';
import {computed} from 'vue';

const props = defineProps({
  time: {
    type: [String, Number, Date],
    default: ''
  },
  noRelative: Boolean,
  short: Boolean,
});

const relativeFormatter = computed(() => {
  const opts = {numeric: 'auto'};
  if (props.short) {
    opts.style = 'short';
  }
  return new Intl.RelativeTimeFormat(undefined, opts);
});
const absoluteFormatter = new Intl.DateTimeFormat(undefined, {weekday: 'short', day: 'numeric', month: 'short'});

const timeObj = computed(() => {
  if (typeof props.time === 'number' || typeof props.time === 'string') return new Date(props.time);
  return props.time;
});

const datetime = computed(() => {
  return format(timeObj.value, 'yyyy-MM-dd\'T\'HH:mm:ss');
});

const {now} = useNow(SECOND/4); // update frequently to avoid jumpy ui updates

const str = computed(() => {
  const dur = timeObj.value - now.value;
  const absDur = Math.abs(dur);

  if (!props.noRelative) {
    if (absDur < MINUTE) {
      return relativeFormatter.value.format(Math.round(dur / SECOND), 'second');
    }
    if (absDur < HOUR) {
      return relativeFormatter.value.format(Math.round(dur / MINUTE), 'minute');
    }
    if (absDur < DAY) {
      return relativeFormatter.value.format(Math.round(dur / HOUR), 'hour');
    }
    if (absDur < 7 * DAY) {
      return relativeFormatter.value.format(Math.round(dur / DAY), 'day');
    }
  }

  return absoluteFormatter.format(timeObj.value);
});
</script>

