import {closeResource, newResourceCollection} from '@/api/resource.js';
import {useAction} from '@/composables/action.js';
import {cap} from '@/util/number.js';
import {watchResource} from '@/util/traits.js';
import {isNullOrUndef} from '@/util/types.js';
import deepEqual from 'fast-deep-equal';
import {computed, onScopeDispose, reactive, ref, toValue, watch} from 'vue';

/**
 * @typedef {Object} ListResponse
 * @template T
 * @property {T[]} items
 * @property {string=} nextPageToken
 * @property {number=} totalSize
 */
/**
 * @typedef {Object} PageRequest
 * @property {number=} pageSize
 * @property {string=} pageToken
 */
/**
 * @typedef {Object} PullChange
 * @template T
 * @property {T?} oldValue
 * @property {T?} newValue
 */
/**
 * @typedef {Object} UseCollectionOptions
 * @template T
 * @property {number=} wantCount - how many items to fetch from the server, -1 for all
 * @property {number=} pageSize - how many items to fetch per request, defaults to cap(missing, 10, 500)
 * @property {boolean=} paused - suspend requests
 * @property {(item: T) => string=} idFn - a function to get the id of an item, defaults to item.id or item.name
 * @property {(item: T) => boolean=} filterFn - a function that filters items from the list.
 * @property {(item: T) => T} transform - a function that transforms items before they are added to the list.
 *   Useful for converting proto Timestamps.
 * @property {(a: T, b: T) => number} cmp - a function that compares two items. Any new item we receive from the
 *   subscription will not be added unless this is specified, as we don't know where to put it in the list.
 */
/**
 * @typedef {Object} UseCollectionResponse
 * @template T
 * @property {Ref<T[]>} items
 * @property {Ref<number>} totalItems
 * @property {Ref<boolean>} hasServerTotalItems
 * @property {Ref<boolean>} hasMorePages
 * @property {Ref<boolean>} loading
 * @property {Ref<boolean>} loadingNextPage
 * @property {Ref<ResourceError[]>} errors
 * @property {function(): void} refresh
 */

/**
 * Executes a list query and keeps the results up to date based on a pull.
 *
 * @template T, R
 * @param {MaybeRefOrGetter<R>} request - the request to use for the collection. Passed to both listFn and pullFn,
 *   for listFn it may have pageSize or pageToken set.
 * @param {{
 *   listFn: (req: R & PageRequest, tracker: ActionTracker<any>) => Promise<ListResponse<T>>,
 *   pullFn?: (req: R, resource: ResourceCollection<T, any>) => void
 * }} client - functions for listing and subscribing to changes based on a request
 * @param {MaybeRefOrGetter<UseCollectionOptions<T>>} [options] - optional options
 * @return {UseCollectionResponse<T>}
 */
