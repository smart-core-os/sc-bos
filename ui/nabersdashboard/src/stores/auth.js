import {defineStore} from 'pinia';
import {ref} from 'vue';
import {grpcWebEndpoint} from '@/api/config.js';
import {useUiConfigStore} from '@/stores/uiConfig.js';

export const useAuthStore = defineStore('nabersdashboard:auth', () => {
  const token = ref('');

  /**
   * Fetches a fresh access token from the SC-BOS OAuth2 endpoint.
   * Credentials are read from the VITE_DASHBOARD_USERNAME/PASSWORD env vars (baked in
   * at build time), falling back to the ui config for local dev overrides.
   */
  const fetchToken = async () => {
    const endpoint = await grpcWebEndpoint();
    if (!endpoint) return;
    const uiConfig = useUiConfigStore();
    const username = import.meta.env.VITE_DASHBOARD_USERNAME || uiConfig.config.username;
    const password = import.meta.env.VITE_DASHBOARD_PASSWORD || uiConfig.config.password;
    if (!username || !password) return;
    try {
      const resp = await fetch(`${endpoint}/oauth2/token`, {
        method: 'POST',
        headers: {'Content-Type': 'application/x-www-form-urlencoded'},
        body: new URLSearchParams({grant_type: 'password', username, password}),
      });
      if (!resp.ok) throw new Error(`token fetch failed: ${resp.status}`);
      const {access_token} = await resp.json();
      token.value = access_token;
    } catch (err) {
      console.error('Failed to fetch access token', err);
    }
  };

  return {token, fetchToken};
});
