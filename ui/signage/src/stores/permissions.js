import {acceptHMRUpdate, defineStore} from 'pinia';
import {computed, ref} from 'vue';

// todo: stop using a fixed permissions list
/**
 * @type {import('@smart-core-os/sc-bos-ui-gen/proto/account_pb').Permission.AsObject[]}
 * @private
 */
const _permissions = [
  {
    'id': 'account:read',
    'displayName': 'Account - Read All',
    'description': 'Read-only access to all account details',
    'impliesIdsList': [
      'account:self:read'
    ]
  },
  {
    'id': 'account:self:read',
    'displayName': 'Account - Read Own Account',
    'description': 'Read details of your own account',
    'impliesIdsList': []
  },
  {
    'id': 'account:self:write',
    'displayName': 'Account - Write Own Account',
    'description': 'Update your own account details, including credentials',
    'impliesIdsList': [
      'account:self:read'
    ]
  },
  {
    'id': 'account:write',
    'displayName': 'Account - Write All',
    'description': 'Manage all accounts; Create and assign roles',
    'impliesIdsList': [
      'account:read'
    ]
  },
  {
    'id': 'traits:history:read',
    'displayName': 'Traits - Read History',
    'description': 'Read all historical data from devices',
    'impliesIdsList': []
  },
  {
    'id': 'traits:read',
    'displayName': 'Traits - Read',
    'description': 'Read all live data from devices',
    'impliesIdsList': []
  },
  {
    'id': 'traits:write',
    'displayName': 'Traits - Write',
    'description': 'Send commands to devices and update device state',
    'impliesIdsList': [
      'traits:read'
    ]
  }
];

export const usePermissionsStore = defineStore('permissions', () => {
  const permissionsList = ref(_permissions);
  const permissionsById = computed(() => permissionsList.value.reduce((acc, curr) => {
    acc[curr.id] = curr;
    return acc;
  }, {}));

  /**
   * A map from permission id to a array of all transitively implied permissions.
   *
   * @type {ComputedRef<Record<string,string[]>>}
   */
  const impliedPermissions = computed(() => {
    const implied = {};
    const permById = permissionsById.value;
    for (const perm of permissionsList.value) {
      const idx = {};
      const toAdd = [perm];
      while (toAdd.length) {
        const p = toAdd.pop();
        if (idx[p.id]) {
          continue;
        }
        idx[p.id] = true;
        toAdd.push(...p.impliesIdsList.map((id) => permById[id]).filter((p) => p));
      }
      delete idx[perm.id];
      implied[perm.id] = Object.keys(idx).sort();
    }
    return implied;
  });
  /**
   * A map from permission id to a array of all permissions that imply it.
   *
   * @type {ComputedRef<Record<string, string[]>>}
   */
  const dependentPermissions = computed(() => {
    const dependent = {};
    for (const [dep, implies] of Object.entries(impliedPermissions.value)) {
      if (!dependent[dep]) {
        dependent[dep] = [];
      }
      for (const imply of implies) {
        if (!dependent[imply]) {
          dependent[imply] = [];
        }
        dependent[imply].push(dep);
      }
    }
    return dependent;
  });

  return {
    loading: ref(false),
    loaded: ref(true),
    permissionsList,
    permissionsById(id) {
      return permissionsById.value[id];
    },
    impliedPermissions(id) {
      return impliedPermissions.value[id];
    },
    dependentPermissions(id) {
      return dependentPermissions.value[id];
    }
  };
});

if (import.meta.hot) {
  import.meta.hot.accept(acceptHMRUpdate(usePermissionsStore, import.meta.hot));
}
