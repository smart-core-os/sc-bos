import {listRoleAssignments} from '@/api/ui/account.js';
import useCollection from '@/composables/collection.js';
import {ChangeType} from '@smart-core-os/sc-bos-ui-gen/proto/smartcore/bos/types/v1/change_pb';
import {computed, toValue} from 'vue';

/**
 * @typedef {UseCollectionOptions<T>} ListOnlyCollectionOptions
 * @property {function(R, ResourceCollection<T, any>): void} pullFn
 * @template R
 * @template T
 */

/**
 * @param {import('vue').MaybeRefOrGetter<Partial<ListRoleAssignmentsRequest.AsObject>>} request
 * @param {import('vue').MaybeRefOrGetter<Partial<ListOnlyCollectionOptions<ListRoleAssignmentsRequest.AsObject, RoleAssignment.AsObject>>>} options
 * @return {UseCollectionResponse<RoleAssignment.AsObject>}
 */
export function useRoleAssignmentsCollection(request, options) {
  const normOpts = computed(() => {
    return {
      cmp: (a, b) => a.id.localeCompare(b.id, undefined, {numeric: true}),
      ...toValue(options)
    };
  });
  const client = {
    async listFn(req, tracker) {
      const res = await listRoleAssignments(req, tracker);
      return {
        items: res.roleAssignmentsList,
        nextPageToken: res.nextPageToken,
        totalSize: res.totalSize
      };
    },
    pullFn(req, resource) {
      const opts = toValue(normOpts);
      if (opts.pullFn) {
        opts.pullFn(req, resource);
      }
    }
  };

  return useCollection(request, client, normOpts);
}

/**
 * Returns an object that looks like a Pull Change that adds the given val.
 * Useful when an API doesn't support Pull but you want to reuse utilities that do.
 *
 * @param {T} val
 * @return {Object}
 * @template T
 */
export function toAddChange(val) {
  const changes = {
    changesList: [{
      type: ChangeType.ADD,
      newValue: val,
    }]
  };
  return {
    toObject() {
      return changes;
    },
    getChangesList() {
      return changes.changesList.map(change => ({toObject() { return change; }}));
    }
  }
}

/**
 * Returns an object that looks like a Pull Change that deletes the given val.
 * Useful when an API doesn't support Pull but you want to reuse utilities that do.
 *
 * @param {T} val
 * @return {Object}
 * @template T
 */
export function toRemoveChange(val) {
  const changes = {
    changesList: [{
      type: ChangeType.REMOVE,
      oldValue: val,
    }]
  };
  return {
    toObject() {
      return changes;
    },
    getChangesList() {
      return changes.changesList.map(change => ({toObject() { return change; }}));
    }
  }
}

/**
 * Returns an object that looks like a Pull Change that updates the val.
 * Useful when an API doesn't support Pull but you want to reuse utilities that do.
 *
 * @param {T} oldVal
 * @param {T} newVal
 * @return {Object}
 * @template T
 */
export function toUpdateChange(oldVal, newVal) {
  const changes = {
    changesList: [{
      type: ChangeType.UPDATE,
      oldValue: oldVal,
      newValue: newVal,
    }]
  };
  return {
    toObject() {
      return changes;
    },
    getChangesList() {
      return changes.changesList.map(change => ({toObject() { return change; }}));
    }
  }
}
