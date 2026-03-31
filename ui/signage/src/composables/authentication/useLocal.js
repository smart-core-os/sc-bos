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
    if (!existingLocalAuth.value) {
      const username = import.meta.env.VITE_DASHBOARD_USERNAME;
      const password = import.meta.env.VITE_DASHBOARD_PASSWORD;
      if (username && password) {
        existingLocalAuth.value = await loginLocal(username, password).catch(() => console.warn('local login rejected'));
      }
    }
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
        const payload = await res.json();
        return Promise.reject(payload);
      }
    } catch {
      return Promise.reject(new Error('Failed to sign in, please try again.'));
    }
  };

  /**
   * LogoutLocal will log in silently to prevent signage players from needing any interaction to maintain a valid token.
   *
   * @return {Promise<AuthenticationDetails>}
   */
  const logoutLocal = async () => {
    existingLocalAuth.value = null;
    await clearStorage();
    const username = import.meta.env.VITE_DASHBOARD_USERNAME;
    const password = import.meta.env.VITE_DASHBOARD_PASSWORD;
    if (username && password) {
      return await loginLocal(username, password).catch(() => console.warn('local login rejected'));
    }
    console.warn('no credentials provided for local login, unable to maintain authentication');
    return null;
  };

  /**
   * Refresh the local authentication details.
   * If the stored token has expired, attempts a silent re-login using env var credentials.
   *
   * @return {Promise<AuthenticationDetails>}
   */
  const refreshToken = async () => {
    let needsLogin;

    try {
      needsLogin = !existingLocalAuth.value ||
        jwtDecode(existingLocalAuth.value.token).exp * 1000 < Date.now();
    } catch (e) {
      console.warn('Failed to decode token, will attempt to re-login', e);
      needsLogin = true;
    }
    if (needsLogin) {
      const username = import.meta.env.VITE_DASHBOARD_USERNAME;
      const password = import.meta.env.VITE_DASHBOARD_PASSWORD;
      if (username && password) {
        return loginLocal(username, password).catch(() => existingLocalAuth.value);
      }
    }
    return existingLocalAuth.value;
  };

  return {
    existingLocalAuth,
    hasCredentials: !!(import.meta.env.VITE_DASHBOARD_USERNAME && import.meta.env.VITE_DASHBOARD_PASSWORD),
    init: initializeLocal,
    login: loginLocal,
    logout: logoutLocal,
    refreshToken
  };
}
