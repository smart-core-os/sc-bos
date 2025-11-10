<template>
  <span class="root">
    <with-enter-leave
        v-if="hasCell('EnterLeaveEvent')"
        v-slot="{ resource }"
        :name="props.item.name"
        :paused="props.paused">
      <enter-leave-event-cell
          v-if="resource?.value?.enterTotal || resource?.value?.leaveTotal"
          v-bind="resource"/>
    </with-enter-leave>
    <with-electric-demand
        v-if="hasCell('ElectricDemand')"
        v-slot="{ resource }"
        :name="props.item.name"
        :paused="props.paused">
      <electric-demand-cell v-bind="resource"/>
    </with-electric-demand>
    <with-energy-storage
        v-if="hasCell('EnergyStorage')"
        v-slot="{ resource }"
        :name="props.item.name"
        :paused="props.paused">
      <energy-storage-cell v-bind="resource"/>
    </with-energy-storage>
    <with-meter v-if="hasCell('Meter')" v-slot="{ resource, info }" :name="props.item.name" :paused="props.paused">
      <meter-cell
          v-bind="resource"
          :info="info?.response"/>
    </with-meter>
    <with-air-temperature
        v-if="hasCell('AirTemperature')"
        v-slot="{ resource }"
        :name="props.item.name"
        :paused="props.paused">
      <air-temperature-cell v-bind="resource"/>
    </with-air-temperature>

    <light-cell v-if="hasCell('Light')" :name="props.item.name" :paused="props.paused"/>

    <with-occupancy v-if="hasCell('Occupancy')" v-slot="{ resource }" :name="props.item.name" :paused="props.paused">
      <occupancy-cell v-bind="resource"/>
    </with-occupancy>

    <!-- If door has no access data reading and has OpenClose reading -->
    <with-open-close
        v-if="hasCell('OpenClose') && !hasCell('AccessAttempt')"
        v-slot="{ resource }"
        :name="props.item.name"
        :paused="props.paused">
      <open-close-cell v-bind="resource"/>
    </with-open-close>

    <!-- If door has access data reading and has no OpenClose reading -->
    <with-access
        v-if="hasCell('AccessAttempt') && !hasCell('OpenClose')"
        v-slot="{ resource }"
        :name="props.item.name"
        :paused="props.paused">
      <access-attempt-cell v-bind="resource"/>
    </with-access>

    <!-- If door has access data reading and has OpenClose reading -->
    <with-access
        v-if="hasCell('AccessAttempt') && hasCell('OpenClose')"
        v-slot="{ resource: accessResource }"
        :name="props.item.name"
        :paused="props.paused">
      <with-open-close v-slot="{ resource: openCloseResource }" :name="props.item.name" :paused="props.paused">
        <access-attempt-cell
            v-bind="accessResource"
            :open-close-percentage="openCloseResource"
            :stream-error="accessResource.streamError || openCloseResource.streamError"/>
      </with-open-close>
    </with-access>
    <!-- End -->

    <with-emergency v-if="hasCell('Emergency')" v-slot="{ resource }" :name="props.item.name" :paused="props.paused">
      <emergency-cell v-bind="resource"/>
    </with-emergency>

    <with-status v-if="hasCell('StatusLog')" v-slot="{ resource }" :name="props.item.name" :paused="props.paused">
      <status-log-cell v-bind="resource"/>
    </with-status>

    <health-checks-cell v-if="healthExperiment" :model-value="props.item.healthChecksList"/>
  </span>
</template>

<script setup>
import {useExperiment} from '@/composables/experiments.js';
import AccessAttemptCell from '@/traits/access/AccessAttemptCell.vue';
import WithAccess from '@/traits/access/WithAccess.vue';
import AirTemperatureCell from '@/traits/airTemperature/AirTemperatureCell.vue';
import WithAirTemperature from '@/traits/airTemperature/WithAirTemperature.vue';
import ElectricDemandCell from '@/traits/electricDemand/ElectricDemandCell.vue';
import WithElectricDemand from '@/traits/electricDemand/WithElectricDemand.vue';
import EmergencyCell from '@/traits/emergency/EmergencyCell.vue';
import WithEmergency from '@/traits/emergency/WithEmergency.vue';
import EnergyStorageCell from '@/traits/energyStorage/EnergyStorageCell.vue';
import WithEnergyStorage from '@/traits/energyStorage/WithEnergyStorage.vue';
import EnterLeaveEventCell from '@/traits/enterLeave/EnterLeaveEventCell.vue';
import WithEnterLeave from '@/traits/enterLeave/WithEnterLeave.vue';
import HealthChecksCell from '@/traits/health/HealthChecksCell.vue';
import LightCell from '@/traits/light/LightCell.vue';
import MeterCell from '@/traits/meter/MeterCell.vue';
import WithMeter from '@/traits/meter/WithMeter.vue';
import OccupancyCell from '@/traits/occupancy/OccupancyCell.vue';
import WithOccupancy from '@/traits/occupancy/WithOccupancy.vue';
import OpenCloseCell from '@/traits/openClose/OpenCloseCell.vue';
import WithOpenClose from '@/traits/openClose/WithOpenClose.vue';
import StatusLogCell from '@/traits/status/StatusLogCell.vue';
import WithStatus from '@/traits/status/WithStatus.vue';
import {hasTrait} from '@/util/devices';
import {computed} from 'vue';

const props = defineProps({
  paused: {
    type: Boolean,
    default: false
  },
  item: {
    type: Object,
    default: () => {
    }
  }
});

const healthExperiment = useExperiment('health');

const visibleCells = computed(() => {
  const cells = {};
  if (hasTrait(props.item, 'smartcore.traits.OccupancySensor')) {
    cells['Occupancy'] = true;
  }
  if (hasTrait(props.item, 'smartcore.traits.AirTemperature')) {
    cells['AirTemperature'] = true;
  }
  if (hasTrait(props.item, 'smartcore.traits.Light')) {
    cells['Light'] = true;
  }
  if (hasTrait(props.item, 'smartcore.traits.Electric')) {
    cells['ElectricDemand'] = true;
  }
  if (hasTrait(props.item, 'smartcore.traits.EnergyStorage')) {
    cells['EnergyStorage'] = true;
  }
  if (hasTrait(props.item, 'smartcore.traits.EnterLeaveSensor')) {
    cells['EnterLeaveEvent'] = true;
  }
  if (hasTrait(props.item, 'smartcore.traits.Emergency')) {
    cells['Emergency'] = true;
  }
  if (hasTrait(props.item, 'smartcore.traits.OpenClose')) {
    cells['OpenClose'] = true;
  }
  if (hasTrait(props.item, 'smartcore.bos.Status')) {
    cells['StatusLog'] = true;
  }
  if (hasTrait(props.item, 'smartcore.bos.Meter')) {
    cells['Meter'] = true;
  }
  if (hasTrait(props.item, 'smartcore.bos.Access')) {
    cells['AccessAttempt'] = true;
  }
  return cells;
});

/**
 * @param {string} name
 * @return {boolean}
 */
function hasCell(name) {
  return Boolean(visibleCells.value[name]);
}
</script>

<style scoped>
.root {
  display: flex;
  flex-direction: row;
  align-items: center;
  justify-content: flex-end;
  gap: 1em;
}
</style>
