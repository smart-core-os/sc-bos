<template>
  <v-card elevation="0" class="rounded-lg">
    <v-toolbar title="Users" color="transparent" class="px-4 pt-2">
      <template #append>
        <delete-accounts-btn
            v-if="showDeleteAccountsBtn"
            color="error"
            variant="outlined"
            :accounts="selectedAccounts"
            @delete="onDelete"/>
        <grant-role-btn
            v-if="showGrantBtn"
            variant="outlined"
            :accounts="allSelectedAccountIds"
            global-only
            @save="onGrantSave"/>
        <new-account-btn variant="flat"
                         color="primary"
                         :account-types="[Account.Type.USER_ACCOUNT]"
                         @save="onNewAccountSave"/>
      </template>
    </v-toolbar>
    <v-expand-transition>
      <div v-if="latestServiceAccount">
        <copy-secret-alert :credential="latestServiceAccount.clientSecret" @close="onSecretClose"/>
      </div>
    </v-expand-transition>
    <v-card-text>
      <v-data-table-server
          v-bind="omit(tableAttrs, 'items')"
          :items="tableItemsWithRoleAssignments"
          :headers="tableHeaders"
          disable-sort
          return-object
          v-model="selectedAccounts"
          :row-props="tableRowProps"
          @click:row="onRowClick">
        <template #top>
          <v-expand-transition>
            <div v-if="tableErrorStr">
              <v-alert type="error" :text="tableErrorStr"/>
            </div>
          </v-expand-transition>
        </template>
        <template #item.type="{item, internalItem, isSelected, toggleSelect}">
          <div class="select--container" :class="{selected: isSelected(internalItem)}">
            <v-checkbox-btn :model-value="isSelected(internalItem)"
                            @click="toggleSelect(internalItem)"
                            color="primary"/>
            <v-icon :icon="accountTypeIcon(item.type)" v-tooltip:bottom="accountTypeStr(item.type)"/>
          </div>
        </template>
        <template #item.displayName="{item}">
          <span>{{ item.displayName }}</span>
          <span class="text-medium-emphasis ml-2" v-if="item.description">{{ item.description }}</span>
        </template>
        <template #item.username="{item}">
          {{ item.userDetails?.username ?? item.serviceDetails?.clientId }}
        </template>
        <template #item.createTime="{item}">
          {{ timestampToDate(item.createTime).toLocaleDateString() }}
        </template>
        <template #item.roles="{item}">
          <v-progress-circular indeterminate v-if="item.roles?.loading" size="small"/>
          <template v-else>
            <span v-for="(role, i) in item.roles?.items.slice(0, 1)" :key="role.id">
              <role-assignment-link :role-assignment="role" no-nav data-skip-row-select="true"/>
              <template v-if="i < item.roles?.items.length - 1">, </template>
            </span>
            <template v-if="item.roles.items.length > 1">
              and {{ item.roles.items.length - 1 }} more
            </template>
          </template>
        </template>
      </v-data-table-server>
    </v-card-text>
  </v-card>
</template>

<script setup>
import {timestampToDate} from '@/api/convpb.js';
import {deleteRoleAssignment, rotateAccountClientSecret, updateAccount} from '@/api/ui/account.js';
import {useDevicesCollection} from '@/composables/devices.js';
import {useDataTableCollection} from '@/composables/table.js';
import {toAddChange, toRemoveChange, toUpdateChange} from '@/routes/auth/accounts.js';
import {
  accountTypeIcon,
  accountTypeStr,
  useAccountsCollection,
  useAccountsRoleAssignments,
  useGetAccount
} from '@/routes/auth/accounts/accounts.js';
import CopySecretAlert from '@/routes/auth/accounts/CopySecretAlert.vue';
import DeleteAccountsBtn from '@/routes/auth/accounts/DeleteAccountsBtn.vue';
import GrantRoleBtn from '@/routes/auth/accounts/GrantRoleBtn.vue';
import NewAccountBtn from '@/routes/auth/accounts/NewAccountBtn.vue';
import RoleAssignmentLink from '@/routes/auth/accounts/RoleAssignmentLink.vue';
import {useGetRoles} from '@/routes/auth/roles/roles.js';
import {useSidebarStore} from '@/stores/sidebar.js';
import {Account, RoleAssignment} from '@smart-core-os/sc-bos-ui-gen/proto/smartcore/bos/account/v1/account_pb';
import {omit} from 'lodash';
import {computed, ref, toValue, watch} from 'vue';
import {useRouter} from 'vue-router';

const props = defineProps({
  accountId: {
    type: String,
    default: null,
  }
});

// used to fake PullAccounts when creating a new account.
// the var isn't reactive, but the value will be whenever it's set
let pullAccountsResource = /** @type {ResourceCollection<Account.AsObject, *> | null} */ null;

