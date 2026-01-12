import {fieldMaskFromObject, setProperties} from '@/api/convpb.js';
import {clientOptions} from '@/api/grpcweb.js';
import {pullResource, setCollection, trackAction} from '@/api/resource.js';
import {WasteApiPromiseClient} from '@smart-core-os/sc-bos-ui-gen/proto/smartcore/bos/waste/v1/waste_grpc_web_pb';
import {ListWasteRecordsRequest, PullWasteRecordsRequest} from '@smart-core-os/sc-bos-ui-gen/proto/smartcore/bos/waste/v1/waste_pb';

/**
 * @param {Partial<ListWasteRecordsRequest.AsObject>} request
 * @param {ActionTracker<ListWasteRecordsResponse.AsObject>} [tracker]
 * @return {Promise<ListWasteRecordsResponse.AsObject>}
 */
export function listWasteRecords(request, tracker) {
  return trackAction('Waste.listWasteRecords', tracker ?? {}, endpoint => {
    const api = apiClient(endpoint);
    return api.listWasteRecords(listWasteRecordsRequestFromObject(request));
  });
}

/**
 * @param {Partial<PullWasteRecordsRequest.AsObject>} request
 * @param {ResourceCollection<WasteRecord.AsObject, PullWasteRecordsResponse>} resource
 */
export function pullWasteRecords(request, resource) {
  pullResource('Waste.pullWasteRecords', resource, endpoint => {
    const api = apiClient(endpoint);
    const stream = api.pullWasteRecords(pullWasteRecordsRequestFromObject(request));
    stream.on('data', msg => {
      const changes = msg.getChangesList();
      for (const change of changes) {
        setCollection(resource, change, v => v.id);
      }
    });
    return stream;
  });
}

/**
 * @param {string} endpoint
 * @return {WasteApiPromiseClient}
 */
function apiClient(endpoint) {
  return new WasteApiPromiseClient(endpoint, clientOptions());
}

/**
 * @param {Partial<ListWasteRecordsRequest.AsObject>} obj
 * @return {ListWasteRecordsRequest}
 */
function listWasteRecordsRequestFromObject(obj) {
  if (!obj) return undefined;
  const dst = new ListWasteRecordsRequest();
  setProperties(dst, obj, 'name', 'pageToken', 'pageSize');
  dst.setReadMask(fieldMaskFromObject(obj.readMask));
  return dst;
}

/**
 * @param {Partial<PullWasteRecordsRequest.AsObject>} obj
 * @return {PullWasteRecordsRequest|undefined}
 */
function pullWasteRecordsRequestFromObject(obj) {
  if (!obj) return undefined;
  const dst = new PullWasteRecordsRequest();
  setProperties(dst, obj, 'name', 'updatesOnly');
  dst.setReadMask(fieldMaskFromObject(obj.readMask));
  return dst;
}
