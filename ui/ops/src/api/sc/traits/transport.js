import {fieldMaskFromObject, setProperties} from '@/api/convpb.js';
import {clientOptions} from '@/api/grpcweb.js';
import {pullResource, setValue, trackAction} from '@/api/resource.js';
import {periodFromObject} from '@/api/sc/types/period.js';
import {TransportApiPromiseClient, TransportInfoPromiseClient} from '@smart-core-os/sc-bos-ui-gen/proto/smartcore/bos/transport/v1/transport_grpc_web_pb';
import {TransportHistoryPromiseClient} from '@smart-core-os/sc-bos-ui-gen/proto/smartcore/bos/transport/v1/transport_history_grpc_web_pb';
import {DescribeTransportRequest, PullTransportRequest} from '@smart-core-os/sc-bos-ui-gen/proto/smartcore/bos/transport/v1/transport_pb';
import {ListTransportHistoryRequest} from '@smart-core-os/sc-bos-ui-gen/proto/smartcore/bos/transport/v1/transport_history_pb';

/**
 * @param {Partial<PullTransportRequest.AsObject>} request
 * @param {ResourceValue<Transport.AsObject, PullTransportResponse>} resource
 */
export function pullTransport(request, resource) {
  pullResource('Transport.pullTransport', resource, endpoint => {
    const api = apiClient(endpoint);
    const stream = api.pullTransport(pullTransportRequestFromObject(request));
    stream.on('data', msg => {
      const changes = msg.getChangesList();
      for (const change of changes) {
        setValue(resource, change.getTransport().toObject());
      }
    });
    return stream;
  });
}

/**
 *
 * @param {Partial<DescribeTransportRequest.AsObject>} request
 * @param {ActionTracker<TransportSupport.AsObject>} [tracker]
 * @return {Promise<TransportSupport.AsObject>}
 */
export function describeTransport(request, tracker) {
  return trackAction('Transport.describeTransport', tracker ?? {}, endpoint => {
    const api = infoClient(endpoint);
    return api.describeTransport(describeTransportRequestFromObject(request));
  });
}

/**
 *
 * @param {Partial<ListTransportHistoryRequest.AsObject>} request
 * @param {ActionTracker<ListTransportHistoryResponse.AsObject>} [tracker]
 * @return {Promise<ListTransportHistoryResponse.AsObject>}
 */
export function listTransportHistory(request, tracker) {
  return trackAction('TransportHistory.listTransportHistory', tracker ?? {}, endpoint => {
    const api = historyClient(endpoint);
    return api.listTransportHistory(listTransportHistoryRequestFromObject(request));
  });
}

/**
 * @param {string} endpoint
 * @return {TransportApiPromiseClient}
 */
function apiClient(endpoint) {
  return new TransportApiPromiseClient(endpoint, null, clientOptions());
}

/**
 * @param {string} endpoint
 * @return {TransportInfoPromiseClient}
 */
function infoClient(endpoint) {
  return new TransportInfoPromiseClient(endpoint, null, clientOptions());
}

/**
 *
 * @param {string} endpoint
 * @return {TransportHistoryPromiseClient}
 */
function historyClient(endpoint) {
  return new TransportHistoryPromiseClient(endpoint, null, clientOptions());
}

/**
 * @param {Partial<DescribeTransportRequest.AsObject>} obj
 * @return {undefined|DescribeTransportRequest}
 */
function describeTransportRequestFromObject(obj) {
  if (!obj) return undefined;
  const dst = new DescribeTransportRequest();
  setProperties(dst, obj, 'name');
  return dst;
}


/**
 * @param {Partial<PullTransportRequest.AsObject>} obj
 * @return {PullTransportRequest|undefined}
 */
function pullTransportRequestFromObject(obj) {
  if (!obj) return undefined;

  const dst = new PullTransportRequest();
  setProperties(dst, obj, 'name', 'updatesOnly');
  dst.setReadMask(fieldMaskFromObject(obj.readMask));
  return dst;
}

/**
 * @param {Partial<ListTransportHistoryRequest.AsObject>} obj
 * @return {ListTransportHistoryRequest|undefined}
 */
function listTransportHistoryRequestFromObject(obj) {
  if (!obj) return undefined;
  const dst = new ListTransportHistoryRequest();
  setProperties(dst, obj, 'name', 'pageToken', 'pageSize', 'orderBy');
  dst.setReadMask(fieldMaskFromObject(obj.readMask));
  dst.setPeriod(periodFromObject(obj.period));
  return dst;
}
