import {defineStore} from 'pinia';
import {ref} from 'vue';
import {useUiConfigStore} from './uiConfig.js';
import {getMeterReading, getFirstMeterReadingInPeriod} from '@/api/sc/traits/meter.js';

export const useMeterUsageStore = defineStore('nabersdashboard:meterUsage', () => {
  const waterUsage  = ref(null);
  const energyUsage = ref(null);
  const loading     = ref(false);
  const error       = ref(null);

  /**
   * Fetch the midnight baseline and current reading for each configured meter,
   * then compute the daily usage delta.
   *
   * @return {Promise<void>}
   */
  async function refresh() {
    const uiConfig   = useUiConfigStore();
    const waterName  = uiConfig.config.waterMeterName;
    const energyName = uiConfig.config.energyMeterName;
    if (!waterName && !energyName) return;

    loading.value = true;
    error.value   = null;

    const midnight    = new Date();
    midnight.setHours(0, 0, 0, 0);
    const midnightEnd = new Date(midnight.getTime() + 60_000);

    /**
     * @param {string} name
     * @return {Promise<number>}
     */
    async function dailyDelta(name) {
      const [midnightRec, current] = await Promise.all([
        getFirstMeterReadingInPeriod(name, midnight, midnightEnd),
        getMeterReading(name)
      ]);
      const base = midnightRec?.meterReading?.usage ?? current.usage;
      return Math.max(0, current.usage - base);
    }

    try {
      const tasks = [];
      if (waterName)  tasks.push(dailyDelta(waterName).then(v  => { waterUsage.value  = v; }));
      if (energyName) tasks.push(dailyDelta(energyName).then(v => { energyUsage.value = v; }));
      await Promise.all(tasks);
    } catch (e) {
      error.value = e;
    } finally {
      loading.value = false;
    }
  }

  return {waterUsage, energyUsage, loading, error, refresh};
});
