import {ref, watch} from 'vue';

/**
 * Accumulates streamed log message batches into a capped buffer.
 * Each message is given a unique, increasing _key for use as a render key.
 *
 * @param {import('@/api/resource.js').ResourceValue} resource - resource whose .value is the latest batch
 * @param {{
 *   max?: number,
 *   onBatch?: (batch: Array<Object>) => void
 * }} [opts] - max caps the buffer size; onBatch is called before each batch is appended
 * @return {{messages: import('vue').Ref<Array<Object>>, clear: () => void}}
 */
export function useLogBuffer(resource, {max = 2000, onBatch} = {}) {
  const messages = ref([]);
  let _keyCounter = 0;

  watch(() => resource.value, (batch) => {
    if (!batch?.length) return;
    onBatch?.(batch);
    for (const m of batch) {
      m._key = _keyCounter++;
      messages.value.push(m);
    }
    if (messages.value.length > max) {
      messages.value.splice(0, messages.value.length - max);
    }
  });

  const clear = () => {
    messages.value = [];
  };

  return {messages, clear};
}
