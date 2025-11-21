<template>
  <span>
    <metadata-card/>
    <template v-if="healthExperiment && healthChecks?.length > 0">
      <v-divider class="mt-4 mb-1"/>
      <health-checks-card :model-value="healthChecks"/>
    </template>
    <with-status v-if="traits['smartcore.bos.Status']" :name="deviceId" v-slot="{resource}">
      <v-divider class="mt-4 mb-1"/>
      <status-log-card v-bind="resource"/>
    </with-status>
    <with-air-temperature v-if="traits['smartcore.traits.AirTemperature']" :name="deviceId" v-slot="{resource, update}">
      <v-divider class="mt-4 mb-1"/>
      <air-temperature-card v-bind="resource" @update-air-temperature="update"/>
    </with-air-temperature>
    <with-on-off v-if="traits['smartcore.traits.OnOff']" :name="deviceId" v-slot="{resource, update}">
      <on-off-card v-bind="resource" @update-on-off="update" :name="deviceId"/>
    </with-on-off>
    <with-air-quality v-if="traits['smartcore.traits.AirQualitySensor']" :name="deviceId" v-slot="{resource}">
      <v-divider class="mt-4 mb-1"/>
      <air-quality-card v-bind="resource" :name="deviceId"/>
    </with-air-quality>
    <v-divider v-if="traits['smartcore.traits.Light']" class="mt-4 mb-1"/>

    <light-card v-if="traits['smartcore.traits.Light']" :name="deviceId"/>

    <v-divider v-if="traits['smartcore.traits.OccupancySensor']" class="mt-4 mb-1"/>
    <with-occupancy v-if="traits['smartcore.traits.OccupancySensor']" :name="deviceId" v-slot="{resource}">
      <occupancy-card v-bind="resource" :name="deviceId"/>
    </with-occupancy>
    <with-electric-demand v-if="traits['smartcore.traits.Electric']" :name="deviceId" v-slot="{resource}">
      <v-divider class="mt-4 mb-1"/>
      <electric-demand-card v-bind="resource"/>
    </with-electric-demand>
    <with-energy-storage v-if="traits['smartcore.traits.EnergyStorage']" :name="deviceId" v-slot="{resource}">
      <v-divider class="mt-4 mb-1"/>
      <energy-storage-card v-bind="resource"/>
    </with-energy-storage>
    <with-meter v-if="traits['smartcore.bos.Meter']" :name="deviceId" v-slot="{resource, info}">
      <v-divider class="mt-4 mb-1"/>
      <meter-card v-bind="resource" :info="info?.response" :name="deviceId"/>
    </with-meter>
    <with-transport v-if="traits['smartcore.bos.Transport']" :name="deviceId">
      <template #transport="{resource, info}">
        <v-divider class="mt-4 mb-1"/>
        <transport-card :value="resource.value" :info="info?.response"/>
      </template>
      <template #history="{history}">
        <v-divider class="mt-4 mb-1"/>
        <transport-history-card :history="history"/>
      </template>
    </with-transport>
    <v-divider v-if="traits['smartcore.bsp.EmergencyLight']" class="mt-4 mb-1"/>
    <emergency-light :name="deviceId" v-if="traits['smartcore.bsp.EmergencyLight']"/>
    <v-divider v-if="traits['smartcore.traits.Mode']" class="mt-4 mb-1"/>
    <mode-card :name="deviceId" v-if="traits['smartcore.traits.Mode']"/>
    <v-divider v-if="traits['smartcore.bos.UDMI']" class="mt-4 mb-1"/>
    <udmi-card :name="deviceId" v-if="traits['smartcore.bos.UDMI']"/>
  </span>
</template>

<script setup>
import {useExperiment} from '@/composables/experiments.js';
import AirQualityCard from '@/traits/airQuality/AirQualityCard.vue';
import WithAirQuality from '@/traits/airQuality/WithAirQuality.vue';
import AirTemperatureCard from '@/traits/airTemperature/AirTemperatureCard.vue';
import WithAirTemperature from '@/traits/airTemperature/WithAirTemperature.vue';
import ElectricDemandCard from '@/traits/electricDemand/ElectricDemandCard.vue';
import WithElectricDemand from '@/traits/electricDemand/WithElectricDemand.vue';
import EmergencyLight from '@/traits/emergency/EmergencyLight.vue';
import EnergyStorageCard from '@/traits/energyStorage/EnergyStorageCard.vue';
import WithEnergyStorage from '@/traits/energyStorage/WithEnergyStorage.vue';
import HealthChecksCard from '@/traits/health/HealthChecksCard.vue';
import LightCard from '@/traits/light/LightCard.vue';
import MetadataCard from '@/traits/metadata/MetadataCard.vue';
import MeterCard from '@/traits/meter/MeterCard.vue';
import WithMeter from '@/traits/meter/WithMeter.vue';
import ModeCard from '@/traits/mode/ModeCard.vue';
import OccupancyCard from '@/traits/occupancy/OccupancyCard.vue';
import WithOccupancy from '@/traits/occupancy/WithOccupancy.vue';
import OnOffCard from '@/traits/onOff/OnOffCard.vue';
import WithOnOff from '@/traits/onOff/WithOnOff.vue';
import StatusLogCard from '@/traits/status/StatusLogCard.vue';
import WithStatus from '@/traits/status/WithStatus.vue';
import TransportCard from '@/traits/transport/TransportCard.vue';
import TransportHistoryCard from '@/traits/transport/TransportHistoryCard.vue';
import WithTransport from '@/traits/transport/WithTransport.vue';
import UdmiCard from '@/traits/udmi/UdmiCard.vue';

defineProps({
  deviceId: {
    type: String,
    default: ''
  },
  traits: {
    type: Object,
    default: () => {
    }
  },
  healthChecks: {
    /** @type {import('vue').PropType<import('@vanti-dev/sc-bos-ui-gen/proto/health_pb').HealthCheck.AsObject[]>} */
    type: Array,
    default: null
  }
});

const healthExperiment = useExperiment('health');
</script>
