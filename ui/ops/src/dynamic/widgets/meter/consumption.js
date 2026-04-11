import {usePullMetadata} from '@/traits/metadata/metadata.js';
import {useMeterReadingsAt} from '@/traits/meter/meter.js';
import {computed, effectScope, reactive, toValue, watch} from 'vue';

/**
 * Returns the consumption (diff between two readings) for each edge for the given name.
 * The consumption at index i is the diff between the reading at edges[i] and edges[i+1].
 *
 * @param {import('vue').MaybeRefOrGetter<string>} name
 * @param {import('vue').MaybeRefOrGetter<Date[]>} edges
 * @return {import('vue').ComputedRef<{x:Date, y:number|null}[]>}
 */
export function useMeterConsumption(name, edges) {
  const readings = useMeterReadingsAt(name, edges);
  return computed(() => {
    const res = [];
    const _edges = toValue(edges);
    const _readings = toValue(readings);
    for (let i = 1; i < _edges.length; i++) {
      const startEdge = _edges[i - 1];
      const startReading = _readings[i - 1];
      const endReading = _readings[i];
      if (!startReading || !endReading) {
        res.push({x: startEdge, y: null});
        continue;
      }
      const consumption = endReading.usage - startReading.usage;
      // Use Math.max to ensure non-negative consumption (handles meter resets or data anomalies)
      res.push({x: startEdge, y: Math.max(0, consumption)});
    }
    return res;
  });
}


/**
 * @typedef {Object} SubConsumption
 * @property {import('vue').MaybeRefOrGetter<string>} title
 * @property {import('vue').ComputedRef<{x:Date, y:number|null}[]>} consumption
 * @property {function():void} stop
 */
/**
 * @typedef {Object} ConfigSubName
 * @property {string} name
 * @property {string} [title]
 */
/**
 * @param {import('vue').MaybeRefOrGetter<(string|ConfigSubName)[]>} names
 * @param {import('vue').MaybeRefOrGetter<Date[]>} edges
 * @return {import('vue').Reactive<Record<string, SubConsumption>>}
 */
export function useMetersConsumption(names, edges) {
  const res = reactive({});
  watch(() => toValue(names), (names) => {
    const toStop = Object.fromEntries(Object.entries(res)); // clone
    for (const item of names) {
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
        const consumption = {consumption: useMeterConsumption(name, edges), stop: () => scope.stop()};
        // use the configured title if possible, otherwise get it from the metadata, or just fall back to the name
        if (title) {
          consumption.title = title;
        } else {
          const {value: md} = usePullMetadata(name);
          consumption.title = computed(() => {
            const mdTitle = md.value?.appearance?.title;
            if (mdTitle) return mdTitle;
            return name;
          })
        }

        res[name] = consumption;
      });
    }

    for (const [name, {stop}] of Object.entries(toStop)) {
      stop();
      delete res[name];
    }
  }, {immediate: true});
  return res;
}
