import {getAccount, listAccounts} from '@/api/ui/account.js';
import {useAction} from '@/composables/action.js';
import useCollection from '@/composables/collection.js';
import {toAddChange, toRemoveChange, useRoleAssignmentsCollection} from '@/routes/auth/accounts.js';
import {Account} from '@smart-core-os/sc-bos-ui-gen/proto/smartcore/bos/account/v1/account_pb';
import {computed, effectScope, reactive, toValue, watch} from 'vue';

/**
 * @param {import('vue').MaybeRefOrGetter<Partial<ListAccountsRequest.AsObject>>} request
 * @param {import('vue').MaybeRefOrGetter<Partial<ListOnlyCollectionOptions<ListAccountsRequest.AsObject, Account.AsObject>>>?} options
 * @return {UseCollectionResponse<Account.AsObject>}
 */
export function useAccountsCollection(request, options) {
  const normOpts = computed(() => {
    return {
      cmp: (a, b) => a.id.localeCompare(b.id, undefined, {numeric: true}),
      ...toValue(options)
    };
  });
  const client = {
    async listFn(req, tracker) {
      const res = await listAccounts(req, tracker);
      return {
        items: res.accountsList,
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
 * @param {import('vue').MaybeRefOrGetter<Partial<GetAccountRequest.AsObject>>} request
 * @return {ToRefs<UnwrapNestedRefs<UseActionResponse<Account.AsObject>>>}
 */
export function useGetAccount(request) {
  return useAction(request, getAccount);
}

/**
 * @param {Account.Type} accountType
 * @return {string}
 */
export function accountTypeIcon(accountType) {
  switch (accountType) {
    case Account.Type.USER_ACCOUNT:
      return 'mdi-account';
    case Account.Type.SERVICE_ACCOUNT:
      return 'mdi-server';
    default:
      return 'mdi-account-question';
  }
}

/**
 * @param {Account.Type} accountType
 * @return {string}
 */
export function accountTypeStr(accountType) {
  switch (accountType) {
    case Account.Type.USER_ACCOUNT:
      return 'User Account';
    case Account.Type.SERVICE_ACCOUNT:
      return 'Service Account';
    default:
      return `Unknown Account Type ${accountType}`;
  }
}

/**
 * @typedef {UseCollectionResponse<RoleAssignment.AsObject>} AccountRoleAssignmentsResponse
 * @property {function((Array<RoleAssignment.AsObject> | RoleAssignment.AsObject)): void} applyCreate - call after a local change to update the collection
 */

/**
 * Returns a reactive object containing a key for each account in accounts.
 * The values associated with each account are like the return value of useRoleAssignmentsCollection.
 * Each value also has an applyCreate method that can be called to update the collection with a local change,
 * say after assigning new roles to the account.
 *
 * @param {import('vue').MaybeRefOrGetter<Array<string|Account.AsObject>>} accounts
 * @param {import('vue').MaybeRefOrGetter<null | Partial<ListRoleAssignmentsRequest.AsObject>>} [baseRequest]
 * @return {import('vue').Reactive<Record<string, AccountRoleAssignmentsResponse>>}
 */
export function useAccountsRoleAssignments(accounts, baseRequest) {
  const dict = reactive(/** @type {Record<string, AccountRoleAssignmentsResponse>} */ {}); // key is accountId
  const closers = /** @type {Record<string, function():void>} */ {}; // key is accountId

  watch(() => toValue(accounts), (accounts) => {
    const toDelete = new Set(Object.keys(closers));
    const toAdd = new Set(); // of accountIds
    for (const account of accounts) {
      const accountId = (() => {
        if (typeof account === 'string') return account;
        return account.id;
      })();
      if (closers[accountId]) {
        toDelete.delete(accountId);
      } else {
        toAdd.add(accountId);
      }
    }

    for (const accountId of toDelete) {
      closers[accountId]();
      delete closers[accountId];
      delete dict[accountId];
    }

    for (const accountId of toAdd) {
      const scope = effectScope();
      closers[accountId] = () => scope.stop();
      scope.run(() => {
        const _request = {...(toValue(baseRequest) || {})};
        _request.filter = `account_id=${accountId}`;
        /** @type {ResourceCollection<RoleAssignment.AsObject>} */
        let pullResource = null; // filled by callback
        dict[accountId] = {
          ...useRoleAssignmentsCollection(_request, {
            pullFn: (req, resource) => {
              pullResource = resource;
            }
          }),
          applyCreate(changes) {
            if (!pullResource) return; // too early
            if (!Array.isArray(changes)) changes = [changes];
            for (const change of changes) {
              pullResource.lastResponse = toAddChange(change);
            }
          },
          applyRemove(changes) {
            if (!pullResource) return; // too early
            if (!Array.isArray(changes)) changes = [changes];
            for (const change of changes) {
              pullResource.lastResponse = toRemoveChange(change);
            }
          }
        };
      })
    }
  }, {deep: true, immediate: true});

  return dict
}
