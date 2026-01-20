<template>
  <allocation-chip
      variant="text"
      size="30"
      layout="right"
      :allocation-total="at"
      :unallocation-total="ut"/>
</template>

<script setup>
import {useAllocation} from '@/traits/allocation/allocation.js';
import allocationChip from '@/traits/allocation/AllocationChip.vue';
import {computed} from 'vue';

const props = defineProps({
  value: {
    type: Object,
    default: () => ({})
  },
  loading: {
    type: Boolean,
    default: false
  },
  streamError: {
    type: Object,
    default: null
  }
});

const {
  allocationTotal,
  unallocationTotal
} = useAllocation(() => props.value);

const at = computed(() => allocationTotal.value || '');
const ut = computed(() => unallocationTotal.value || '');
</script>

<style scoped>
</style>

