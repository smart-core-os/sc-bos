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
 * Derives a device type from a Smart Core source name: the token before the first hyphen
 * of the last '/'-segment (e.g. "a/b/FCU-LN1-01" -> "FCU"). Falls back to the whole short
 * name when there is no hyphen.
 *
 * @param {string} sourceName
 * @return {string}
 */
export function deviceType(sourceName) {
  return (sourceName.split('/').pop() ?? '').split('-')[0];
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
 * Builds one row per message — `Source name, Topic, Point 1..N` — the row widening to hold
 * each device's point names.
 *
 * @param {Array<{sourceName: string, topic: string, payload: string}>} messages
 * @return {{header: string[], rows: string[][]}}
 */
export function rowsByDevice(messages) {
  let maxPoints = 0;
  const rows = messages.map((msg) => {
    const points = parsePoints(msg.payload);
    maxPoints = Math.max(maxPoints, points.length);
    return [msg.sourceName, msg.topic, ...points];
  });
  return {header: ['Source name', 'Topic', ...pointColumns(maxPoints)], rows};
}

/**
 * Builds one row per distinct point set within a device type — `Device type, Devices,
 * Point 1..N`. Devices of a type are grouped, but only those with an identical point set
 * are collapsed, so a subset/superset variant appears as its own (longer) row rather than
 * being silently merged. Types are sorted alphabetically; variants within a type by size.
 *
 * @param {Array<{sourceName: string, topic: string, payload: string}>} messages
 * @return {{header: string[], rows: string[][]}}
 */
export function rowsByDeviceType(messages) {
  // type -> (signature -> {points, count})
  const byType = new Map();
  for (const msg of messages) {
    const type = deviceType(msg.sourceName);
    const points = parsePoints(msg.payload);
    const signature = JSON.stringify([...points].sort());
    if (!byType.has(type)) byType.set(type, new Map());
    const variants = byType.get(type);
    const existing = variants.get(signature);
    if (existing) existing.count++;
    else variants.set(signature, {points, count: 1});
  }

  let maxPoints = 0;
  const rows = [];
  for (const type of [...byType.keys()].sort()) {
    const variants = [...byType.get(type).values()].sort((a, b) => a.points.length - b.points.length);
    for (const {points, count} of variants) {
      maxPoints = Math.max(maxPoints, points.length);
      rows.push([type, String(count), ...points]);
    }
  }
  return {header: ['Device type', 'Devices', ...pointColumns(maxPoints)], rows};
}

/**
 * Builds the full CSV rows (header first) for the given messages and grouping mode.
 *
 * @param {Array<{sourceName: string, topic: string, payload: string}>} messages
 * @param {'device'|'type'} mode
 * @return {string[][]}
 */
export function buildPointsCsv(messages, mode) {
  const {header, rows} = mode === 'type' ? rowsByDeviceType(messages) : rowsByDevice(messages);
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
