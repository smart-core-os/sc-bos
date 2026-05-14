import {roleToPermissions} from '@/assets/roleToPermissions';
import {useAccountStore} from '@/stores/account';
import {useUiConfigStore} from '@/stores/uiConfig.js';
import {storeToRefs} from 'pinia';
import {computed} from 'vue';

/**
 * Initializing the authentication setup
 *
 * @return {{
 * roles: import('vue').ComputedRef<string[]>,
 * accessLevel: (function(string): boolean),
 * hasNoAccess: (function(string): boolean),
 * isLoggedIn: import('vue').ComputedRef<boolean>,
 * blockActions: import('vue').ComputedRef<boolean>,
 * blockSystemEdit: import('vue').ComputedRef<boolean>,
 * }}
 */
export default function() {
  const uiConfig = useUiConfigStore();
  const accountStore = useAccountStore();
  const {isLoggedIn} = storeToRefs(accountStore);

  // Logged in user's roles
  // replacing '-' in a role to be camelCase
  const roles = computed(() => {
    return accountStore.roles.map((role) => role.replace(/-([a-z])/g, (g) => g[1].toUpperCase()));
  });

  // Logged in user's permissions depending on the role
  const rolePermissions = computed(() => {
    return roles.value.map((role) => ({
      role,
      permissions: roleToPermissions[role] || {fullAccess: [], limitedAccess: [], blockedAccess: []}
    }));
  });

  // The user should have 3 access levels: fullAccess, limitedAccess, blockedAccess
  // Depending on the role and role permissions, we are going to check if the user has
  // permission (and access) to certain pages and functionalities
  const accessLevel = (name) => {
    if (!uiConfig.auth.disabled) {
      // Formatting the name to match the main path (e.g. /site or /devices)
      const formattedName = !name.includes('/') ? `/${name}` : name;

      // If the roles or role permissions are not defined, the user has blocked access
      if (roles.value.length === 0 || !rolePermissions.value.length) {
        return {
          fullAccess: false,
          limitedAccess: false,
          blockedAccess: true
        };
      }

      // Match the main path (e.g. /site or /devices) with the role permissions
      return rolePermissions.value.map((rp) => ({
        role: rp.role,
        fullAccess: rp.permissions.fullAccess.includes(formattedName),
        limitedAccess: rp.permissions.limitedAccess.includes(formattedName),
        blockedAccess: rp.permissions.blockedAccess.includes(formattedName)
      }));
    } else {
      return {
        fullAccess: true,
        limitedAccess: false,
        blockedAccess: false
      };
    }
  };

  // Checking if the user has any of the roles
  const hasAnyRole = (...targetRoles) => {
    return targetRoles.some(role => roles.value.includes(role));
  };

  // Checking if the user has no access to certain pages and functionalities
  // depending on multiple roles and role permissions
  const hasNoAccess = (name) => {
    const accessLevels = accessLevel(name);

    if (Array.isArray(accessLevels)) {
      return accessLevels.some(access => access.blockedAccess);
    }

    return accessLevels.blockedAccess;
  };

  // The following roles have access to actions
  // (e.g. edit, delete, light control etc.) - depending on where we use these roles for disabling actions
  const allowActions = computed(() => hasAnyRole('admin', 'superAdmin', 'commissioner', 'operator'));

  // The following roles have access to system edit
  // (e.g. add, edit, delete, restart etc.) - depending on where we use these roles for disabling system edit
  const allowEdits = computed(() => hasAnyRole('admin', 'superAdmin', 'commissioner'));

  // Blocking actions (e.g. edit, delete, light control etc.)
  const blockActions = computed(() => {
    if (!uiConfig.auth.disabled) {
      return !allowActions.value;
    } else return false;
  });

  // Blocking system edit (e.g. add, edit, delete, restart etc.)
  const blockSystemEdit = computed(() => {
    if (!uiConfig.auth.disabled) {
      return !allowEdits.value;
    } else {
      return false;
    }
  });

  return {
    roles,
    accessLevel,
    hasNoAccess,
    isLoggedIn,
    blockActions,
    blockSystemEdit
  };
}
