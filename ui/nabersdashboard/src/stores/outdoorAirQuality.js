import {defineStore} from 'pinia';
import {computed, ref, watch} from 'vue';
import {useUiConfigStore} from '@/stores/uiConfig.js';

const mdiEmotes = {
  1: {text: 'Good', emote: 'mdi-emoticon-cool', color: '#139a4c'},
  2: {text: 'Fair', emote: 'mdi-emoticon-happy', color: '#139a4c'},
  3: {text: 'Moderate', emote: 'mdi-emoticon-neutral', color: '#FF9E18'},
  4: {text: 'Poor', emote: 'mdi-emoticon-sad', color: '#d33828'},
  5: {text: 'Very Poor', emote: 'mdi-emoticon-sick', color: '#d33828'}
};

export const useOutdoorAirQualityStore = defineStore('nabersdashboard:outdoorAirQuality', () => {
  const uiConfig = useUiConfigStore();

  const outdoorAqData = ref(null);
  const outdoorWeatherData = ref(null);
  const loading = ref(true);
  const pollInterval = 60 * 60 * 1000; // 1 hour
  let intervalId = null;

  const apiKey = computed(() => uiConfig.owmKey);

  // Where to ask for outdoor conditions. Comes from config because the same
  // build serves any site, and there is no sensible default to fall back on:
  // a guessed location would quietly report another city's air.
  const coords = computed(() => {
    const {latitude, longitude} = uiConfig.config ?? {};
    if (typeof latitude !== 'number' || typeof longitude !== 'number') return null;
    return {lat: latitude, lon: longitude};
  });

  const airPollutionUrl = computed(() => {
    if (!coords.value) return null;
    const {lat, lon} = coords.value;
    return `https://api.openweathermap.org/data/2.5/air_pollution?lat=${lat}&lon=${lon}&appid=${apiKey.value}`;
  });

  const weatherUrl = computed(() => {
    if (!coords.value) return null;
    const {lat, lon} = coords.value;
    return `https://api.openweathermap.org/data/2.5/weather?lat=${lat}&lon=${lon}&appid=${apiKey.value}&units=metric`;
  });

  const hasData = computed(() => {
    return !loading.value && outdoorAqData.value?.list?.length > 0;
  });

  const outdoorComponents = computed(() => {
    if (!hasData.value) return null;
    return outdoorAqData.value.list[0].components;
  });

  /** Maps outdoor data to the same metric keys used by indoor sensors */
  const outdoorBaselines = computed(() => {
    const components = outdoorComponents.value;
    const weather = outdoorWeatherData.value;
    return {
      particulateMatter25: components?.pm2_5 ?? null,
      particulateMatter10: components?.pm10 ?? null,
      carbonDioxideLevel: 500,         // typical outdoor CO2 baseline is 400 - 450, (between 400 - 600 in urban areas (ppm))
      volatileOrganicCompounds: 0.1,   // typical outdoor VOC baseline (ppm)
      ambientTemperature: weather?.main?.temp ?? null,
      ambientHumidity: weather?.main?.humidity ?? null,
      brightnessLux: 500,  // typical office lighting target (lux)
      soundPressureLevel: 45  // typical office ambient noise (dB)
    };
  });

  const currentAqiIndex = computed(() => {
    if (!hasData.value) return null;
    return outdoorAqData.value.list[0].main.aqi ?? null;
  });

  const currentAqi = computed(() => {
    if (loading.value || !outdoorAqData.value) {
      return {emote: 'mdi-loading', color: 'grey', text: 'Loading...'};
    }
    if (!outdoorAqData.value.list || outdoorAqData.value.list.length === 0) {
      return {emote: 'mdi-alert-circle', color: 'grey', text: 'No data'};
    }
    const aqi = outdoorAqData.value.list[0].main.aqi;
    return mdiEmotes[aqi] || {emote: 'mdi-help-circle', color: 'grey', text: 'Unknown'};
  });

  /** @return {void} */
  function fetchData() {
    if (!apiKey.value || !coords.value) return;
    loading.value = true;
    Promise.all([
      fetch(airPollutionUrl.value).then(r => r.json()),
      fetch(weatherUrl.value).then(r => r.json())
    ])
        .then(([aqData, weatherData]) => {
          outdoorAqData.value = aqData;
          outdoorWeatherData.value = weatherData;
        })
        .catch(error => {
          console.error('Error fetching outdoor data:', error);
        })
        .finally(() => {
          loading.value = false;
        });
  }

  /** @return {void} */
  function startPolling() {
    fetchData();
    if (!intervalId) {
      intervalId = setInterval(fetchData, pollInterval);
    }
  }

  /** @return {void} */
  function stopPolling() {
    if (intervalId) {
      clearInterval(intervalId);
      intervalId = null;
    }
  }

  // Auto-start once both the API key and the site's coordinates are known.
  watch([apiKey, coords], ([newKey, newCoords]) => {
    if (newKey && newCoords) {
      startPolling();
    }
  }, {immediate: true});

  const outdoorWeatherSummary = computed(() => {
    const w = outdoorWeatherData.value;
    if (!w?.main) return null;
    return {
      temp: w.main.temp,
      humidity: w.main.humidity
    };
  });

  return {
    loading,
    hasData,
    outdoorComponents,
    outdoorBaselines,
    outdoorWeatherSummary,
    currentAqi,
    currentAqiIndex,
    startPolling,
    stopPolling
  };
});
