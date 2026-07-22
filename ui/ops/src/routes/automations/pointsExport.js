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
  if (!parsed || typeof parsed !== 'object') return [];
  // Prefer the conformant envelope; fall back to treating the whole object as the
  // points map when its values look like point values (have a present_value).
  let points = parsed.points;
  if (!points || typeof points !== 'object') {
    const looksLikePointsMap = Object.values(parsed).some(
        (v) => v && typeof v === 'object' && Object.hasOwn(v, 'present_value'));
    if (!looksLikePointsMap) return [];
    points = parsed;
  }
  return Object.keys(points);
}

/**
 * Extracts the BDNS functional asset name from an MQTT topic: the single path segment
 * immediately before "/event/pointset/points"
 * (e.g. "site/.../devices/FCU-LN1-01/event/pointset/points" -> "FCU-LN1-01"). Returns an
 * empty string when the topic has no such suffix.
 *
 * @param {string} topic
 * @return {string}
 */
export function bdnsAssetName(topic) {
  const marker = '/event/pointset/points';
  const i = topic.indexOf(marker);
  if (i < 0) return '';
  return topic.slice(0, i).split('/').pop() ?? '';
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
 * each device's point names.
 *
 * @param {Array<{sourceName: string, topic: string, payload: string}>} messages
 * @return {string[][]}
 */
export function buildPointsCsv(messages) {
  let maxPoints = 0;
  const rows = messages.map((msg) => {
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
      const res = await listServices({name: node.name + '/' + ServiceNames.Automations, pageSize: 1000});
      services = (res?.servicesList ?? []).filter((s) => s.type === 'udmi');
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
      if (!name) return [];
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
