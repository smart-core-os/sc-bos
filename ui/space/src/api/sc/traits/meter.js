import {fieldMaskFromObject, setProperties} from '@/api/convpb';
import {clientOptions} from '@/api/grpcweb';
import {pullResource, setValue, trackAction} from '@/api/resource.js';
import {periodFromObject} from '@/api/sc/types/period';
import {MeterHistoryPromiseClient} from '@smart-core-os/sc-bos-ui-gen/proto/meter_history_grpc_web_pb';
import {ListMeterReadingHistoryRequest} from '@smart-core-os/sc-bos-ui-gen/proto/meter_history_pb';
import {MeterApiPromiseClient, MeterInfoPromiseClient} from '@smart-core-os/sc-bos-ui-gen/proto/meter_grpc_web_pb';
import {DescribeMeterReadingRequest, PullMeterReadingsRequest} from '@smart-core-os/sc-bos-ui-gen/proto/meter_pb';

/**
 * @param {Partial<PullMeterReadingsRequest.AsObject>} request
 * @param {ResourceValue<MeterReading.AsObject, PullMeterReadingsResponse>} resource
 */
export function pullMeterReading(request, resource) {
  pullResource('MeterApi.pullMeterReadings', resource, (endpoint) => {
    const api = apiClient(endpoint);
    const stream = api.pullMeterReadings(pullMeterReadingsRequestFromObject(request));
    stream.on('data', (msg) => {
      const changes = msg.getChangesList();
      for (const change of changes) {
        setValue(resource, change.getMeterReading().toObject());
      }
    });
    return stream;
  });
}

/**
 *
 * @param {Partial<DescribeMeterReadingRequest.AsObject>} request
 * @param {ActionTracker<MeterReadingSupport.AsObject>} [tracker]
 * @return {Promise<MeterReadingSupport.AsObject>}
 */
export function describeMeterReading(request, tracker) {
  return trackAction('MeterInfo.DescribeMeterReading', tracker ?? {}, (endpoint) => {
    const api = infoClient(endpoint);
    return api.describeMeterReading(describeMeterReadingRequestFromObject(request));
  });
}

/**
 *
 * @param {Partial<ListMeterReadingHistoryRequest.AsObject>} request
 * @param {ActionTracker<ListMeterReadingHistoryResponse.AsObject>} [tracker]
 * @return {Promise<ListMeterReadingHistoryResponse.AsObject>}
 */
export function listMeterReadingHistory(request, tracker) {
  return trackAction('MeterReadingHistory.listMeterReadingHistory', tracker, endpoint => {
    const api = historyClient(endpoint);
    return api.listMeterReadingHistory(listMeterReadingHistoryRequestFromObject(request));
  });
}


/**
 *
 * @param {string} endpoint
 * @return {MeterHistoryPromiseClient}
 */
function historyClient(endpoint) {
  return new MeterHistoryPromiseClient(endpoint, null, clientOptions());
}

/**
 * @param {string} endpoint
 * @return {MeterApiPromiseClient}
 */
function apiClient(endpoint) {
  return new MeterApiPromiseClient(endpoint, null, clientOptions());
}


/**
 * @param {string} endpoint
 * @return {MeterInfoPromiseClient}
 */
function infoClient(endpoint) {
  return new MeterInfoPromiseClient(endpoint, null, clientOptions());
}

/**
 * @param {Partial<PullMeterReadingsRequest.AsObject>} obj
 * @return {PullMeterReadingsRequest|undefined}
 */
function pullMeterReadingsRequestFromObject(obj) {
  if (!obj) return undefined;
  const dst = new PullMeterReadingsRequest();
  setProperties(dst, obj, 'name', 'updatesOnly');
  dst.setReadMask(fieldMaskFromObject(obj.readMask));
  return dst;
}

/**
 * @param {Partial<DescribeMeterReadingRequest.AsObject>} obj
 * @return {undefined|DescribeMeterReadingRequest}
 */
function describeMeterReadingRequestFromObject(obj) {
  if (!obj) return undefined;
  const dst = new DescribeMeterReadingRequest();
  setProperties(dst, obj, 'name');
  return dst;
}

/**
 * @param {Partial<ListMeterReadingHistoryRequest.AsObject>} obj
 * @return {ListMeterReadingHistoryRequest|undefined}
 */
function listMeterReadingHistoryRequestFromObject(obj) {
  if (!obj) return undefined;
  const dst = new ListMeterReadingHistoryRequest();
  setProperties(dst, obj, 'name', 'pageToken', 'pageSize');
  dst.setReadMask(fieldMaskFromObject(obj.readMask));
  dst.setPeriod(periodFromObject(obj.period));
  return dst;
}
