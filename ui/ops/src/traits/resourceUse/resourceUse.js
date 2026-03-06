import {closeResource, newResourceValue} from '@/api/resource';
import {pullResourceUse} from '@/api/sc/traits/resource-use';
import {toQueryObject, watchResource} from '@/util/traits.js';
import {computed, onScopeDispose, reactive, toRefs, toValue} from 'vue';

/**
 * @param {MaybeRefOrGetter<string|PullResourceUseRequest.AsObject>} query
 * @param {MaybeRefOrGetter<boolean>=} paused
 * @return {ToRefs<ResourceValue<ResourceUse.AsObject, PullResourceUseResponse>>}
 */
export function usePullResourceUse(query, paused = false) {
  const resource = reactive(newResourceValue());
  onScopeDispose(() => closeResource(resource));

  const queryObject = computed(() => toQueryObject(query));

  watchResource(
      () => toValue(queryObject),
      () => toValue(paused),
      (req) => {
        pullResourceUse(req, resource);
        return () => closeResource(resource);
      }
  );

  return toRefs(resource);
}

/**
 * @param {MaybeRefOrGetter<ResourceUse.AsObject>} value
 * @return {{cpuPercent: ComputedRef<number|null>, memPercent: ComputedRef<number|null>}}
 */
export function useResourceUse(value) {
  const _v = computed(() => toValue(value));

  const cpuPercent = computed(() => _v.value?.cpu?.percentUtilised ?? null);
  const memPercent = computed(() => _v.value?.memory?.percentUsed ?? null);

  return {cpuPercent, memPercent};
}
