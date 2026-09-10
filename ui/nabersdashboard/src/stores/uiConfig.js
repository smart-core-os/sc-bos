import {defineStore} from 'pinia';
import {computed, ref, toValue} from 'vue';

export const useUiConfigStore = defineStore('nabersdashboard:uiConfig', () => {
  /**
   * @private
   */
  const _config = ref({});
  const _loaded = ref(false);
  let _configResolve;
  const configPromise = new Promise((resolve) => _configResolve = resolve);

  /** @type {import('vue').ComputedRef<string>} */
  const configUrl = computed(() => {
    let url = import.meta.env.VITE_UI_CONFIG_URL;
    if (!url) {
      url = import.meta.env.BASE_URL + 'dashboards-config.json';
    }
    return url;
  });

  /**
   * Loads the config from the server
   */
  async function loadConfig() {
    console.debug('Loading UI config from', configUrl.value);
    if (_loaded.value) {
      return;
    }
    const url = configUrl.value;
    try {
      const res = await fetch(url);
      const json = await res.json();
      const section = json.airquality ?? json;
      if (!section?.config) {
        console.warn(
          `UI config at ${url} has no "config" object; every dashboard will read as unconfigured`,
          section);
      }
      migrateConfig(section?.config);
      _config.value = section;
    } catch (e) {
      console.warn('Failed to load config from server, using default config', e);
      _config.value = _defaultConfig;
    }
    _configResolve(config.value);
    _loaded.value = true;
  }

  const config = computed(() => _config.value?.config ?? {});

  /**
   * Gets the value of path from either uiConfig config or defaultConfig, depending on presence.
   *
   * @template T
   * @param {string} path
   * @param {T?} def
   * @return {T}
   */
  const getOrDefault = (path, def) => {
    const parts = path.split('.');
    let a = config.value;
    let b = _defaultConfig?.config;
    for (let i = 0; i < parts.length; i++) {
      a = a?.[parts[i]];
      b = b?.[parts[i]];
    }
    return a ?? b ?? toValue(def);
  };

  const owmKey = computed(() => {
    // First check config file, then fall back to environment variable
    return config.value.openWeatherMapApiKey || import.meta.env.VITE_OPENWEATHERMAP_API_KEY || '';
  });

  return {
    configUrl,
    loadConfig,
    config,
    owmKey,
    configPromise,
    defaultConfig: _defaultConfig,
    getOrDefault
  };
});

/**
 * Converts legacy config properties to their current equivalents.
 *
 * Tolerates a missing config object. It used to throw on one, and because the
 * only caller is inside `loadConfig`'s try block the whole file was then
 * discarded for the built-in default — so a config served with a typo'd or absent
 * `config` key produced a dashboard with no meters and no message saying why.
 *
 * @param {Object} [config] The config to migrate - modified in place
 */
function migrateConfig(config) {
  if (!config || typeof config !== 'object') return;
  if (Object.hasOwn(config, 'proxy') && !Object.hasOwn(config, 'gateway')) {
    console.warn('ui config property "proxy" is deprecated, please use "gateway" instead');
    config.gateway = config.proxy;
    delete config.proxy;
  }
}

/**
 * The default config for the UI.
 *
 * @private
 */
const _defaultConfig = {
  config: {
    title: 'NABERS Dashboard',
    sensors: []
  }
};
