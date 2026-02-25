<template>
  <v-card :class="props.class" :style="props.style">
    <security-events-table v-bind="$attrs" :fixed-header="false"/>
  </v-card>
</template>

<script setup>
import SecurityEventsTable from '@/routes/ops/security-events/SecurityEventsTable.vue';

const props = defineProps({
  style: {type: [String, Object, Array], default: null},
  class: {type: [String, Object, Array], default: null},
})

defineOptions({inheritAttrs: false})
</script>

<style scoped>

</style>