const wantCount = ref(20);
const accountsCollectionOpts = computed(() => {
  return {
    wantCount: wantCount.value,
    pullFn: (_, resource) => {
      pullAccountsResource = resource;
    },
    filterFn: (item) => item.type === Account.Type.USER_ACCOUNT
  };
})
const accountsCollection = useAccountsCollection({}, accountsCollectionOpts);

const tableAttrs = useDataTableCollection(wantCount, accountsCollection);
const tableHeaders = computed(() => {
  return [
    {key: 'type', width: '1.5rem', cellProps: {class: 'pr-0'}},
    {title: 'Name', key: 'displayName', maxWidth: '10em', cellProps: {class: 'text-overflow-ellipsis'}},
    {title: 'Username', key: 'username', width: '20em', cellProps: {class: 'text-overflow-ellipsis'}},
    {title: 'Created', key: 'createTime', width: '8em'},
    {title: 'Roles', key: 'roles', maxWidth: '10em', cellProps: {class: 'text-overflow-ellipsis'}},
  ]
});

// additional info we want to inline in the table, instead of showing ids
const accountsOnCurrentPage = computed(() => tableAttrs.items);
const accountsToGetRoleAssignments = computed(() => {
  if (props.accountId) {
    return [...accountsOnCurrentPage.value, props.accountId];
  }
  return accountsOnCurrentPage.value;
})
const accountsRoleAssignments = useAccountsRoleAssignments(accountsToGetRoleAssignments);
const allRoleIds = computed(() => Object.values(accountsRoleAssignments).map((r) => r.items ?? []).flat().map((r) => r.roleId));
const rolesById = useGetRoles(null, allRoleIds);
const isNamedAssignment = (r) => {
  return r.scope?.resourceType === RoleAssignment.ResourceType.NAMED_RESOURCE ||
      r.scope?.resourceType === RoleAssignment.ResourceType.NAMED_RESOURCE_PATH_PREFIX
}
const allNamedResources = computed(() => Object.values(accountsRoleAssignments).map((r) => r.items ?? []).flat()
    .filter((r) => isNamedAssignment(r))
    .map((r) => r.scope?.resource));
const {items: nameResourceDevices} = useDevicesCollection(() => {
  return {query: {conditionsList: [{field: 'name', stringIn: {stringsList: allNamedResources.value}}]}};
});
const namedResourceDevicesByName = computed(() => {
  const devices = {};
  for (const device of nameResourceDevices.value) {
    devices[device.name] = device;
  }
  return devices;
});

/**
 * @param {import('vue').Reactive<AccountRoleAssignmentsResponse>} roleAssignments
 * @return {import('vue').Reactive<AccountRoleAssignmentsResponse & {items: Array<RoleAssgnment.AsObject & {role: Role.AsObject}>}>}
 */
const hydrateRoleAssignments = (roleAssignments) => {
  const _devicesByName = namedResourceDevicesByName.value;
  const assignments = (roleAssignments?.items ?? [])
      .map((assignment) => {
        const role = rolesById[assignment.roleId];
        if (!role) return assignment;
        const device = (() => {
          if (isNamedAssignment(assignment)) {
            return _devicesByName[assignment.scope?.resource];
          }
          return null;
        })();
        return {
          ...assignment,
          role: role.response, _role: role,
          device,
        };
      });
  const loading = computed(() => roleAssignments.loading || assignments.some((a) => toValue(a._role?.loading)));
  return {...roleAssignments, items: assignments, loading};
}
const hydratedAccountsRoleAssignments = computed(() => {
  const hydrated = {};
  for (const [id, roleAssignments] of Object.entries(accountsRoleAssignments)) {
    hydrated[id] = hydrateRoleAssignments(roleAssignments);
  }
  return hydrated;
});

const tableItemsWithRoleAssignments = computed(() => {
  const accounts = accountsOnCurrentPage.value;
  const roleAssignmentsByAccount = hydratedAccountsRoleAssignments.value;
  for (const account of accounts) {
    account.roles = roleAssignmentsByAccount[account.id];
  }
  return accounts;
});

const tableRowProps = ({item}) => {
  return {
    class: {
      'row-selected': selectedAccounts.value.includes(item),
      'row-active': item.id === props.accountId,
    }
  };
};
const tableErrorStr = computed(() => {
  const errors = accountsCollection.errors.value;
  if (errors.length === 0) return null;
  return 'Error fetching accounts: ' + errors.map((e) => (e.error ?? e).message ?? e).join(', ');
});

