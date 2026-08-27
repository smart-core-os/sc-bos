import {trackAction} from '@/api/resource.js';
import {clientOptions} from '@/api/grpcweb.js';
import {timestampToDate} from '@/api/convpb.js';
import {MeterApiPromiseClient} from '@smart-core-os/sc-bos-ui-gen/proto/smartcore/bos/meter/v1/meter_grpc_web_pb';
import {MeterHistoryPromiseClient} from '@smart-core-os/sc-bos-ui-gen/proto/smartcore/bos/meter/v1/meter_history_grpc_web_pb';
import {GetMeterReadingRequest} from '@smart-core-os/sc-bos-ui-gen/proto/smartcore/bos/meter/v1/meter_pb';
import {ListMeterReadingHistoryRequest} from '@smart-core-os/sc-bos-ui-gen/proto/smartcore/bos/meter/v1/meter_history_pb';
import {periodFromObject} from '@/api/sc/types/period.js';

/**
 * Fetch the current meter reading for the given device.
 *
 * @param {string} name
 * @return {Promise<{usage: number}>}
 */
export async function getMeterReading(name) {
  const tracker = {loading: false, response: null, error: null};
  const req = new GetMeterReadingRequest();
  req.setName(name);
  return trackAction('getMeterReading', tracker, (endpoint) => {
    const client = new MeterApiPromiseClient(endpoint, null, clientOptions());
    return client.getMeterReading(req);
  });
}

/**
 * Fetch the first history record within the given period.
 * Used to retrieve the midnight baseline reading.
 *
 * @param {string} name
 * @param {Date} startTime
 * @param {Date} endTime
 * @return {Promise<{meterReading?: {usage: number}}|null>}
 */
export async function getFirstMeterReadingInPeriod(name, startTime, endTime) {
  const tracker = {loading: false, response: null, error: null};
  const req = new ListMeterReadingHistoryRequest();
  req.setName(name);
  req.setPeriod(periodFromObject({startTime, endTime}));
  req.setPageSize(1);
  return trackAction('listMeterReadingHistory', tracker, (endpoint) => {
    const client = new MeterHistoryPromiseClient(endpoint, null, clientOptions());
    return client.listMeterReadingHistory(req);
  }).then(res => res.meterReadingRecordsList?.[0] ?? null);
}

/**
 * A meter reading sample: a cumulative accumulator value and when it was taken.
 *
 * `at` matters as much as `usage`. It is what tells a boundary whether a reading
 * sits on it or a fortnight before it, what bounds the stretch reported as
 * unrecorded, and what a forward projection measures its rate over — and a
 * boundary read that landed 40 hours late is a different fact from one that
 * landed on the boundary.
 *
 * @typedef {{usage: number, at: Date}} MeterSample
 */

/**
 * A run of consecutive records from one end of a window.
 *
 * The sort decides which end, and `limit` how many, so this is one query however
 * much history the window spans. Asking for more than one record costs nothing
 * extra in queries — only a slightly larger response — and a run of consecutive
 * readings is what measuring a meter's resolution needs.
 *
 * @param {string} name
 * @param {Date} startTime
 * @param {Date} endTime
 * @param {'asc'|'desc'} order which end of the window to take from
 * @param {number} limit
 * @return {Promise<MeterSample[]>} ascending by time; empty when nothing matched
 */
async function readingsAtEdge(name, startTime, endTime, order, limit) {
  const tracker = {loading: false, response: null, error: null};
  const req = new ListMeterReadingHistoryRequest();
  req.setName(name);
  req.setPeriod(periodFromObject({startTime, endTime}));
  req.setPageSize(Math.max(1, limit));
  req.setOrderBy(`record_time ${order}`);
  const res = await trackAction('listMeterReadingHistory', tracker, (endpoint) => {
    const client = new MeterHistoryPromiseClient(endpoint, null, clientOptions());
    return client.listMeterReadingHistory(req);
  });

  return (res.meterReadingRecordsList ?? [])
    .filter(rec => rec?.meterReading?.usage != null && rec?.recordTime)
    .map(rec => ({usage: rec.meterReading.usage, at: timestampToDate(rec.recordTime)}))
    // Always ascending, whichever direction the server sorted, so callers never
    // have to care which end of the window a run came from.
    .sort((a, b) => a.at.getTime() - b.at.getTime());
}

/**
 * @param {string} name
 * @param {Date} startTime
 * @param {Date} endTime
 * @param {'asc'|'desc'} order
 * @return {Promise<MeterSample|null>}
 */
async function readingAtEdge(name, startTime, endTime, order) {
  const run = await readingsAtEdge(name, startTime, endTime, order, 1);
  return run.length ? run[0] : null;
}

/**
 * The last `limit` readings at or before `at`, within `windowDays` of it.
 *
 * The nearest reading is the run's *last* element, since runs are ascending.
 *
 * @param {string} name
 * @param {Date} at
 * @param {number} windowDays
 * @param {number} limit
 * @return {Promise<MeterSample[]>} ascending by time
 */
export function getMeterReadingsBefore(name, at, windowDays, limit) {
  const from = new Date(at.getTime() - windowDays * 24 * 60 * 60 * 1000);
  return readingsAtEdge(name, from, at, 'desc', limit);
}

/**
 * The earliest reading at or after `at`, within `windowDays` of it.
 *
 * @param {string} name
 * @param {Date} at
 * @param {number} windowDays how far forward to look before giving up
 * @return {Promise<MeterSample|null>}
 */
export function getMeterReadingAfter(name, at, windowDays) {
  const to = new Date(at.getTime() + windowDays * 24 * 60 * 60 * 1000);
  return readingAtEdge(name, at, to, 'asc');
}
