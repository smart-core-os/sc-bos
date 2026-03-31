import {useUiConfigStore} from '@/stores/uiConfig.js';
import {loadFromBrowserStorage, saveToBrowserStorage} from '@/util/browserStorage';
import {jwtDecode} from 'jwt-decode';
import {computed, ref, toValue} from 'vue';


/**
 * @typedef TokenResponse
 * @property {string} access_token
 * @property {string} token_type
 * @property {number} expires_in Seconds since the token was issued
 * @property {number} refresh_expires_in Seconds since the token was issued
 * @property {string} refresh_token
 * @property {string} scope
 * @property {number} timestamp Date.now() when the token was retrieved
 */

/**
 * Code is the data we retrieve from the server to show to the user to complete the authentication.
 *
 * @typedef CodeResponse
 * @property {string} device_code A temporary unique code for the device that initiated the flow, pass this code to
 *  checkDeviceToken to see if the user has completed the flow.
 * @property {string} user_code The code the user needs to enter.
 * @property {string} verification_uri A url the user can visit to enter the user_code, can be displayed to the user
 *  as is.
 * @property {string} verification_uri_complete A url that auto fills the user_code, it typically too long to type,
 *  encode as QR or similar image for scanning.
 * @property {number} expires_in Seconds until device_code and user_code expire
 * @property {number} interval In seconds, recommended polling interval for checkDeviceToken
 * @property {number} timestamp Date.now() when the code was retrieved
 */

/**
 * Context contains the information to show to the user to complete the authentication.
 * The {@code complete} promise will be settled when the user has completed the authentication, or
 * {@code code.expires_in} has passed.
 *
 * @typedef Context
 * @property {CodeResponse} code The data to show to the user to complete the authentication.
 * @property {Promise<AuthenticationDetails>} complete Settled when the user has completed the authentication, or
 *  the flow has expired or been aborted.
 * @property {function(): void} cancel Call to stop checking for the user to complete the authentication.
 */
/**
 * @typedef Config
 * @property {MaybeRefOrGetter<string>} clientId
 * @property {MaybeRefOrGetter<string>} deviceUrl
 * @property {MaybeRefOrGetter<string>} tokenUrl
 */

/**
 * Implements AuthenticationProvider using the OAuth2 Device Flow.
 *
 * @param {MaybeRefOrGetter<Config>} config
 * @return {AuthenticationProvider & {
 *  begin: (function(string[]?): Promise<Context>),
 * }}
 */
