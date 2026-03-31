import {useUiConfigStore} from '@/stores/ui-config.js';
import Keycloak from 'keycloak-js';

/**
 * @type {Promise<Keycloak> | null}
 */
let instance = null;
/**
 * Events from the keycloak instance.
 *
 * Supported events are and follow the semantics and payload of the corresponding onXxx keycloak methods:
 *  - ready
 *  - authError
 *  - authLogout
 *  - authSuccess
 *  - authRefreshError
 *  - authRefreshSuccess
 *  - tokenExpired
 *
 * @type {EventTarget}
 */
export const events = new EventTarget();

/**
 * @return {Promise<Keycloak>}
 */
export function keycloak() {
  if (instance === null) {
    instance = newKeycloak();
  }
  return instance;
}

/**
 * @return {Promise<Keycloak>}
 */
async function newKeycloak() {
  const kcConfig = await constructorConfig();
  if (!kcConfig) {
    return null;
  }
  const kc = new Keycloak(kcConfig);
  // setup event handling
  kc.onReady = (authenticated) => {
    const e = new Event('ready');
    e.authenticated = authenticated;
    events.dispatchEvent(e);
  };
  kc.onAuthError = (errorData) => {
    const e = new Event('authError');
    e.errorData = errorData;
    events.dispatchEvent(e);
  };
  kc.onAuthLogout = () => events.dispatchEvent(new Event('authLogout'));
  kc.onAuthSuccess = () => events.dispatchEvent(new Event('authSuccess'));
  kc.onAuthRefreshError = () => events.dispatchEvent(new Event('authRefreshError'));
  kc.onAuthRefreshSuccess = () => events.dispatchEvent(new Event('authRefreshSuccess'));
  kc.onTokenExpired = () => events.dispatchEvent(new Event('tokenExpired'));

  await kc.init(await initConfig());
  return kc;
}

/**
 * @return {Promise<import('keycloak-js').KeycloakConfig | string>}
 */
async function constructorConfig() {
  const uiConfig = useUiConfigStore();
  await uiConfig.configPromise; // make sure it's loaded
  return uiConfig.auth.keycloak ?? {
    realm: import.meta.env.VITE_KEYCLOAK_REALM || 'smart-core',
    url: import.meta.env.VITE_KEYCLOAK_URL || 'http://localhost:8888/',
    clientId: import.meta.env.VITE_KEYCLOAK_CLIENT_ID || 'sc-apps'
  };
}

/**
 * @return {Promise<import('keycloak-js').KeycloakInitOptions>}
 */
async function initConfig() {
  // todo: get keycloak init config from somewhere non-hard-coded
  return {
    onLoad: 'check-sso',
    silentCheckSsoRedirectUri:
        window.location.origin + import.meta.env.BASE_URL + 'silent-check-sso-v2.html',
    silentCheckSsoFallback: false
  };
}
