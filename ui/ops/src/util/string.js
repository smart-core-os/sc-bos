/**
 * Inserts spaces into camelCased string
 *
 * @param {string} key
 * @return {string}
 */
export function camelToSentence(key) {
  return key.replace(/([A-Z])/g, ' $1');
}

/**
 * Capitalizes the first letter of a string
 *
 * @param {string|undefined} str
 * @return {string}
 */
export function toSentenceCase(str) {
  if (!str) return '';
  return str.charAt(0).toUpperCase() + str.slice(1);
}
