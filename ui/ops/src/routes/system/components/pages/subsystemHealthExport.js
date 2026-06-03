import {timestampToDate} from '@/api/convpb.js';
import {listHealthChecks, reliabilityStateToString} from '@/api/sc/traits/health.js';
import {downloadCSVRows} from '@/util/downloadCSV.js';
import {HealthCheck} from '@smart-core-os/sc-bos-ui-gen/proto/smartcore/bos/health/v1/health_pb';

const reliableState = HealthCheck.Reliability.State.RELIABLE;
const normalNormality = HealthCheck.Normality.NORMAL;

const csvHeaders = [
  'Subsystem',
  'Group',
  'Check',
  'Item ID',
  'Item Name',
  'Status',
  'Reliability',
  'Offline Since',
  'Last Error',
  'Faults'
];

/**
 * Wraps a value in quotes if it contains commas, quotes, or newlines.
 *
 * @param {string|number|null|undefined} val
 * @return {string}
 */
function csvCell(val) {
  const s = val == null ? '' : String(val);
  if (s.includes(',') || s.includes('"') || s.includes('\n')) {
    return '"' + s.replace(/"/g, '""') + '"';
  }
  return s;
}

/**
 * @param {HealthCheck.AsObject} item
 * @return {string}
 */
function itemStatus(item) {
  if ((item.reliability?.state ?? 0) > reliableState) return 'Fault';
  if ((item.normality ?? 0) > normalNormality) return 'Degraded';
  return 'OK';
}

/**
 * @param {{seconds: number, nanos: number}|undefined} ts
 * @return {string}
 */
function formatTs(ts) {
  if (!ts) return '';
  return timestampToDate(ts).toLocaleString();
}

/**
 * Downloads a CSV snapshot of all configured subsystem health checks.
 *
 * @param {Array<{
 *   title: string,
 *   icon?: string,
 *   description?: string,
 *   checks: Array<
 *     {name: string, displayName?: string} |
 *     {displayName: string, checks: Array<{name: string, displayName?: string}>}
 *   >
 * }>} subsystems
 * @return {Promise<void>}
 */
export async function downloadSubsystemHealthReport(subsystems) {
  const rows = [];

  await Promise.all(subsystems.map(async (subsystem) => {
    const subsystemTitle = subsystem.title ?? '';

    // Flatten checks into {groupLabel, leaf} pairs
    /** @type {Array<{groupLabel: string, leaf: {name: string, displayName?: string}}>} */
    const leafEntries = subsystem.checks?.flatMap(c => {
      if (c.checks) {
        return c.checks.map(leaf => ({groupLabel: c.displayName ?? '', leaf}));
      }
      return [{groupLabel: '', leaf: c}];
    }) ?? [];

    await Promise.all(leafEntries.map(async ({groupLabel, leaf}) => {
      const checkLabel = leaf.displayName ?? leaf.name;
      let healthChecks = [];
      try {
        const response = await listHealthChecks({name: leaf.name, pageSize: 1000});
        healthChecks = response.healthChecksList ?? [];
      } catch {
        // Leave healthChecks empty — still emit one row showing the check with no data
      }

      if (healthChecks.length === 0) {
        rows.push([subsystemTitle, groupLabel, checkLabel, '', '', 'No data', '', '', '', ''].map(csvCell));
        return;
      }

      for (const item of healthChecks) {
        const status = itemStatus(item);
        const isHealthy = status === 'OK';
        const faults = (item.faults?.currentFaultsList ?? [])
            .map(f => f.summaryText)
            .filter(Boolean)
            .join('; ');
        // LastError and UnreliableTime are preserved historically after recovery (by design in
        // the health check infrastructure). Suppress them when the item is currently healthy to
        // avoid stale data appearing alongside an OK/Reliable status.
        rows.push([
          subsystemTitle,
          groupLabel,
          checkLabel,
          item.id ?? '',
          item.displayName ?? '',
          status,
          reliabilityStateToString(item.reliability?.state ?? 0),
          isHealthy ? '' : formatTs(item.reliability?.unreliableTime),
          isHealthy ? '' : (item.reliability?.lastError?.summaryText ?? ''),
          faults
        ].map(csvCell));
      }
    }));
  }));

  // Sort rows by Subsystem → Group → Check for a consistent output order
  rows.sort((a, b) => {
    for (let i = 0; i < 3; i++) {
      const cmp = a[i].localeCompare(b[i]);
      if (cmp !== 0) return cmp;
    }
    return 0;
  });

  const now = new Date();
  const filename = `subsystem-health_${now.getFullYear()}-${now.getMonth() + 1}-${now.getDate()}.csv`;
  downloadCSVRows(filename, [csvHeaders, ...rows]);
}
