<template>
  <status-alert v-if="props.streamError" icon="mdi-thermometer-low" :resource="props.streamError"/>

  <temperature-chip
      v-else-if="hasMeasured && !props.streamError"
      variant="text" size="30" layout="right"
      :measured="measured"/>
</template>
<script setup>
import StatusAlert from '@/components/StatusAlert.vue';
import {useTemperature} from '@/traits/temperature/temperature.js';
import TemperatureChip from '@/traits/temperature/TemperatureChip.vue';

const props = defineProps({
  value: {
    type: Object,
    default: () => {
    }
  },
  loading: {
    type: Boolean,
    default: false
  },
  showChangeDuration: {
    type: Number,
    default: 30 * 1000
  },
  streamError: {
    type: Object,
    default: null
  }
});

const {
  hasMeasured,
  measured
} = useTemperature(() => props.value);
</script>

<style scoped>
.temp-cell {
  display: flex;
  align-items: center;
}

.popup > .v-card__text {
  display: flex;
  justify-content: space-between;
  gap: 8px;
}
</style>

