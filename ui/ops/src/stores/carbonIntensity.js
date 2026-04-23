import {useNow} from '@/components/now';
import {DAY, HOUR, MINUTE} from '@/util/date.js';
import {acceptHMRUpdate, defineStore} from 'pinia';
import {computed, ref, watchEffect} from 'vue';

export const useCarbonIntensity = defineStore('carbonIntensity', () => {
  const now24h = useNow(30 * MINUTE);
  const now7d = useNow(2 * HOUR);
  const now30d = useNow(6 * HOUR);

  const last24h = ref(null);
  const last7d = ref(null);
  const last30d = ref(null);

  // Creating a separate watchEffect for different data resolutions
  // For last 24 hours data, which might update more frequently
  watchEffect(async () => {
    const date = now24h.now.value;
    last24h.value = await carbonIntensityApi.intensity.pt24h(date);
  });

  // For last 7 days data, assuming it updates once a day
  watchEffect(async () => {
    const date = now7d.now.value;
    last7d.value = await carbonIntensityApi.intensity.dateFromTo(new Date(date - 7 * DAY), date);
  });

  // For last 30 days data, assuming it updates once a day
  watchEffect(async () => {
    const date = now30d.now.value;
    last30d.value = await carbonIntensityApi.intensity.dateFromTo(new Date(date - 30 * DAY), date);
  });

  // Process the data for the last 24 hours
  const last24Hours = computed(() =>
    last24h.value?.data?.map(d => ({
      from: new Date(d.from),
      to: new Date(d.to),
      intensity: d.intensity
    })) || []
  );

  // Process the data for the last 7 days
  const last7Days = computed(() =>
    last7d.value?.data?.map(d => ({
      from: new Date(d.from),
      to: new Date(d.to),
      intensity: d.intensity
    })) || []
  );

  // Process the data for the last 30 days
  const last30Days = computed(() =>
    last30d.value?.data?.map(d => ({
      from: new Date(d.from),
      to: new Date(d.to),
      intensity: d.intensity
    })) || []
  );

  return {
    last24Hours,
    last7Days,
    last30Days
  };
});

const baseUrl = 'https://api.carbonintensity.org.uk';
const carbonIntensityApi = {
  intensity: {
    date(date) {
      let url = `${baseUrl}/intensity/date`;
      if (date) url += `/${date.toISOString().substring(0, 10)}`;
      return fetch(url)
          .then(response => response.json());
    },
    fw24h(date) {
      if (!date) {
        date = new Date();
        date.setHours(date.getHours() - 24);
      }
      const url = `${baseUrl}/intensity/${date.toISOString()}/fw24h`;
      return fetch(url)
          .then(response => response.json());
    },
    pt24h(date) {
      if (!date) {
        date = new Date();
      }
      const url = `${baseUrl}/intensity/${date.toISOString()}/pt24h`;
      return fetch(url)
          .then(response => response.json());
    },
    dateFromTo(from, to) {
      const url = `${baseUrl}/intensity/${from.toISOString()}/${to.toISOString()}`;
      return fetch(url)
          .then(response => response.json());
    }
  }
};

// make sure to pass the right store definition, `useCarbon` in this case.
if (import.meta.hot) {
  import.meta.hot.accept(acceptHMRUpdate(useCarbonIntensity, import.meta.hot));
}
