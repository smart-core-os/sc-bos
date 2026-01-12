import {fieldMaskFromObject, setProperties, timestampToDate} from '@/api/convpb';
import {clientOptions} from '@/api/grpcweb';
import {pullResource, setValue, trackAction} from '@/api/resource';
import {periodFromObject} from '@/api/sc/types/period';
import {ElectricHistoryPromiseClient} from '@smart-core-os/sc-bos-ui-gen/proto/smartcore/bos/electric/v1/electric_history_grpc_web_pb';
import {ListElectricDemandHistoryRequest} from '@smart-core-os/sc-bos-ui-gen/proto/smartcore/bos/electric/v1/electric_history_pb';
import {ElectricApiPromiseClient} from '@smart-core-os/sc-api-grpc-web/traits/electric_grpc_web_pb';
import {GetDemandRequest, PullDemandRequest} from '@smart-core-os/sc-api-grpc-web/traits/electric_pb';

/**
 * @param {Partial<PullDemandRequest.AsObject>} request
 * @param {ResourceValue<ElectricDemand.AsObject, PullDemandResponse>} resource
 */
export function pullDemand(request, resource) {
  pullResource('Electric.pullDemand', resource, endpoint => {
    const api = apiClient(endpoint);
    const stream = api.pullDemand(pullDemandRequestFromObject(request));
    stream.on('data', msg => {
      const changes = msg.getChangesList();
      for (const change of changes) {
        setValue(resource, change.getDemand().toObject());
      }
    });
    return stream;
  });
}

/**
 * @param {Partial<GetDemandRequest.AsObject>} request
 * @param {ActionTracker<ElectricDemand.AsObject>} [tracker]
 * @return {Promise<ElectricDemand.AsObject>}
 */
export function getDemand(request, tracker) {
  return trackAction('Electric.getDemand', tracker ?? {}, endpoint => {
    const api = new ElectricApiPromiseClient(endpoint, null, clientOptions());
    return api.getDemand(getDemandRequestFromObject(request));
  });
}

/**
 * @param {Partial<ListElectricDemandHistoryRequest.AsObject>} request
 * @param {ActionTracker<ListElectricDemandHistoryResponse.AsObject>} [tracker]
 * @return {Promise<ListElectricDemandHistoryResponse.AsObject>}
 */
export function listElectricDemandHistory(request, tracker = {}) {
  return trackAction('ElectricDemandHistory.listElectricDemandHistory', tracker, endpoint => {
    const api = historyClient(endpoint);
    return api.listElectricDemandHistory(listElectricDemandHistoryRequestFromObject(request));
  });
}

/**
 * @param {ElectricDemandRecord | ElectricDemandRecord.AsObject} obj
 * @return {ElectricDemandRecord.AsObject & {recordTime: Date|undefined}}
 */
export function electricDemandRecordToObject(obj) {
  if (!obj) return undefined;
  if (typeof obj.toObject === 'function') obj = obj.toObject();
  if (obj.recordTime) obj.recordTime = timestampToDate(obj.recordTime);
  return obj;
}

/**
 * @param {string} endpoint
 * @return {ElectricApiPromiseClient}
 */
function apiClient(endpoint) {
  return new ElectricApiPromiseClient(endpoint, null, clientOptions());
}

/**
 * @param {string} endpoint
 * @return {ElectricHistoryPromiseClient}
 */
function historyClient(endpoint) {
  return new ElectricHistoryPromiseClient(endpoint, null, clientOptions());
}

/**
 * @param {Partial<PullDemandRequest.AsObject>} obj
 * @return {PullDemandRequest|undefined}
 */
function pullDemandRequestFromObject(obj) {
  if (!obj) return undefined;

  const dst = new PullDemandRequest();
  setProperties(dst, obj, 'name', 'updatesOnly');
  dst.setReadMask(fieldMaskFromObject(obj.readMask));
  return dst;
}

/**
 * @param {Partial<GetDemandRequest.AsObject>} obj
 * @return {undefined|GetDemandRequest}
 */
function getDemandRequestFromObject(obj) {
  if (!obj) return undefined;

  const dst = new GetDemandRequest();
  setProperties(dst, obj, 'name');
  dst.setReadMask(fieldMaskFromObject(obj.readMask));
  return dst;
}

/**
 * @param {Partial<ListElectricDemandHistoryRequest.AsObject>} obj
 * @return {ListElectricDemandHistoryRequest|undefined}
 */
function listElectricDemandHistoryRequestFromObject(obj) {
  if (!obj) return undefined;
  const dst = new ListElectricDemandHistoryRequest();
  setProperties(dst, obj, 'name', 'pageToken', 'pageSize', 'orderBy');
  dst.setReadMask(fieldMaskFromObject(obj.readMask));
  dst.setPeriod(periodFromObject(obj.period));
  return dst;
}
