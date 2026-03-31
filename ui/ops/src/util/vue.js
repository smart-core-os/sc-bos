import {ref, toValue, watch} from 'vue';

/**
 * Returns a ref that can be written but will be overridden by the prop value when it changes.
 *
 * @template T
 * @param {import('vue').MaybeRefOrGetter<T>} prop
 * @return {import('vue').Ref<T>}
 */
export function useLocalProp(prop) {
  const local = ref(toValue(prop.value));
  watch(() => toValue(prop), (value) => {
    local.value = value;
  });
  return local;
}

/**
 * @typedef {Object} asyncWatchAction
 * @property {T} ov
 * @property {T} nv
 * @property {import('vue').OnCleanup} onCleanup
 * @template T
 */

/**
 * Like watch but guarantees that only one callback is executed at a time with the latest change.
 * If source changes frequently, intermediate changes are ignored, the last change will be executed after the async callback completes.
 * The value and oldValue arguments passed to the callback will be the first and last changes seen by the callback.
 * To cancel tasks when changes are made, use the onCleanup function.
 *
 * @param {import('vue').WatchSource | import('vue').WatchSource[] | import('vue').WatchEffect | object} source
 * @param {import('vue').WatchCallback} callback
 * @param {import('vue').WatchOptions} [options]
 * @return {{refresh: () => void}}
 */
export function asyncWatch(source, callback, options) {
  // What follows is a race safe way to perform requests whenever request changes.
  // We queue up tasks (calls to action) and only process the latest one.
  const activeTask = ref(/** @type {asyncWatchAction | null} */ null); // the task we are awaiting
  const nextTask = ref(/** @type {asyncWatchAction | null} */ null); // our queue, except we only care about the tip
  watch(source, (nv, ov, onCleanup) => {
    // keep the original old value if there is one
    nextTask.value = {nv, ov: nextTask.value?.ov ?? ov, onCleanup};
  }, options);
  watch(nextTask, (task) => {
    if (!task) return; // break the loop
    if (activeTask.value) return; // already running a task
    activeTask.value = task;
    nextTask.value = null;
  }, options);
  watch(activeTask, async (task) => {
    if (!task) return; // break the loop
    try {
      const {nv, ov, onCleanup} = task;

      await callback(nv, ov, onCleanup);
    } catch (e) {
      console.error('asyncAction error:', e);
    } finally {
      const next = nextTask.value;
      nextTask.value = null;
      activeTask.value = next;
    }
  }, options);

  return {
    refresh() {
      const v = toValue(source);
      nextTask.value = {nv: v, ov: v};
    }
  };
}