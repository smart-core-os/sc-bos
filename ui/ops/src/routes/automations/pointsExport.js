import {listServices, ServiceNames} from '@/api/ui/services';
import {listExportedPoints} from '@/api/ui/udmiExport';

/**
 * Extracts the point names from a UDMI pointset payload. Handles both the conformant
 * envelope (`{points: {name: {...}}}`) and a bare points map (`{name: {present_value}}`,
 * as the mock driver emits). Returns an empty array for payloads that aren't a pointset
 * (state/metadata/other).
 *
 * @param {string} payload
 * @return {string[]}
 */
export function parsePoints(payload) {
  let parsed;
  try {
    parsed = JSON.parse(payload);
  } catch {
    return [];
  }
  if (!parsed || typeof parsed !== 'object' || Array.isArray(parsed)) return [];
  // Prefer the conformant envelope `{points: {name: {...}}}` — every key under it is a point.
  const envelope = parsed.points;
  if (envelope && typeof envelope === 'object' && !Array.isArray(envelope)) {
    return Object.keys(envelope);
  }
  // Fall back to a bare points map `{name: {present_value}}` (as the mock driver emits):
  // return only the keys whose value looks like a point value, so sibling metadata keys
  // (timestamp, version, ...) alongside points are not mistaken for point names.
  return Object.entries(parsed)
      .filter(([, v]) => v && typeof v === 'object' && Object.hasOwn(v, 'present_value'))
      .map(([k]) => k);
}

// Matches the pointset-event portion of a UDMI topic, covering both the UDMI standard
// suffix ".../event/pointset/points" and the shorter ".../events/pointset" some drivers use.
const POINTSET_EVENT_RE = /\/events?\/pointset/;

/**
 * Extracts the BDNS functional asset name from a pointset event topic: the single path
 * segment immediately before the "/event(s)/pointset" portion
 * (e.g. "JLL/GB-LON-1BG/AV/AMP-109151/events/pointset" -> "AMP-109151";
 * "site/.../FCU-LN1-01/event/pointset/points" -> "FCU-LN1-01"). Returns an empty string
 * when the topic isn't a pointset event topic.
 *
 * @param {string} topic
 * @return {string}
 */
export function bdnsAssetName(topic) {
  const m = POINTSET_EVENT_RE.exec(topic);
  if (!m) return '';
  return topic.slice(0, m.index).split('/').pop() ?? '';
}

/**
 * Returns `n` CSV column headers named "Point 1" .. "Point n".
 *
 * @param {number} n
 * @return {string[]}
 */
function pointColumns(n) {
  return Array.from({length: n}, (_, i) => `Point ${i + 1}`);
}

/**
 * Builds the full CSV rows (header first): one row per device (message) —
 * `Source name, Topic, BDNS functional asset name, Point 1..N` — the row widening to hold
 * each device's point names. Only pointset event topics are included; state, metadata and
 * other topics are excluded. Rows are de-duplicated by source+topic so a device exported by
 * more than one udmi automation (e.g. a gateway and the node it proxies) appears once.
 *
 * @param {Array<{sourceName: string, topic: string, payload: string}>} messages
 * @return {string[][]}
 */
export function buildPointsCsv(messages) {
  const seen = new Set();
  const pointsetMessages = (messages ?? []).filter((msg) => {
    if (!POINTSET_EVENT_RE.test(msg.topic)) return false;
    const key = JSON.stringify([msg.sourceName, msg.topic]);
    if (seen.has(key)) return false;
    seen.add(key);
    return true;
  });
  let maxPoints = 0;
  const rows = pointsetMessages.map((msg) => {
    const points = parsePoints(msg.payload);
    maxPoints = Math.max(maxPoints, points.length);
    return [msg.sourceName, msg.topic, bdnsAssetName(msg.topic), ...points];
  });
  const header = ['Source name', 'Topic', 'BDNS functional asset name', ...pointColumns(maxPoints)];
  return [header, ...rows];
}

/**
 * Collects exported messages from every udmi automation across the given cohort nodes.
 *
 * For each node it lists that node's automations, keeps the ones of type "udmi", reads the
 * automation's configured name from its configRaw, and calls ListExportedPoints for it.
 * All requests go over the single grpc-web endpoint and are routed by name. Best-effort:
 * a node or automation that fails is skipped and recorded in `errors`, so one unreachable
 * node never fails the whole export.
 *
 * @param {import('@/stores/cohort.js').CohortNode[]} nodes
 * @return {Promise<{messages: Array<{sourceName: string, topic: string, payload: string}>, errors: Array<{node: string, automation?: string, error: *}>}>}
 */
export async function collectCohortMessages(nodes) {
  const errors = [];

  const perNode = await Promise.all((nodes ?? []).map(async (node) => {
    let services;
    try {
      services = [];
      let pageToken = '';
      do {
        const res = await listServices(
            {name: node.name + '/' + ServiceNames.Automations, pageSize: 1000, pageToken});
        services.push(...(res?.servicesList ?? []).filter((s) => s.type === 'udmi'));
        pageToken = res?.nextPageToken ?? '';
      } while (pageToken);
    } catch (error) {
      errors.push({node: node.name, error});
      return [];
    }

    const perAutomation = await Promise.all(services.map(async (service) => {
      let name;
      try {
        name = service.configRaw ? JSON.parse(service.configRaw).name : undefined;
      } catch {
        name = undefined;
      }
      if (!name) {
        // Record rather than silently drop, so the caller's "may be incomplete" warning fires.
        errors.push({node: node.name, automation: service.id, error: new Error('udmi automation has no configured name')});
        return [];
      }
      try {
        const out = await listExportedPoints({name});
        return out?.messagesList ?? [];
      } catch (error) {
        errors.push({node: node.name, automation: name, error});
        return [];
      }
    }));
    return perAutomation.flat();
  }));

  return {messages: perNode.flat(), errors};
}
