import {closeResource, newResourceValue} from '@/api/resource';
import {pullStorage} from '@/api/sc/traits/storage.js';
import {toQueryObject, watchResource} from '@/util/traits.js';
import {computed, onScopeDispose, reactive, toRefs, toValue} from 'vue';

/**
 * @param {MaybeRefOrGetter<string|PullStorageRequest.AsObject>} query
 * @param {MaybeRefOrGetter<boolean>=} paused
 * @return {ToRefs<ResourceValue<Storage.AsObject, PullStorageResponse>>}
 */
export function usePullStorage(query, paused = false) {
  const resource = reactive(newResourceValue());
  onScopeDispose(() => closeResource(resource));

  const queryObject = computed(() => toQueryObject(query));

  watchResource(
      () => toValue(queryObject),
      () => toValue(paused),
      (req) => {
        pullStorage(req, resource);
        return () => closeResource(resource);
      }
  );

  return toRefs(resource);
}
