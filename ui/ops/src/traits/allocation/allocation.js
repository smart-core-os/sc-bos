/**
 * @typedef {import('@smart-core-os/sc-bos-ui-gen/proto/smartcore/bos/allocation/v1/allocation_pb').PullAllocationRequest} PullAllocationRequest
 * @typedef {import('@smart-core-os/sc-bos-ui-gen/proto/smartcore/bos/allocation/v1/allocation_pb').PullAllocationResponse} PullAllocationResponse
 * @typedef {import('@smart-core-os/sc-bos-ui-gen/proto/smartcore/bos/allocation/v1/allocation_pb').Allocation} Allocation
 * @typedef {import('vue').UnwrapNestedRefs} UnwrapNestedRefs
 * @typedef {import('vue').ToRefs} ToRefs
 * @typedef {import('vue').ComputedRef} ComputedRef
 * @typedef {import('vue').MaybeRefOrGetter} MaybeRefOrGetter
 * @typedef {import('@/api/resource').ResourceValue} ResourceValue
 */

import {closeResource, newResourceValue} from '@/api/resource.js';
import {pullAllocation} from '@/api/sc/traits/allocation.js';
import {toQueryObject, watchResource} from '@/util/traits.js';
import {computed, onScopeDispose, reactive, toRefs, toValue} from 'vue';

/**
 * @param {MaybeRefOrGetter<string|PullAllocationRequest.AsObject>} query
 * @param {MaybeRefOrGetter<boolean>=} paused
 * @return {ToRefs<ResourceValue<Allocation.AsObject, PullAllocationResponse>>}
 */
export function usePullAllocation(query, paused = false) {
 const resource = reactive(
   /** @type {ResourceValue<Allocation.AsObject, PullAllocationResponse>} */
   newResourceValue()
 );
 onScopeDispose(() => closeResource(resource));
 
 const queryObject = computed(() => toQueryObject(query));
 
 watchResource(
   () => toValue(queryObject),
   () => toValue(paused),
   (req) => {
    pullAllocation(req, resource);
    return () => closeResource(resource);
   }
 );
 
 return toRefs(resource);
}

/**
 * @param {MaybeRefOrGetter<Allocation.AsObject|null>} value
 * @return {{
 *   allocationTotal: ComputedRef<number>,
 *   unallocationTotal: ComputedRef<number>,
 *   table: ComputedRef<Array<{label:string, value:string}>>
 * }}
 */
export function useAllocation(value) {
  const _v = computed(() => toValue(value));
 
  const allocationTotal = computed(() => _v.value?.allocationTotal || 0);
  const unallocationTotal = computed(() => _v.value?.unallocationTotal || 0);
 
 const table = computed(() => {
  return [
   {
    label: 'Allocated',
    value: allocationTotal.value
   },
   {
    label: 'Unallocated',
    value: unallocationTotal.value
   }
  ];
 });
 return {
    allocationTotal,
    unallocationTotal,
    table
 }
}