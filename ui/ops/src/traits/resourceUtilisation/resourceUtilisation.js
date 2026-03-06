import {closeResource, newResourceValue} from '@/api/resource';
import {pullResourceUtilisation} from '@/api/sc/traits/resource-utilisation';
import {toQueryObject, watchResource} from '@/util/traits.js';
import {computed, onScopeDispose, reactive, toRefs, toValue} from 'vue';

/**
 * @param {MaybeRefOrGetter<string|PullResourceUtilisationRequest.AsObject>} query
 * @param {MaybeRefOrGetter<boolean>=} paused
 * @return {ToRefs<ResourceValue<ResourceUtilisation.AsObject, PullResourceUtilisationResponse>>}
 */
export function usePullResourceUtilisation(query, paused = false) {
  const resource = reactive(newResourceValue());
  onScopeDispose(() => closeResource(resource));

  const queryObject = computed(() => toQueryObject(query));

  watchResource(
      () => toValue(queryObject),
      () => toValue(paused),
      (req) => {
        pullResourceUtilisation(req, resource);
        return () => closeResource(resource);
      }
  );

  return toRefs(resource);
}

/**
 * @param {MaybeRefOrGetter<ResourceUtilisation.AsObject>} value
 * @return {{cpuPercent: ComputedRef<number|null>, memPercent: ComputedRef<number|null>}}
 */
export function useResourceUtilisation(value) {
  const _v = computed(() => toValue(value));

  const cpuPercent = computed(() => _v.value?.cpu?.percentUtilised ?? null);
  const memPercent = computed(() => _v.value?.memory?.percentUsed ?? null);

  return {cpuPercent, memPercent};
}
