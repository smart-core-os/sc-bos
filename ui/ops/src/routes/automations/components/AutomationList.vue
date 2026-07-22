<template>
  <div>
    <div class="d-flex justify-end px-4 pt-4">
      <cohort-points-export-btn/>
    </div>
    <services-list :name="ServiceNames.Automations" :type="props.type"/>
  </div>
</template>

<script setup>
import {ServiceNames} from '@/api/ui/services';
import CohortPointsExportBtn from '@/routes/automations/components/CohortPointsExportBtn.vue';
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
