<template>
  <v-card tile elevation="0" class="ma-0 pa-0">
    <v-card-title class="text-title-caps-large text-neutral-lighten-3">Health Checks</v-card-title>
    <health-check-card v-for="check in props.modelValue ?? []" :key="check.id" :model-value="check" class="check"/>
  </v-card>
</template>

<script setup>
import HealthCheckCard from '@/traits/health/HealthCheckCard.vue';

const props = defineProps({
  modelValue: {
    /** @type {import('vue').PropType<Array<import('@vanti-dev/sc-bos-ui-gen/proto/health_pb').HealthCheck.AsObject>>} */
    type: Array,
    default: null
  }
});
</script>

<style scoped>
.check:not(:last-of-type) {
  margin-bottom: 16px;
}
</style>