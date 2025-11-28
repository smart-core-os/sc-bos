<template>
  <span>
    <span>{{ props.abnormalCount }}</span>
    <span> / </span>
    <span>{{ props.totalCount }}</span>
    <v-progress-circular
        class="ml-2"
        :model-value="progressValue"
        bg-color="error"
        color="white"
        :size="18"
        :width="5"/>
  </span>
</template>

<script setup>
import {computed} from 'vue';

const props = defineProps({
  abnormalCount: {
    type: Number,
    default: 0
  },
  totalCount: {
    type: Number,
    default: 0
  }
});

const progressValue = computed(() => {
  if (props.totalCount === 0) return 100;
  return (props.totalCount - props.abnormalCount) / props.totalCount * 100;
});
</script>

<style scoped>

</style>