import useDeviceFlow, {useUiConfig as deviceFlowUseUiConfig} from '@/composables/authentication/useDeviceFlow';
import useKeyCloak from '@/composables/authentication/useKeyCloak';
import useLocal from '@/composables/authentication/useLocal';
import {useUiConfigStore} from '@/stores/ui-config';
import {defineStore} from 'pinia';
import {computed, ref, watch} from 'vue';
import {useRoute, useRouter} from 'vue-router';

/**
 * @typedef AuthenticationDetails
 * @property {Object<string,*>} claims
 * @property {boolean} loggedIn
 * @property {string} token
 */
/**
 * Describes the interface an authentication provider should implement.
 * Providers should also provide a mechanism for beginning the authentication flow.
 *
 * @typedef AuthenticationProvider
 * @property {function(): Promise<AuthenticationDetails|null>} init
 *  Initialise the auth provider, returning null if not authenticated
 * @property {function(): Promise<void>} logout
 * @property {function(number): Promise<AuthenticationDetails|null>} refreshToken
 */

export const useAccountStore = defineStore('accountStore', () => {
  const uiConfig = useUiConfigStore();
  const keyCloak = useKeyCloak();
  const localAuth = useLocal();
  const deviceFlow = useDeviceFlow(deviceFlowUseUiConfig());

  const forceLogIn = ref(false); // true when login is required even when already logged in
  const router = useRouter();
  const route = useRoute();

  // initComplete is resolved (or rejected) the first time initialise is called.
  // Functions can `await initComplete` to make sure that any authenticationDetails -
  // including authProvider - are set correctly.
  let initResolved;
  let initRejected;
  const initComplete = new Promise((resolve, reject) => {
    initResolved = resolve;
    initRejected = reject;
  });
  const isInitialized = ref(false);

  // Set up the storage for the login: authProvider, claims, login status and token
  const authenticationDetails = ref(
      /** @type {AuthenticationDetails & {authProvider: string}} */
      {
        authProvider: '',
        claims: {},
        loggedIn: false,
        token: ''
      });
  const snackbar = ref({
    message: 'Failed to sign in, please try again',
    visible: false
  });


  /**
   * Reset the store values to defaults
   *
   * @return {void}
   */
  const resetStoreToDefaults = () => {
    authenticationDetails.value = {
      authProvider: '',
      claims: {},
      loggedIn: false,
      token: ''
    };
  };

  /**
   * Helper for initialise that contains all the logic so any 'finally' logic can be run for all cases.
   *
   * @param {string[]} [providerNames] List of providers to initialise, defaults to all
   * @return {Promise<void>}
   */
  const _initialise = async (providerNames) => {
    if (uiConfig.auth.disabled) {
      return;
    }

    const providers = [
      {
        name: 'keyCloakAuth',
        init: keyCloak.init,
        enabled: () => Boolean(uiConfig.auth.keycloak)
      },
      {
        name: 'deviceFlow',
        init: deviceFlow.init,
        enabled: () => Boolean(uiConfig.config?.auth?.deviceFlow)
      },
      {
        name: 'localAuth',
        init: localAuth.init,
        enabled: () => true
      }
    ]
        // only initialise the providers we've been asked to (or all of them)
        .filter((provider) => !providerNames || providerNames.includes(provider.name));

    let loginDetails = null; // details from the first successful init attempt (that returned 'logged in')
    const availableProviderNames = [];
    for (const provider of providers) {
      if (!provider.enabled()) continue;
      try {
        if (loginDetails === null) {
          const details = await provider.init();
          if (details) {
            loginDetails = {
              ...details,
              authProvider: provider.name
            };
          }
        }
        availableProviderNames.push(provider.name);
      } catch (e) {
        console.error(`${provider.name} initialization failed`, e);
        snackbar.value = {
          message: `${provider.name} initialization failed: ${e?.error ?? e}`,
          visible: true
        };
      }
    }

    availableAuthProviders.value = availableProviderNames;
    if (loginDetails) {
      // we are logged in already
      authenticationDetails.value = loginDetails;
    }
  };

  /**
   * Initialize Keycloak and Local Auth instances, so we can check if the user is logged in and/or manage the login flow
   *
   * @param {string[]} [providerNames] List of providers to initialise, defaults to all
   * @return {Promise<void>}
   */
  const initialise = async (providerNames) => {
    if (isInitialized.value) return;
    try {
      await _initialise(providerNames);
      if (!isLoggedIn.value) {
        await tryAutoLogin();
      }
      initResolved();
    } catch (e) {
      initRejected(e);
    } finally {
      isInitialized.value = true;
    }
  };

  //
  // ----------------------------------- //
  //
  /**
   * Check if authentication is disabled
   *
   * @type {import('vue').ComputedRef<boolean>}
   */
  const isAuthenticationDisabled = computed(() => {
    return uiConfig.auth.disabled;
  });

  /**
   * Collect the available authentication providers
   *
   * @type {import('vue').Ref<string[]>}
   */
  const availableAuthProviders = ref(['localAuth']);

  /**
   * Retrieve the authentication provider used for login
   *
   * @type {import('vue').ComputedRef<string|null>}
   */
  const activeAuthProvider = computed(() => {
    return authenticationDetails.value.authProvider || null;
  });

  /**
   * Returns the login status depending on the authentication provider
   *
   * @type {import('vue').ComputedRef<boolean>}
   */
  const isLoggedIn = computed(() => {
    const detailsAvailable = !!(authenticationDetails.value.token);

    return detailsAvailable || isAuthenticationDisabled.value;
  });

  const claims = computed(() => authenticationDetails.value.claims ?? {});

  /**
   * Returns the full name of the logged in user
   *
   * @type {import('vue').ComputedRef<string>}
   */
  const fullName = computed(() => authenticationDetails.value.claims?.name || '');

  /**
   * Returns the email of the logged in user
   *
   * @type {import('vue').ComputedRef<string>}
   */
  const email = computed(() => authenticationDetails.value.claims?.email || '');

  /**
   * Returns the roles of the logged in user
   *
   * @type {import('vue').ComputedRef<string[]>}
   */
  const roles = computed(() => authenticationDetails.value.claims?.roles || []);
  const zones = computed(() => claims.value?.zones ?? []);
  //
  // ----------------------------------- //
  //
  // Dynamic controls - depending on the active authentication provider
  //
  /** @typedef {{ username: string, password: string }} LocalAuthLoginValues */

  /**
   * Perform a login using the given fn and store details and redirect if needed.
   *
   * @param {string} authProvider
   * @param {function(): Promise<AuthenticationDetails>} fn
   * @return {Promise<void>}
   */
  const doLogin = async (authProvider, fn) => {
    await initComplete;
    const details = await fn();
    if (details) {
      authenticationDetails.value = {...details, authProvider};
      forceLogIn.value = false;
      await redirectAfterLogin();
    }
  };

  /**
   * Log in with local authentication using the given username and password
   *
   * @param {LocalAuthLoginValues} values
   * @return {Promise<void>}
   */
  const loginWithLocalAuth = async (values) => {
    return doLogin('localAuth', () => localAuth.login(values.username, values.password));
  };

  /**
   * Log in with KeyCloak using the given options.
   *
   * @param {import('keycloak-js').KeycloakLoginOptions} [options]
   * @return {Promise<void>}
   */
  const loginWithKeyCloak = async (options) => {
    if (forceLogIn.value) {
      options.prompt = 'login';
    }
    return doLogin('keyCloakAuth', () => keyCloak.login(options));
  };

  /**
   * Begin the OAuth Device Flow, returning context information to display to the user for them to complete the flow.
   *
   * @param {string[]} [scopes]
   * @return {Promise<import('@/composables/authentication/useDeviceFlow').Context>}
   */
  const beginDeviceFlow = async (scopes) => {
    await initComplete;
    const ctx = await deviceFlow.begin(scopes);
    ctx.complete = ctx.complete.then(async (details) => {
      if (details) {
        authenticationDetails.value = {...details, authProvider: 'deviceFlow'};
        await redirectAfterLogin();
      }
      return details;
    });
    return ctx;
  };

  /**
   * Redirect to the login page if the user is not already there.
   *
   * @return {Promise<void>}
   */
  const redirectToLogin = async () => {
    if (router.currentRoute.path !== '/login') {
      await router.push('/login');
    }
  };

  /**
   * Redirect to the home page after a login.
   * The router interceptor will deal with the actual routing if there are redirects or reconfiguration events happening.
   *
   * @return {Promise<void>}
   */
  const redirectAfterLogin = async () => {
    await router.push(uiConfig.homePath);
  };

  /**
   * Logout of the active authentication provider, then reset the store to defaults.
   * If a reason is provided, display a snackbar with the reason.
   *
   * @param {string} reason
   * @return {Promise<void>}
   */
  const logout = async (reason) => {
    await initComplete;
    if (activeAuthProvider.value === 'keyCloakAuth') {
      await keyCloak.logout();
    } else if (activeAuthProvider.value === 'localAuth') {
      await localAuth.logout();
    } else if (activeAuthProvider.value === 'deviceFlow') {
      await deviceFlow.logout();
    }

    if (reason) {
      snackbar.value = {
        message: 'Logged out: ' + reason,
        visible: true
      };
    }

    resetStoreToDefaults();
    await redirectToLogin();
  };

  /**
   * @param {function(): Promise<AuthenticationDetails>} fn
   * @return {Promise<void>}
   */
  const doRefreshToken = async (fn) => {
    try {
      const details = await fn();
      authenticationDetails.value = {...details, authProvider: authenticationDetails.value.authProvider};
    } catch (e) {
      resetStoreToDefaults();
      snackbar.value = {
        message: 'Session expired, please log in again: ' + e,
        visible: true
      };
    }
  };

  const refreshToken = async (leeway) => {
    await initComplete;
    if (activeAuthProvider.value === 'keyCloakAuth') {
      return doRefreshToken(() => keyCloak.refreshToken(leeway));
    } else if (activeAuthProvider.value === 'localAuth') {
      return doRefreshToken(() => localAuth.refreshToken(leeway));
    } else if (activeAuthProvider.value === 'deviceFlow') {
      return doRefreshToken(() => deviceFlow.refreshToken(leeway));
    }
  };

  // Deduplicate concurrent auto-relogin attempts (e.g. multiple simultaneous 401 errors)
  let _reloginPromise = null;

  /**
   * Silently re-authenticates using auto-login credentials from the UI config.
   * Returns true on success, false if credentials are not configured or login fails.
   * Concurrent calls are collapsed into a single login attempt.
   *
   * @return {Promise<boolean>}
   */
  const tryAutoLogin = () => {
    if (_reloginPromise) return _reloginPromise;
    _reloginPromise = (async () => {
      try {
        const creds = await localAuth.getSavedCredentials();
        if (!creds?.username || !creds?.password) return false;
        const details = await localAuth.login(creds.username, creds.password);
        if (details) {
          authenticationDetails.value = {...details, authProvider: 'localAuth'};
          forceLogIn.value = false;
          return true;
        }
        return false;
      } catch {
        return false;
      } finally {
        _reloginPromise = null;
      }
    })();
    return _reloginPromise;
  };

  const shouldRedirect = computed(() => {
    return isLoggedIn.value && route.name === 'login' && !forceLogIn.value;
  });
  watch(shouldRedirect, (should) => {
    if (!should) return;
    redirectAfterLogin().catch(() => {});
  }, {immediate: true});

  return {
    initialise,
    isInitialized,
    authenticationDetails,
    snackbar,
    resetStoreToDefaults,

    forceLogIn,
    shouldRedirect,

    isAuthenticationDisabled,
    availableAuthProviders,
    isLoggedIn,
    claims,
    fullName,
    email,
    roles,
    zones,

    loginWithLocalAuth,
    loginWithKeyCloak,
    beginDeviceFlow,
    logout,
    refreshToken,
    tryAutoLogin,


    /**
     * Check if the given provider is available.
     *
     * @param {string} provider
     * @return {boolean}
     */
    hasProvider(provider) {
      return availableAuthProviders.value.includes(provider);
    },

    /**
     * Check if the given provider is the only available provider.
     *
     * @param {string} provider
     * @return {boolean}
     */
    isOnlyProvider(provider) {
      return availableAuthProviders.value.length === 1 && this.hasProvider(provider);
    }
  };
});
