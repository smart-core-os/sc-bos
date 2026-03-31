<template>
  <v-card class="weather-container" :loading="loading" :elevation="0">
    <v-toolbar v-if="!hideToolbar" color="transparent">
      <v-toolbar-title class="text-h4">{{ title }}</v-toolbar-title>
    </v-toolbar>
    <template v-if="!apiKey">
      <v-card-text class="weather-details py-0">
        <div class="text-h4">API Key Required</div>
      </v-card-text>
    </template>
    <template v-else-if="!loading && error">
      <v-card-text class="weather-details py-0">
        <div class="text-h4 text-error-darken-1">Error: {{ error }}</div>
      </v-card-text>
    </template>
    <template v-else-if="loading">
      <v-card-text class="weather-details py-0">Loading weather data...</v-card-text>
    </template>
    <template v-else>
      <v-card-text class="weather-details py-0">
        <div class="weather-temp text-h2">
          <div>{{ Math.round(weatherData.main.temp) }}°C</div>
          <owm-icon class="owm-icon"/>
        </div>
        <div class="weather-icon">
          <img :src="weatherIconUrl" :alt="weatherData.weather[0].description">
        </div>
        <div class="weather-info">
          <div class="weather-description">{{ weatherData.weather[0].description }}</div>
          <div class="weather-stats">
            <span>Humidity: {{ weatherData.main.humidity }}%</span>
            <span>Wind: {{ Math.round(weatherData.wind.speed) }} m/s</span>
          </div>
        </div>
      </v-card-text>
    </template>
  </v-card>
</template>

<script setup>
import {MINUTE} from '@/components/now.js';
import {computed, onBeforeUnmount, onMounted, ref} from 'vue';
import OwmIcon from './open-weather-logo.svg'

// Props
const props = defineProps({
  apiKey: {
    type: String,
    required: true
  },
  city: {
    type: String,
    default: 'London'
  },
  country: {
    type: String,
    default: ''
  },
  refreshInterval: {
    type: Number,
    default: 30 * MINUTE
  },
  title: {
    type: String,
    default: 'Weather'
  },
  hideToolbar: {
    type: Boolean,
    default: false
  },
});

const weatherData = ref(null);
const loading = ref(true);
const error = ref(null);

const weatherIconUrl = computed(() => {
  if (!weatherData.value) return '';
  const iconCode = weatherData.value.weather[0].icon;
  return `https://openweathermap.org/img/wn/${iconCode}@2x.png`;
});

const fetchWeather = async () => {
  loading.value = !weatherData.value;
  try {
    let url = `https://api.openweathermap.org/data/2.5/weather?q=${props.city}&units=metric&appid=${props.apiKey}`

    if (props.country !== '') {
      url = `https://api.openweathermap.org/data/2.5/weather?q=${props.city},${props.country}&units=metric&appid=${props.apiKey}`
    }

    const response = await fetch(url);

    if (!response.ok) {
      if (response.status === 401) {
        error.value = 'Invalid API Key';
      } else if (response.status === 404) {
        error.value = `City "${props.city}" not found`;
      } else {
        error.value = `Weather API error: ${response.statusText}`;
      }
      console.error('Error fetching weather data:', error.value);
      return;
    }

    weatherData.value = await response.json();
  } catch (err) {
    error.value = err.message || err;
    console.error('Error fetching weather data:', err);
  } finally {
    loading.value = false;
  }
};

let intervalId = null;
onMounted(() => {
  fetchWeather();
  intervalId = setInterval(fetchWeather, props.refreshInterval);
});
onBeforeUnmount(() => clearInterval(intervalId));
</script>

<style scoped>
.weather-container {
  display: flex;
  flex-direction: column;
}

.weather-details {
  display: flex;
  flex: 1;
  gap: 8px;
  align-items: center;
  justify-content: center;
}

.weather-icon {
  width: 70px;
  height: 70px;
}

.weather-icon img {
  width: 100%;
  height: 100%;
  vertical-align: top;
}

.weather-info {
}

.weather-description {
  text-transform: capitalize;
  margin-bottom: 4px;
}

.weather-stats {
  display: flex;
  flex-direction: column;
  opacity: .8;
}

.weather-temp {
  position: relative;
  line-height: 1;
}

.owm-icon {
  position: absolute;
  top: calc(100% + 2px);
  left: 0;
  opacity: .5;
  height: 1.2ex;
  vertical-align: top;
}
</style>