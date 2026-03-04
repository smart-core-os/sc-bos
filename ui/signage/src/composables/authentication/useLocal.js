import {localLogin} from '@/api/localLogin.js';
import {loadFromBrowserStorage, saveToBrowserStorage} from '@/util/browserStorage';
import {jwtDecode} from 'jwt-decode';
import {ref} from 'vue';

/**
 * Local authentication composable
 * This composable is responsible for handling all local authentication related functionality
 *
 * @return {AuthenticationProvider & {
 *  login: (function(string, string): Promise<AuthenticationDetails>),
 * }}
 */
export default function() {
  /**
   * Load the local authentication details from local storage
   *
   * @type {import('vue').Ref<AuthenticationDetails|null>}
   */
  const existingLocalAuth = ref(null);

  /**
   * Read the local authentication details from local storage.
   *
   * @return {Promise<AuthenticationDetails|null>}
   */
  const readFromStorage = async () => {
    return loadFromBrowserStorage(
        'local',
        'authenticationDetails',
        null
    )[0];
  };

  /**
   * Write the local authentication details to local storage.
   *
   * @param {AuthenticationDetails} details
   * @return {Promise<void>}
   */
  const writeToStorage = async (details) => {
    saveToBrowserStorage('local', 'authenticationDetails', details);
  };

  /**
   * Clear the local storage.
   *
   * @return {Promise<void>}
   */
  const clearStorage = async () => {
    await window.localStorage.removeItem('authenticationDetails');
  };

  /**
   * Initialize local authentication - set the existing local authentication details
   * or the default store values
   *
   * @return {Promise<AuthenticationDetails|null>}
   */
  const initializeLocal = async () => {
    existingLocalAuth.value = await readFromStorage();
    return existingLocalAuth.value;
  };

  /**
   * Login using local authentication provider
   *
   * @param {string} username
   * @param {string} password
   * @return {Promise<AuthenticationDetails>}
   */
  const loginLocal = async (username, password) => {
    try {
      const res = await localLogin(username, password);

      if (res.status === 200) {
        const payload = await res.json();

        if (payload?.access_token) {
          const details = /** @type {AuthenticationDetails} */ {
            claims: {
              email: username,
              ...jwtDecode(payload.access_token)
            },
            loggedIn: true,
            token: payload.access_token
          };
          await writeToStorage(details);
          existingLocalAuth.value = details;
          return details;
        }
      } else {
        existingLocalAuth.value = null;
        const payload = await res.json();
        await clearStorage();
        return Promise.reject(payload);
      }
    } catch {
      existingLocalAuth.value = null;
      await clearStorage();
      return Promise.reject(new Error('Failed to sign in, please try again.'));
    }
  };

  /**
   * Logout using the store reset function, then store the cleared store in local storage
   *
   * @return {Promise<void>}
   */
  const logoutLocal = async () => {
    existingLocalAuth.value = null;
    await clearStorage();
  };

  /**
   * Refresh the local authentication details.
   *
   * @return {Promise<AuthenticationDetails>}
   */
  const refreshToken = async () => {
    return existingLocalAuth.value;
  };

  return {
    existingLocalAuth,
    init: initializeLocal,
    login: loginLocal,
    logout: logoutLocal,
    refreshToken
  };
}
