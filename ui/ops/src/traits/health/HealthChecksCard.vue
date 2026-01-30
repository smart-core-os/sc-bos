<template>
  <v-card tile elevation="0" class="ma-0 pa-0">
    <v-card-title class="text-title-caps-large text-neutral-lighten-3">Health Checks</v-card-title>
    <health-check-card v-for="check in healthChecksList" :key="check.id" :model-value="check" class="check"/>
  </v-card>
</template>

<script setup>
import HealthCheckCard from '@/traits/health/HealthCheckCard.vue';
import {computed} from 'vue';

const props = defineProps({
  value: {
    type: Object,
    default: null
  },
  loading: {
    type: Boolean,
    default: false
  }
});

const healthChecksList = computed(() => {
  if (!props.value) return [];
  return Object.values(props.value);
});
</script>

<style scoped>
.check:not(:last-of-type) {
  margin-bottom: 16px;
}
</style>