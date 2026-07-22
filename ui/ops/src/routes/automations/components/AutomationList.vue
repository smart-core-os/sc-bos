<template>
  <services-list :name="ServiceNames.Automations" :type="props.type"/>
</template>

<script setup>
import {ServiceNames} from '@/api/ui/services';
import ServicesList from '@/routes/system/components/ServicesList.vue';

const props = defineProps({
  type: {
    type: String,
    default: 'all'
  }
});

</script>

<style lang="scss" scoped>
</style>