const router = useRouter();
const onRowClick = (e, {item}) => {
  if (item.id === props.accountId) return; // don't click on the same item
  if (e.target.closest('[data-skip-row-select]')) return; // something else is handling the click
  router.push({name: 'users', params: {accountId: item.id}});
}
const sidebar = useSidebarStore();
const {response: sidebarAccount, refresh: refreshSidebarAccount} = useGetAccount(() => {
  if (!props.accountId) return null;
  return {id: props.accountId};
});
const sidebarRoleAssignments = computed(() => {
  if (!sidebarAccount.value) return null;
  return hydratedAccountsRoleAssignments.value[sidebarAccount.value.id];
})
watch(sidebarAccount, (item) => {
  if (!item) {
    sidebar.closeSidebar();
    return;
  }
  sidebar.title = item.displayName || `Account ${props.roleId}`;
  sidebar.data = {
    account: item,
    roleAssignments: sidebarRoleAssignments,
    updateAccount: onAccountUpdate,
    rotateServiceAccountSecret: onRotateServiceAccountSecret,
    removeRole: onGrantRemove
  };
  sidebar.visible = true;
}, {immediate: true});

const latestAccount = ref(null);
const latestServiceAccount = computed(() => latestAccount.value?.serviceDetails);

const onNewAccountSave = ({account}) => {
  if (pullAccountsResource) {
    pullAccountsResource.lastResponse = toAddChange(account);
  }
  latestAccount.value = account;
};
const onAccountUpdate = async ({account}) => {
  const oldAccount = (() => {
    if (sidebarAccount.value?.id === account.id) return sidebarAccount.value;
    return accountsCollection.items.value.find((r) => r.id === account.id);
  })();

  const newAccount = await updateAccount({account})

  if (!oldAccount) {
    // we can't do a dynamic of the account, so just refresh the whole list
    accountsCollection.refresh();
  } else {
    pullAccountsResource.lastResponse = toUpdateChange(oldAccount, newAccount);
  }
  if (latestAccount.value?.id === account.id) {
    latestAccount.value = newAccount;
  }
  selectedAccounts.value = selectedAccounts.value.map((r) => {
    if (r.id === account.id) return newAccount;
    return r;
  });
  refreshSidebarAccount();

  return newAccount;
}

const onRotateServiceAccountSecret = async ({account, expireTime}) => {
  const res = await rotateAccountClientSecret({id: account.id, previousSecretExpireTime: expireTime});
  account.serviceDetails.clientSecret = res.clientSecret;
  latestAccount.value = account;
  refreshSidebarAccount();
}
const onSecretClose = () => {
  latestAccount.value = null;
};

const selectedAccounts = ref([]);
const allSelectedAccountIds = computed(() => {
  const ids = {};
  for (const account of selectedAccounts.value) {
    ids[account.id] = true;
  }
  if (props.accountId) {
    ids[props.accountId] = true;
  }
  return Object.keys(ids).sort();
})

const showGrantBtn = computed(() => allSelectedAccountIds.value.length > 0);
const onGrantSave = (ras) => {
  for (const ra of ras) {
    const col = accountsRoleAssignments[ra.accountId];
    if (!col) continue;
    col.applyCreate(ra);
  }
};
const onGrantRemove = async (ra) => {
  await deleteRoleAssignment({id: ra.id, allowMissing: true});
  const col = accountsRoleAssignments[ra.accountId];
  if (!col) return;
  col.applyRemove(ra);
}

const showDeleteAccountsBtn = computed(() => selectedAccounts.value.length > 0);
const onDelete = () => {
  if (pullAccountsResource) {
    for (const account of selectedAccounts.value) {
      pullAccountsResource.lastResponse = toRemoveChange(account);
    }
  }
  const _latest = latestAccount.value;
  for (const account of selectedAccounts.value) {
    if (_latest && account.id === _latest.id) {
      latestAccount.value = null;
    }
    if (account.id === props.accountId) {
      router.push({name: 'users'});
    }
  }
  selectedAccounts.value = [];
}
</script>

<style scoped lang="scss">
:deep(.v-toolbar__append) {
  gap: 1rem;
}

.select--container {
  display: grid;
  justify-items: center;
  align-items: center;

  .v-checkbox-btn {
    grid-column: 1 / -1;
    grid-row: 1 / -1;
  }

  .v-icon {
    grid-column: 1 / -1;
    grid-row: 1 / -1;
  }

  &:not(:hover, .selected) {
    .v-checkbox-btn {
      visibility: hidden;
    }
  }

  &:hover, &.selected {
    .v-icon {
      visibility: hidden;
    }
  }
}

.v-data-table {
  :deep(.row-selected) {
    background-color: rgba(var(--v-theme-primary), 0.1);
  }

  :deep(.row-active) {
    background-color: rgba(var(--v-theme-primary), 0.4);
  }
}
</style>