export default function useCollection(request, client, options) {
  // The final items of the collection, combining listed and pulled items.
  const items = ref(/** @type {T[]} */ []);

  // used to fetch the next page when needed
  const lastListResponse = ref(/** @type {ListResponse<T>} */ null);

  // changes to items that have yet to be applied.
  // Populated by both list and pull.
  // Processed and cleared by processChanges.
  const unprocessedChanges = ref(/** @type {PullChange<T>[]} */ []);

  // related to calls to client.pullFn
  const pullResource = reactive(/** @type {ResourceCollection<T, any>} */ newResourceCollection());
  watch(() => pullResource.lastResponse, (r) => {
    // most pull responses look like {changesList: [{oldValue, newValue}]}
    if (r && typeof r.toObject === 'function' && typeof r.getChangesList === 'function') {
      unprocessedChanges.value.push(...r.getChangesList().map(change => change.toObject()));
      pullCountDelta.value += processChanges();
    }
  }, {flush: 'sync'});

  const targetListCount = computed(() => toValue(options)?.wantCount ?? 20);
  // Are there (likely) more pages available on the server?
  // If we've never asked, or the server says there are more pages, then we return true.
  const hasMorePages = computed(() => !lastListResponse.value || !!lastListResponse.value.nextPageToken);
  const shouldFetch = computed(() => {
    if (!toValue(request)) return false; // don't fetch if there isn't a request
    if (toValue(options)?.paused ?? false) return false; // don't fetch if paused
    if (!hasMorePages.value) return false; // don't fetch if there are no more pages
    // otherwise, fetch if we haven't fetched enough items
    return targetListCount.value === -1 || items.value.length < targetListCount.value;
  });
  // A guess at how many total items there are, either from the server or calculated locally based on fetched items.
  const totalItems = computed(() => {
    if (toValue(options)?.filterFn) return items.value.length; // can't use server total if we filter items locally
    const serverTotalSize = lastListResponse.value?.totalSize;
    if (isNullOrUndef(serverTotalSize)) return items.value.length;
    return lastListResponse.value?.totalSize + pullCountDelta.value;
  });
  // A positive or negative number indicating how many items we added/removed as a result of pull changes.
  const pullCountDelta = ref(0);
  // Is totalItems a value returned by the server or calculated locally.
  const hasServerTotalItems = computed(() => Boolean(lastListResponse.value?.totalSize));

  // pull fetching
  const pullRequest = computed(() => {
    const req = toValue(request);
    if (!req) return null; // don't fetch if we shouldn't
    const _req = {...req};
    _req.updatesOnly = true; // list will get the existing values
    return _req;
  });
  const stopPull = () => closeResource(pullResource);
  watchResource(
      pullRequest,
      () => toValue(options)?.paused,
      (req) => {
        client.pullFn(req, pullResource);
        return () => stopPull();
      }
  );
  onScopeDispose(() => stopPull());

  // list fetching
  const listRequest = computed(() => {
    if (!shouldFetch.value) return null; // don't fetch if we shouldn't
    const _req = {...toValue(request)};
    if (lastListResponse.value) _req.pageToken = lastListResponse.value.nextPageToken;
    _req.pageSize = _req.pageSize ??
        toValue(options)?.pageSize ??
        cap(targetListCount.value - items.value.length, 10, 500);
    return _req;
  });
  const {refresh: refreshList, ...listTrackerRefs} = useAction(listRequest, async (req, tracker) => {
    try {
      const pageResponse = await client.listFn(req, tracker);
      lastListResponse.value = pageResponse;
      unprocessedChanges.value.push(...pageResponse.items.map(v => ({newValue: v})));
      processChanges();
    } catch (e) {
      // todo: add options to not log the error because the caller is handling it
      console.warn(e);
    }
  });
  const listTracker = reactive(listTrackerRefs);

  // reset our item list when the request changes
  watch(() => toValue(request), (o, n) => {
    if (deepEqual(o, n)) return; // no change
    items.value = [];
    lastListResponse.value = null;
    // the change in request will also cause the pull watcher to trigger,
    // but there's no way in there to know if we were paused or the request changed.
    unprocessedChanges.value = [];
  }, {deep: true});

  // aggregate status of our requests
  const loading = computed(() => listTracker.loading || pullResource.loading);
  const loadingNextPage = computed(() => listTracker.loading);
  const errors = computed(() => [listTracker.error, pullResource.streamError]
      .filter(e => e));

  /**
   * Force the collection to refresh, clearing all items and listing items again from page 1.
   */
  function refresh() {
    items.value = [];
    lastListResponse.value = null;
    unprocessedChanges.value = [];
    refreshList();
  }

  /**
   * Applies filterFn to changes, adjusting their type accordingly.
   *
   * - If a change is an add but filterFn returns false, it is omitted.
   * - If a change is a delete and filterFn returns false, it is omitted.
   * - If a change is an update
   *   - If filterFn returns false for both old and new values, it is omitted.
   *   - If filterFn returns false for the old value but true for the new value, it is converted to an add.
   *   - If filterFn returns true for the old value but false for the new value, it is converted to a delete.
   * - All other changes are kept as is.
   *
   * @param {PullChange<T>[]} changes
   * @param {(T) => boolean} filterFn
   * @return {PullChange<T>[]}
   */
  const filterChanges = (changes, filterFn) => {
    if (!filterFn) return changes; // no filter, return as is
    return changes.map(change => {
      if (change.newValue && !change.oldValue) {
        // add
        if (filterFn(change.newValue)) return change; // keep
        return null; // omit
      } else if (!change.newValue && change.oldValue) {
        // delete
        if (filterFn(change.oldValue)) return change; // keep
        return null; // omit
      } else if (change.newValue && change.oldValue) {
        // update
        const oldValid = filterFn(change.oldValue);
        const newValid = filterFn(change.newValue);
        if (oldValid && newValid) return change; // keep as is
        if (!oldValid && newValid) {
          // convert to add
          return {newValue: change.newValue, oldValue: null};
        }
        if (oldValid && !newValid) {
          // convert to delete
          return {newValue: null, oldValue: change.oldValue};
        }
        return null; // omit both
      }
      return change; // keep as is, should not happen
    }).filter(v => v !== null); // remove nulls
  }

  /**
   * Removes duplicate changes from the list, returning the deduplicated changes.
   * Items are compared based on the idFn of the options.
   * Changes later in the list override earlier changes.
   *
   * The rules are as follows:
   * - If two adds for the same item are found, only the second is kept.
   * - If an add is followed by an update, the update is converted to an add and kept.
   * - Two updates keep the second update.
   * - Anything followed by a delete will become a delete.
   *
   * @param {PullChange<T>[]} changes
   * @param {(T) => string} idFn
   * @return {PullChange<T>[]}
   */
  const removeDuplicateChanges = (changes, idFn) => {
    const seen = /** @type {Map<string, {indexes: number[], change: PullChange<T>}>} */ new Map();
    for (let i = 0; i < changes.length; i++) {
      const change = changes[i];
      const id = getId(change.newValue ?? change.oldValue, idFn);
      if (!id) continue; // no id, skip
      const existing = seen.get(id);
      if (!existing) {
        seen.set(id, {indexes: [i], change});
        continue;
      }

      // process duplicates
      // a=add, u=update, d=delete
      // o=old, n=new
      // o1=set oldValue to existing, nd=set newValue to null
      //
      //    1 | existing               |
      // 2    |  a     |  u    |  d    |
      // -----+--------+-------+-------+
      // n  a | n2     | n2    | n2 od |
      // e  u | od n2  | n2    | n2    |
      // w  d | o2 nd  | o2 nd |       |
      const isAdd = change.newValue && !change.oldValue;
      const isUpdate = change.newValue && change.oldValue;
      const wasAdd = existing.change.newValue && !existing.change.oldValue;
      const wasDelete = !existing.change.newValue && existing.change.oldValue;
      if (isAdd || isUpdate) {
        existing.change.newValue = change.newValue;
        if (wasDelete && isAdd) existing.change.oldValue = null;
        if (wasAdd && isUpdate) existing.change.oldValue = null;
      } else {
        // isDelete
        existing.change.oldValue = change.oldValue;
        existing.change.newValue = null;
      }
      existing.indexes.push(i);
    }

    const newChanges = [];
    const changesSortedByLowestIndex = Array.from(seen.values()).sort((a, b) => a.indexes[0] - b.indexes[0]);
    for (const {change} of changesSortedByLowestIndex) {
      newChanges.push(change);
    }
    return newChanges;
  };

  /**
   * Processes changes we received from a pull against those we received from a list.
   * unprocessedChanges will be empty after this call.
   *
   * If options.cmp is not defined, items created via pull will not be added to the list.
   *
   * @return {number} delta of items added/removed from the items list
   */
  function processChanges() {
    const _items = items.value;
    const _changes = unprocessedChanges.value;
    if (!_changes.length) return 0; // no changes, nothing to do
    const opts = toValue(options);
    const transform = opts?.transform ?? (v => v);
    const filterFn = opts?.filterFn;
    const idFn = opts?.idFn;

    // optimise the lookup of items by id if we're going to do it a bunch
    let getIndex = (id) => _items.findIndex(v => getId(v, idFn) === id);
    if (_changes.length > 10) {
      // compute an index for the items
      const index = new Map();
      for (let i = 0; i < _items.length; i++) {
        index.set(getId(_items[i], idFn), i);
      }
      getIndex = (id) => {
        if (index.has(id)) return index.get(id);
        return -1;
      };
    }

    // delay mutating the items until later so the indexes remain accurate
    const toDeleteIndexes = [];
    const toAddItems = [];
    let changes = unprocessedChanges.value;
    changes = filterChanges(changes, filterFn);
    changes = removeDuplicateChanges(changes, idFn);
    for (const change of changes) {
      const index = getIndex(getId(change.newValue ?? change.oldValue, idFn));
      if (change.oldValue && !change.newValue) {
        // deletion
        if (index !== -1) toDeleteIndexes.push(index);
      } else if (change.newValue) {
        if (index !== -1) {
          // update
          _items[index] = transform(change.newValue);
        } else {
          // add
          toAddItems.push(transform(change.newValue));
        }
      }
    }

    let countDelta = -toDeleteIndexes.length;

    // process deletes first
    toDeleteIndexes.sort((a, b) => b - a); // sort in reverse order
    for (const i of toDeleteIndexes) {
      _items.splice(i, 1);
    }

    // only insert if options.cmp is defined, then insert in the correct place
    const cmp = toValue(options)?.cmp;
    if (cmp) {
      countDelta += toAddItems.length;
      for (const item of toAddItems) {
        let i = 0;
        // todo: replace with a binary search
        while (i < _items.length && cmp(item, _items[i]) > 0) i++;
        _items.splice(i, 0, item);
      }
    } else {
      // don't actually add the items as we'd have no way to know where to put them
    }

    items.value = _items;
    unprocessedChanges.value = [];
    return countDelta;
  }

  return {
    items,
    totalItems,
    hasServerTotalItems,
    hasMorePages,

    loading,
    loadingNextPage,
    errors,

    refresh,

    _listTracker: listTracker,
    _pullResource: pullResource
  };
}

/**
 * Get the id for the value. Uses idFn or .id or .name or returns null.
 *
 * @template T
 * @param {T} v
 * @param {(T) => string=} idFn
 * @return {string|null}
 */
function getId(v, idFn) {
  if (idFn) return idFn(v);
  return v?.id ?? v?.name ?? null;
}
