import {usePullElectricDemand} from '@/traits/electricDemand/electric.js';
import {usePullMetadata} from '@/traits/metadata/metadata.js';
import {isNullOrUndef} from '@/util/types.js';
import {computed, effectScope, reactive, toValue, watch} from 'vue';
import * as colors from 'vuetify/util/colors';

/**
 * @typedef {Object} ElectricDemandRecord
 * @property {import('vue').MaybeRefOrGetter<string>} title
 * @property {import('vue').ComputedRef<number | null>} demand
 * @property {function():void} stop
 */

/**
 * @param {import('vue').MaybeRefOrGetter<(string | {title?:string, name:string})[]>} queries
 * @param {import('vue').ComputedRef<string>} metric
 * @return {import('vue').ComputedRef<ElectricDemandRecord[]>} - in queries order
 */
export function usePullElectricDemands(queries, metric) {
  const res = reactive(
      /** @type {Record<string, ElectricDemandRecord>} */
      {}
  );

  watch(() => toValue(queries), (queries) => {
    const toStop = Object.fromEntries(Object.entries(res)); // clone
    for (const item of queries) {
      let name = item;
      let title = undefined;
      if (typeof name === 'object') {
        name = item.name;
        title = item.title;
      }
      if (res[name]) {
        delete toStop[name];
        continue;
      }
      const scope = effectScope();
      scope.run(() => {
        const record = {demand: usePullElectricDemandRecord(name, metric), stop: () => scope.stop()};
        if (title) {
          // make sure record.title is always a computed ref as Vue optimises this kind of thing
          record.title = computed(() => title);
        } else {
          const {value: md} = usePullMetadata(name);
          record.title = computed(() => {
            const mdTitle = md.value?.appearance?.title;
            if (mdTitle) return mdTitle;
            return name;
          })
        }

        res[name] = record;
      });
    }

    for (const [name, {stop}] of Object.entries(toStop)) {
      stop();
      delete res[name];
    }
  }, {immediate: true});

  return computed(() => {
    return toValue(queries).map(q => {
      if (typeof q === 'object') {
        return res[q.name];
      }
      return res[q];
    })
  });
}

/**
 * @param {import('vue').MaybeRefOrGetter<string|PullDemandRequest.AsObject>} query
 * @param {import('vue').ComputedRef<string>} metric
 * @return {import('vue').ComputedRef<number|null>}
 */
export function usePullElectricDemandRecord(query, metric) {
  const {value} = usePullElectricDemand(query);
  return computed(() => {
    const v = value.value;
    if (!v) return null;
    const m = toValue(metric);
    if ((m === 'realPower') && (typeof v.realPower === 'number' && !isNaN(v.realPower))) {
      return v.realPower / 1000; // in kW
    }
    if ((m === 'current') && (typeof v.current === 'number' && !isNaN(v.current))) {
      return v.current;
    }
    return null;
  })
}

/**
 * @param {import('vue').MaybeRefOrGetter<number|null>} total
 * @param {import('vue').MaybeRefOrGetter<Array<{title:string,value:number}>>} parts
 * @return {import('vue').ComputedRef<import('chart.js').ChartData>}
 */
export function useChartTotalDataset(total, parts) {
  return computed(() => {
    const _total = toValue(total);
    const _parts = toValue(parts) || [];

    const labels = [];
    const data = [];

    let partSum = 0;
    for (const part of _parts) {
      if (isNullOrUndef(part.demand)) continue;
      labels.push(part.title);
      data.push(part.demand);
      partSum += part.demand;
    }

    if (_parts.length > 0 && !isNullOrUndef(_total)) {
      labels.push('Other');
      data.push(_total - partSum);
    }

    const backgroundColor = [
      colors.blue.base,
      colors.green.base,
      colors.orange.base,
      colors.yellow.base,
      colors.red.base,
    ].filter(Boolean)

    return {
      labels,
      datasets: [{
        data,
        backgroundColor,
      }],
    }
  });
}