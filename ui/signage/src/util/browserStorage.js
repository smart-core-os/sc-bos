/**
 * This utility is used to save values to browser storage.
 * It takes in sources, keys, and values. Each can be a single value or an array.
 * Loops through the keys and values and saves them to the specified browser storage(s).
 * If the value is not a string, it will be stringified.
 *
 * @example
 * Suppose you want to store user data and a theme preference
 * const user = { name: 'Alice', age: 30 };
 * const theme = 'dark';
 * saveToBrowserStorage('local', ['user', 'theme'], [user, theme]);
 * This will save the user object and theme string to localStorage
 *
 * @param {string | string[]} sources - 'local', 'session', or array of both
 * @param {string | string[]} keys - Key or array of keys for storage
 * @param {*} values - Value or array of values corresponding to the keys
 */
export const saveToBrowserStorage = (sources, keys, values) => {
  const normalizedSources = Array.isArray(sources) ? sources : [sources];
  const normalizedKeys = Array.isArray(keys) ? keys : [keys];
  const normalizedValues = Array.isArray(values) ? values : [values];

  normalizedSources.forEach(source => {
    const storage = source === 'local' ? window.localStorage : window.sessionStorage;

    normalizedKeys.forEach((key, index) => {
      if (index < normalizedValues.length) {
        const value = normalizedValues[index];
        const valueToStore = typeof value === 'string' ? value : JSON.stringify(value);
        storage.setItem(key, valueToStore);
      }
    });
  });
};


/**
 * This utility is used to retrieve values from browser storage.
 * It takes in sources, keys, and default values. Each can be a single value or an array.
 * Retrieves each item by key from the specified browser storage(s) and returns them in an array.
 *
 * @example
 * Suppose you have stored user and theme in localStorage and want to retrieve them
 * with 'defaultUser' and 'light' as their respective default values
 * const [user, theme] = loadFromBrowserStorage(
 *   'local',
 *   ['user', 'theme'],
 *   [{ name: 'defaultUser' }, 'light']
 * );
 * This will load the 'user' and 'theme' from localStorage, or will use the provided default values
 *
 * @param {string | string[]} sources - 'local', 'session', or array of both
 * @param {string | string[]} keys - Key or array of keys to retrieve from storage
 * @param {MaybeRefOrGetter | MaybeRefOrGetter[]} defaults - Default value or array of default values
 * @return {any[]} - Array of retrieved values
 */
export const loadFromBrowserStorage = (sources, keys, defaults) => {
  const normalizedSources = Array.isArray(sources) ? sources : [sources];
  const normalizedKeys = Array.isArray(keys) ? keys : [keys];
  const normalizedDefaults = Array.isArray(defaults) ? defaults : [defaults];

  return normalizedSources.map(source => {
    const storage = source === 'local' ? window.localStorage : window.sessionStorage;

    return normalizedKeys.map((key, index) => {
      const storedValue = storage.getItem(key);

      // If stored value is null, use default value
      if (storedValue === null) {
        if (index < normalizedDefaults.length) {
          if (typeof normalizedDefaults[index] === 'function') {
            return normalizedDefaults[index]();
          } else {
            return normalizedDefaults[index];
          }
        }
        return null; // In case there's no default value for this index
      }

      // Parse the value if it's JSON, otherwise return it as is
      try {
        return JSON.parse(storedValue);
      } catch {
        return storedValue;
      }
    });
  }).flat(); // Flatten the array because each source maps to an array of values, resulting in an array of arrays.
};


