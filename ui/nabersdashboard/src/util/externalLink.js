/**
 * Validation for links the config supplies.
 *
 * The dashboard's outbound links are configured per site, not written in the
 * source, so their destinations arrive as strings from `dashboards-config.json`.
 * A string that reaches an `href` unchecked is a `javascript:` URL away from
 * running as script, and a typo is a link that looks live and goes nowhere.
 * Both are cheap to rule out at the point the value enters the component.
 *
 * @module util/externalLink
 */

/** The only schemes a configured link may use. */
const ALLOWED_PROTOCOLS = new Set(['http:', 'https:']);

/**
 * Normalise a configured link, or reject it.
 *
 * Rejects anything that is not an absolute — or `base`-resolvable — http(s) URL,
 * so `javascript:`, `data:` and `file:` never reach an anchor. Note that a
 * protocol-relative `//host/path` is rejected when no `base` is given: without
 * one it does not parse, and silently inheriting the page's scheme is not
 * something a config author should have to reason about.
 *
 * @param {*} value the configured value, of whatever type the JSON held
 * @param {string} [base] resolve relative paths against this, for deployments
 *   that serve the ops UI from the dashboard's own origin
 * @return {string|null} the normalised absolute URL, or null if unusable
 */
export function safeHttpUrl(value, base = undefined) {
  if (typeof value !== 'string') return null;
  const trimmed = value.trim();
  if (!trimmed) return null;

  let parsed;
  try {
    parsed = new URL(trimmed, base);
  } catch {
    return null;
  }
  if (!ALLOWED_PROTOCOLS.has(parsed.protocol)) return null;
  return parsed.href;
}
