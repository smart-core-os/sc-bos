import {closeResource, newResourceValue} from '@/api/resource.js';
import {listServices, pullService, pullServiceMetadata, pullServices} from '@/api/ui/services.js';
import useCollection from '@/composables/collection.js';
import {toQueryObject, watchResource} from '@/util/traits.js';
import {computed, onScopeDispose, reactive, toRefs, toValue} from 'vue';

/**
 * @param {MaybeRefOrGetter<string|Partial<ListServicesRequest.AsObject & PullServicesRequest.AsObject>>} request
 * @param {MaybeRefOrGetter<Partial<UseCollectionOptions>>?} options
 * @return {UseCollectionResponse<Service.AsObject>}
 */
export function useServicesCollection(request, options) {
  const normOptions = computed(() => {
    const optArg = toValue(options);
    return {
      cmp: (a, b) => a.id.localeCompare(b.id),
      ...optArg
    };
  });
  const normRequest = computed(() => toQueryObject(request));
  const client = {
    async listFn(req, tracker) {
      const res = await listServices(req, tracker);
      return {
        items: res.servicesList,
        nextPageToken: res.nextPageToken,
        totalSize: res.totalSize
      };
    },
    pullFn(req, resource) {
      pullServices(req, resource);
    }
  };
  return useCollection(normRequest, client, normOptions);
}


/**
 * @param {import('vue').MaybeRefOrGetter<string|PullServiceMetadataRequest.AsObject>} request
 * @param {import('vue').MaybeRefOrGetter<{paused?: boolean}>?} options
 * @return {import('vue').ToRefs<ResourceValue<ServiceMetadata.AsObject, PullServiceMetadataRequest>>}
 */
export function usePullServiceMetadata(request, options) {
  const resource = reactive(
      /** @type {ResourceValue<ServiceMetadata.AsObject, PullServiceMetadataRequest>} */
      newResourceValue()
  );
  onScopeDispose(() => closeResource(resource));

  const requestObject = computed(() => toQueryObject(request));

  watchResource(
      () => toValue(requestObject),
      () => toValue(options)?.paused ?? false,
      (req) => {
        pullServiceMetadata(req, resource);
        return () => closeResource(resource);
      }
  );

  return toRefs(resource);
}

/**
 * @param {import('vue').MaybeRefOrGetter<PullServicesRequest.AsObject>} request - must have an id, should have a name.
 * @param {import('vue').MaybeRefOrGetter<{paused?: boolean}>?} options
 * @return {ToRefs<ResourceValue<Service.AsObject, PullServiceResponse>>}
 */
export function usePullService(request, options) {
  const resource = reactive(
      newResourceValue()
  );
  onScopeDispose(() => closeResource(resource));

  const requestObject = computed(() => toQueryObject(request));

  watchResource(
      () => toValue(requestObject),
      () => toValue(options)?.paused ?? false,
      (req) => {
        pullService(req, resource);
        return () => closeResource(resource);
      }
  );

  return toRefs(resource);
}

/**
 * Checks whether the given node name has an active hub system (primary or proxy).
 *
 * @param {MaybeRefOrGetter<string>} name
 * @return {{hasHubSystem: ComputedRef<boolean>, loading: ComputedRef<boolean>, error: ComputedRef<ResourceError>}}
 */
export function useHasHubSystem(name) {
  const {value, streamError: error, loading} = usePullService(
      () => ({name: toValue(name) + '/systems', id: 'hub'})
  );
  const hasHubSystem = computed(() => value.value ? value.value.active : false);
  const isProxyHub = computed(() => {
    if (!value.value?.configRaw) return false;
    try {
      return JSON.parse(value.value.configRaw)?.storage?.type === 'proxy';
    } catch {
      return false;
    }
  });
  return {hasHubSystem, isProxyHub, loading, error};
}

/**
 * Checks whether the given node name is a gateway.
 *
 * @param {MaybeRefOrGetter<string>} name
 * @return {{isGateway: Ref<boolean>, loading: ComputedRef<boolean>, error: ComputedRef<ResourceError>}}
 */
export function useIsGateway(name) {
  const {value, streamError: error, loading} = usePullService({
    name: name + '/systems',
    id: 'gateway'
  });
  // the gateway system used to be called the proxy system, so check for both
  const {value: legacyValue, streamError: legacyError, loading: legacyLoading} = usePullService({
    name: name + '/systems',
    id: 'proxy'
  });

  const isGateway = computed(() => {
    if (value.value) return value.value.active;
    if (legacyValue.value) return legacyValue.value.active;
    return false;
  });
  return {
    isGateway,
    loading: computed(() => loading.value || legacyLoading.value),
    error: computed(() => {
      if (!error.value) return null; // if the modern query worked, everything worked
      return error.value || legacyError.value;
    })
  };
}