export default function useDeviceFlow(config) {
  const clientId = computed(() => toValue(toValue(config)?.clientId));
  const deviceUrl = computed(() => toValue(toValue(config)?.deviceUrl));
  const tokenUrl = computed(() => toValue(toValue(config)?.tokenUrl));

  const storageKey = 'deviceToken';
  /**
   * Read the token response from local storage.
   *
   * @return {Promise<TokenResponse|null>}
   */
  const readFromStorage = async () => {
    return loadFromBrowserStorage(
        'local',
        storageKey,
        null
    )[0];
  };

  /**
   * Write the token response to local storage.
   *
   * @param {TokenResponse} details
   * @return {Promise<void>}
   */
  const writeToStorage = async (details) => {
    saveToBrowserStorage('local', storageKey, details);
  };

  /**
   * Clear the local storage.
   *
   * @return {Promise<void>}
   */
  const clearStorage = async () => {
    await window.localStorage.removeItem(storageKey);
  };

  /**
   * Convert a TokenResponse to AuthenticationDetails.
   *
   * @param {TokenResponse} tokenResponse
   * @return {AuthenticationDetails}
   */
  const tokenResponseToAuthDetails = (tokenResponse) => {
    return {
      claims: jwtDecode(tokenResponse.access_token),
      loggedIn: true,
      token: tokenResponse.access_token
    };
  };

  /**
   * @param {number} issued Timestamp, unix nanos, aka Date.now()
   * @param {number} expiresIn seconds after issued that the token expires
   * @param {number} leeway seconds before the token expires that it is considered expired
   * @return {boolean}
   * @private
   */
  const _expired = (issued, expiresIn, leeway = 0) => {
    return issued + expiresIn * 1000 < Date.now() + leeway * 1000;
  };

  /**
   * Return if the given access token has expired.
   *
   * @param {TokenResponse} tokenResponse
   * @param {number} [leeway] If the token expires within this many seconds, it is considered expired.
   * @return {boolean}
   */
  const isAccessTokenExpired = (tokenResponse, leeway = 0) => {
    return _expired(tokenResponse.timestamp, tokenResponse.expires_in, leeway);
  };

  /**
   * @param {string[]} scopes
   * @return {Promise<CodeResponse>}
   */
  const postBeginDeviceFlow = async (scopes) => {
    return fetch(deviceUrl.value, {
      method: 'POST',
      body: new URLSearchParams({
        client_id: clientId.value,
        scope: scopes.join(' ')
      })
    })
        .then(res => {
          if (!res.ok) {
            throw new Error(`Failed to start device auth flow: ${res.status} ${res.statusText}`);
          }
          return res.json();
        })
        .then(data => {
          data.timestamp = Date.now();
          return data;
        });
  };

  /**
   * @param {CodeResponse} lastResponse
   * @return {Promise<TokenResponse>}
   */
  const postCheckDeviceToken = async (lastResponse) => {
    return fetch(tokenUrl.value, {
      method: 'POST',
      body: new URLSearchParams({
        client_id: clientId.value,
        grant_type: 'urn:ietf:params:oauth:grant-type:device_code',
        device_code: lastResponse.device_code
      })
    })
        .then(res => {
          if (!res.ok) {
            throw new Error(`Failed to check device auth flow: ${res.status} ${res.statusText}`);
          }
          return res.json();
        });
  };

  /**
   * @param {TokenResponse} lastResponse
   * @return {Promise<TokenResponse>}
   */
  const postRefreshToken = async (lastResponse) => {
    if (!lastResponse.refresh_token) {
      throw new Error('tokenResponse does not contain a refresh token');
    }
    return fetch(tokenUrl.value, {
      method: 'POST',
      body: new URLSearchParams({
        client_id: clientId.value,
        grant_type: 'refresh_token',
        refresh_token: lastResponse.refresh_token
      })
    })
        .then(res => {
          if (!res.ok) {
            throw new Error(`Failed to refresh token: ${res.status} ${res.statusText}`);
          }
          return res.json();
        });
  };

  // An in-memory cache of the token response either from the server or from local storage.
  const tokenResponse = ref(/** @type {TokenResponse | null} */ null);

  /**
   * Initialise state and cached data.
   *
   * @return {Promise<AuthenticationDetails|null>}
   */
  const init = async () => {
    // check config
    if (!clientId.value) throw new Error('clientId not configured');
    if (!deviceUrl.value) throw new Error('deviceUrl not configured');
    if (!tokenUrl.value) throw new Error('tokenUrl not configured');

    const data = await readFromStorage();
    if (!data) {
      return null; // not authenticated
    }
    tokenResponse.value = data;
    return tokenResponseToAuthDetails(data);
  };

  /**
   * Begin the device login flow.
   * The returned promise will be rejected if the flow could not be started.
   *
   * @param {string[]} [scopes]
   * @return {Promise<Context>}
   */
  const begin = async (scopes = ['profile']) => {
    const code = await postBeginDeviceFlow(scopes);
    let cancel = () => {}; // filled in the promise below
    const complete = new Promise((resolve, reject) => {
      let lastError = null;
      // this interval checks for the user to complete the authentication periodically
      const poll = setInterval(async () => {
        try {
          const newTokenResponse = await postCheckDeviceToken(code);
          await writeToStorage(newTokenResponse);
          tokenResponse.value = newTokenResponse;
          cancel(); // success, stop any timeouts
          resolve(tokenResponseToAuthDetails(newTokenResponse));
        } catch (e) {
          lastError = e; // don't stop, only report the error if all attempts fail
        }
      }, code.interval * 1000);
      // this timeout stops the interval and rejects the promise if the flow expires
      const expired = setTimeout(() => {
        cancel();
        if (!lastError) {
          lastError = new Error('User Code has expired');
        }
        reject(lastError);
      }, code.expires_in * 1000);

      // stop watching for completion (or timeout)
      cancel = () => {
        clearInterval(poll);
        clearTimeout(expired);
      };
    });
    return {code, complete, cancel};
  };


  /**
   * Logout using the store reset function, then store the cleared store in local storage
   *
   * @return {Promise<void>}
   */
  const logout = async () => {
    tokenResponse.value = null;
    await clearStorage();
  };

  /**
   * Refresh the local authentication details.
   *
   * @return {Promise<AuthenticationDetails>}
   */
  const refreshToken = async () => {
    // type cast not strictly correct (can be null at this point) but prevents editor warnings.
    const oldTokenResponse = /** @type {TokenResponse} */ tokenResponse.value;
    if (!oldTokenResponse) {
      throw new Error('Not authenticated');
    }
    // don't bother refreshing if the token is not close to expiry.
    if (!isAccessTokenExpired(oldTokenResponse, 5)) return tokenResponseToAuthDetails(oldTokenResponse);

    const newTokenResponse = await postRefreshToken(oldTokenResponse);
    await writeToStorage(newTokenResponse);
    tokenResponse.value = newTokenResponse;
    return tokenResponseToAuthDetails(newTokenResponse);
  };

  return {
    tokenResponse,
    init,
    begin,
    logout,
    refreshToken
  };
}

/**
 * Returns a {@link Config} that is based on the ui config in the store.
 * Can be passed directly to {@link useDeviceFlow}.
 *
 * @return {MaybeRefOrGetter<Config>}
 */
export function useUiConfig() {
  const uiConfig = useUiConfigStore();
  return computed(() => {
    const deviceFlowConfig = uiConfig.config?.auth?.deviceFlow;
    if (!deviceFlowConfig) return {}; // invalid, device flow not configured
    if (deviceFlowConfig === true) {
      // use keycloak config as the basis for our config
      const kcConfig = computed(() => uiConfig.auth.keycloak);
      const kcRealmPath = (path) => {
        let baseUrl = kcConfig.value?.url;
        const realm = kcConfig.value?.realm;
        if (!baseUrl || !realm) {
          return null;
        }
        if (baseUrl.endsWith('/')) {
          baseUrl = baseUrl.substring(0, baseUrl.length - 1);
        }
        if (path.startsWith('/')) {
          path = path.substring(1);
        }
        return `${baseUrl}/realms/${realm}/${path}`;
      };
      return {
        clientId: computed(() => kcConfig.value?.clientId),
        deviceUrl: computed(() => kcRealmPath('/protocol/openid-connect/auth/device')),
        tokenUrl: computed(() => kcRealmPath('/protocol/openid-connect/token'))
      };
    } else {
      return deviceFlowConfig;
    }
  });
}
