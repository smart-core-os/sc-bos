import {listExportedPoints} from '@/api/ui/udmiExport';

/**
 * Collects the exported devices from every node in the given cohort.
 *
 * UdmiExportApi is announced against each node's name and covers every udmi automation on
 * that node, so this is one request per node. All requests go over the single grpc-web
 * endpoint and are routed by name. Best-effort: a node that fails is skipped and recorded
 * in `errors`, so one unreachable node never fails the whole export.
 *
 * @param {import('@/stores/cohort.js').CohortNode[]} nodes
 * @return {Promise<{devices: DevicePoints.AsObject[], errors: Array<{node: string, error: *}>}>}
 */
export async function collectCohortDevices(nodes) {
  const errors = [];

  const perNode = await Promise.all((nodes ?? []).map(async (node) => {
    try {
      const res = await listExportedPoints({name: node.name});
      return res?.devicesList ?? [];
    } catch (error) {
      errors.push({node: node.name, error});
      return [];
    }
  }));

  return {devices: perNode.flat(), errors};
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
 * Lays the exported devices out as CSV rows (header first): one row per device —
 * `Source name, Topic, BDNS functional asset name, Point 1..N` — the table widening to hold
 * the device with the most points. Devices are de-duplicated by source and topic, so one
 * exported by more than one node (e.g. a gateway and the node it proxies) appears once.
 *
 * @param {DevicePoints.AsObject[]} devices
 * @return {string[][]}
 */
export function buildPointsCsv(devices) {
  const seen = new Set();
  let maxPoints = 0;
  const rows = [];
  for (const device of devices ?? []) {
    const key = JSON.stringify([device.sourceName, device.topic]);
    if (seen.has(key)) continue;
    seen.add(key);
    const points = device.pointsList ?? [];
    maxPoints = Math.max(maxPoints, points.length);
    rows.push([device.sourceName, device.topic, device.assetName, ...points]);
  }
  const header = ['Source name', 'Topic', 'BDNS functional asset name', ...pointColumns(maxPoints)];
  return [header, ...rows];
}
