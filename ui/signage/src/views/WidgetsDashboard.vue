<template>
  <section class="widgets">
    <AirQuality :name="props.airQuality.name"
                :min-value="props.airQuality.minValue"
                :max-value="props.airQuality.maxValue"
                id="AirQuality"/>
    <BuildingOccupancy :name="props.occupancy.name"
                       :max-value="props.occupancy.maxValue"
                       id="BuildingOccupancy"/>
    <EnergyUsage :name="props.energy.name"
                 id="EnergyUsage"/>
    <TemperatureSystems :name="props.temperature.name"
                        id="TemperatureSystems"/>
    <WaterUsage :name="props.water.name"
                id="WaterUsage"/>
    <footer>
      Powered by
      <img src="@/assets/smart-core-logo.svg" alt="SmartCore">
    </footer>
  </section>
</template>

<script setup>
import AirQuality from '@/views/AirQuality.vue';
import BuildingOccupancy from '@/views/BuildingOccupancy.vue';
import EnergyUsage from '@/views/EnergyUsage.vue';
import TemperatureSystems from '@/views/TemperatureSystems.vue';
import WaterUsage from '@/views/WaterUsage.vue';

const props = defineProps({
  airQuality: {
    type: Object,
    default: () => ({})
  },
  occupancy: {
    type: Object,
    default: () => ({})
  },
  energy: {
    type: Object,
    default: () => ({})
  },
  temperature: {
    type: Object,
    default: () => ({})
  },
  water: {
    type: Object,
    default: () => ({})
  }
});
</script>

<style lang="scss" scoped>
.widgets {
  display: grid;
  grid-template-areas:
    'Energy'
    'Temperature'
    'Water'
    'Building'
    'Flower';
  grid-template-columns: 1fr;
  grid-template-rows: 1fr 1fr 1fr 1fr 1fr;
  gap: 2vw 0;
  width: 100%;
  height: auto;

  // :deep(h2) {
  //   font-family: sans-serif;
  //   font-weight: 400;
  //   font-size: 30px;
  //   margin-top: 1rem;
  // }

  #WaterUsage,
  #BuildingOccupancy,
  #AirQuality,
  #EnergyUsage,
  #TemperatureSystems,
  #Logo {
    grid-area: Water;
    display: flex;
    justify-content: center;
    align-items: center;
  }

  #BuildingOccupancy {
    grid-area: Building;
  }

  #AirQuality {
    grid-area: Flower;
  }

  #EnergyUsage {
    grid-area: Energy;
  }

  #TemperatureSystems {
    grid-area: Temperature;
  }
}

footer {
  font-family: sans-serif;
  font-weight: 6s00;
  font-size: 14px;
  text-align: center;
  color: #000;

  img {
    display: block;
    height: 20px;
    width: 100%;
  }
}
</style>
