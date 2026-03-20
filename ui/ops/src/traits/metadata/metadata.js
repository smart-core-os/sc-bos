import {closeResource, newResourceValue} from '@/api/resource.js';
import {pullMetadata} from '@/api/sc/traits/metadata.js';
import {toQueryObject, watchResource} from '@/util/traits.js';
import {isNullOrUndef} from '@/util/types.js';
import {computed, onScopeDispose, reactive, toRefs, toValue} from 'vue';

/**
 * @typedef {
 *   import('@smart-core-os/sc-bos-ui-gen/proto/smartcore/bos/metadata/v1/metadata_pb').PullMetadataRequest
 * } PullMetadataRequest
 * @typedef {
 *   import('@smart-core-os/sc-bos-ui-gen/proto/smartcore/bos/metadata/v1/metadata_pb').PullMetadataResponse
 * } PullMetadataResponse
 * @typedef {import('vue').Ref} Ref
 * @typedef {import('vue').UnwrapNestedRefs} UnwrapNestedRefs
 * @typedef {import('vue').ToRefs} ToRefs
 * @typedef {import('vue').ComputedRef} ComputedRef
 * @typedef {import('@/api/resource').ResourceValue} ResourceValue
 * @typedef {import('@/api/resource').ActionTracker} ActionTracker
 */

/**
 * @param {MaybeRefOrGetter<string|Partial<PullMetadataRequest.AsObject>>} query
 * @param {MaybeRefOrGetter<boolean>=} paused
 * @return {ToRefs<ResourceValue<Metadata.AsObject, PullMetadataResponse>>}
 */
export function usePullMetadata(query, paused = false) {
  const resource = reactive(
      /** @type {ResourceValue<Metadata.AsObject, PullMetadataResponse>} */
      newResourceValue()
  );
  onScopeDispose(() => closeResource(resource));

  const queryObject = computed(() => toQueryObject(query));

  watchResource(
      () => toValue(queryObject),
      () => toValue(paused),
      (req) => {
        pullMetadata(req, resource);
        return () => closeResource(resource);
      }
  );

  return toRefs(resource);
}

/**
 * Captures all populated fields of the given metadata object(s).
 * Nested fields are dot separated.
 * Arrays are treated as leaf fields, arrays of objects are not traversed.
 * Excludes name and traits properties.
 *
 * @example
 * const fields = usePopulatedFields({name: 'foo', appearance: {title: 'bar'}});
 * console.log(fields.value); // ['appearance.title']
 *
 * @param {MaybeRefOrGetter<Metadata.AsObject | Metadata.AsObject[]>} metadata
 * @return {ComputedRef<string[]>}
 */
export function usePopulatedFields(metadata) {
  const metadataList = computed(() => {
    let v = toValue(metadata) ?? [];
    if (!Array.isArray(v)) v = [v];
    return v;
  });
  // all the populated fields from all the devices
  return computed(() => {
    const props = /** @type {Set<string>} */ new Set();
    for (const device of metadataList.value) {
      for (const [key, obj] of Object.entries(device)) {
        if (key === 'name' || key === 'traits') continue; // handled explicitly
        collectFields(props, key, obj, o => !isNullOrUndef(o));
      }
    }
    return Array.from(props).sort();
  });
}

const collectFields = (dst, key, obj, include) => {
  if (!include(obj)) return;
  if (key.endsWith('Map')) return obj.forEach(([key2]) => dst.add(key + '.' + key2));
  if (Array.isArray(obj)) return dst.add(key);
  if (typeof obj === 'object') {
    for (const [key2, obj2] of Object.entries(obj)) {
      collectFields(dst, key + '.' + key2, obj2, include);
    }
    return;
  }
  dst.add(key);
};
