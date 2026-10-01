import {timestampToDate} from '@/api/convpb.js';
import {
  equipmentImpactToString,
  normalityToString,
  occupantImpactToString,
  reliabilityStateToString
} from '@/api/sc/traits/health.js';
import {listDevices} from '@/api/ui/devices.js';
import {countChecks} from '@/traits/health/health.js';
import {dateStamp} from '@/util/date.js';
import {downloadCSVRows} from '@/util/downloadCSV.js';
import {HealthCheck} from '@smart-core-os/sc-bos-ui-gen/proto/smartcore/bos/health/v1/health_pb';
import {format} from 'date-fns';

/**
 * Formats a protobuf Timestamp as an ISO-8601 instant, or '' if absent.
 *
 * ISO-8601 sorts correctly as text in a spreadsheet, where toLocaleString() does not.
 *
 * @param {Timestamp.AsObject | undefined} ts
 * @return {string}
 */
function formatTs(ts) {
  const d = timestampToDate(ts);
  if (!d) return '';
  return format(d, "yyyy-MM-dd'T'HH:mm:ssxxx");
}

/**
 * @param {HealthCheck.AsObject} check
 * @return {boolean} whether the check's connection is currently good
 */
function isReliable(check) {
  return check?.reliability?.state === HealthCheck.Reliability.State.RELIABLE;
}

/**
 * The human-readable label for a check, matching HealthCheckRow.
 *
 * @param {HealthCheck.AsObject} check
 * @return {string}
 */
function checkLabel(check) {
  return check?.displayName || check?.id || '';
}

/**
 * Free-text explanation of why a check is unhappy: its current faults, else the last
 * connection error.
 *
 * Mirrors HealthCheckRow's Value cell minus the bounds branch - Device.health_checks omits
 * measured values by design, so we have no bounds figures to show. lastError is retained
 * after recovery by the health check infrastructure, so it's suppressed once the check reads
 * RELIABLE to keep stale text out of an otherwise healthy row.
 *
 * @param {HealthCheck.AsObject} check
 * @return {string}
 */
function checkDetail(check) {
  const faults = (check?.faults?.currentFaultsList ?? [])
      .map(f => f.summaryText)
      .filter(Boolean)
      .join('; ');
  if (faults) return faults;
  if (isReliable(check)) return '';
  return check?.reliability?.lastError?.summaryText ?? '';
}

/**
 * The CSV columns, one row per (device, health check) pair.
 *
 * Cell values are returned raw; downloadCSVRows handles RFC 4180 quoting and formula-injection
 * escaping, so nothing here should escape its own output.
 *
 * @type {Array<{title: string, val: function(Device.AsObject, HealthCheck.AsObject): string}>}
 */
export const healthCheckCsvColumns = [
  {title: 'Device', val: (d) => d.metadata?.appearance?.title || d.name || ''},
  {title: 'Device name', val: (d) => d.name ?? ''},
  {title: 'Floor', val: (d) => d.metadata?.location?.floor ?? ''},
  {title: 'Zone', val: (d) => d.metadata?.location?.zone ?? ''},
  {title: 'Subsystem', val: (d) => d.metadata?.membership?.subsystem ?? ''},
  {title: 'Check', val: (d, c) => checkLabel(c)},
  {title: 'Description', val: (d, c) => c.description ?? ''},
  {title: 'Health', val: (d, c) => normalityToString(c.normality ?? 0)},
  {
    title: 'Health since',
    val: (d, c) => formatTs(c.normality === HealthCheck.Normality.NORMAL ? c.normalTime : c.abnormalTime)
  },
  {title: 'Connection', val: (d, c) => reliabilityStateToString(c.reliability?.state ?? 0)},
  {
    title: 'Connection since',
    val: (d, c) => formatTs(isReliable(c) ? c.reliability?.reliableTime : c.reliability?.unreliableTime)
  },
  {title: 'Deviation', val: (d, c) => c.deviation ? String(c.deviation) : ''},
  {title: 'Occupant impact', val: (d, c) => occupantImpactToString(c.occupantImpact ?? 0)},
  {title: 'Equipment impact', val: (d, c) => equipmentImpactToString(c.equipmentImpact ?? 0)},
  {title: 'Detail', val: (d, c) => checkDetail(c)},
  {title: 'Device issues', val: (d) => String(countChecks(d.healthChecksList)?.abnormalCount ?? 0)},
  {title: 'Device checks', val: (d) => String(countChecks(d.healthChecksList)?.totalCount ?? 0)}
];

/**
 * Lays the given devices out as CSV rows (header first): one row per health check.
 *
 * Every check of a matching device is emitted, normal ones included, because that's what the
 * table's expanded row shows - a device-level row saying "Abnormal since 3d" doesn't tell an
 * engineer which check failed. Rows are sorted by device then check so the output order is
 * stable regardless of the order pages arrived in.
 *
 * @param {Device.AsObject[]} devices
 * @return {string[][]}
 */
export function buildHealthCheckCsv(devices) {
  const rows = [];
  for (const device of devices ?? []) {
    for (const check of device.healthChecksList ?? []) {
      rows.push(healthCheckCsvColumns.map(col => col.val(device, check)));
    }
  }
  rows.sort((a, b) => {
    // column 1 is the fully-qualified device name, column 5 the check label
    return a[1].localeCompare(b[1]) || a[5].localeCompare(b[5]);
  });
  return [healthCheckCsvColumns.map(col => col.title), ...rows];
}

/**
 * Downloads a CSV of the health checks belonging to every device matching the given query.
 *
 * Pages the server directly rather than reading the table's collection, which only holds the
 * pages you've visited - exporting "what's loaded" would silently emit 20 rows out of hundreds.
 *
 * Reports rowCount 0 without downloading anything when nothing matched, so the caller can say
 * so rather than handing over a header-only file. A failure part-way through still downloads
 * what was collected and returns the error, so the caller can warn the file is incomplete.
 *
 * @param {Device.Query.AsObject} query
 * @return {Promise<{rowCount: number, error: *}>}
 */
export async function downloadHealthCheckCsv(query) {
  const devices = [];
  let error = null;
  try {
    let pageToken = undefined;
    do {
      const page = await listDevices({query, pageSize: 1000, pageToken});
      devices.push(...(page.devicesList ?? []));
      pageToken = page.nextPageToken;
    } while (pageToken);
  } catch (err) {
    error = err;
  }

  const rows = buildHealthCheckCsv(devices);
  // rows always contains the header row; a length of 1 means nothing was exported.
  if (rows.length <= 1) return {rowCount: 0, error};
  downloadCSVRows(`health-checks - ${dateStamp()}.csv`, rows);
  return {rowCount: rows.length - 1, error};
}
