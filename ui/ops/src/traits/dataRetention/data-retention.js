import {closeResource, newResourceValue} from '@/api/resource.js';
import {pullDataRetention} from '@/api/sc/traits/data-retention.js';
import {toQueryObject, watchResource} from '@/util/traits.js';
import {computed, onScopeDispose, reactive, toRefs, toValue} from 'vue';

/**
 * @param {MaybeRefOrGetter<string|PullDataRetentionRequest.AsObject>} query
 * @param {MaybeRefOrGetter<boolean>=} paused
 * @return {ToRefs<ResourceValue<DataRetention.AsObject, PullDataRetentionResponse>>}
 */
export function usePullDataRetention(query, paused = false) {
  const resource = reactive(newResourceValue());
  onScopeDispose(() => closeResource(resource));

  const queryObject = computed(() => toQueryObject(query));

  watchResource(
      () => toValue(queryObject),
      () => toValue(paused),
      (req) => {
        pullDataRetention(req, resource);
        return () => closeResource(resource);
      }
  );

  return toRefs(resource);
}
