import {fieldMaskFromObject, setProperties, timestampToDate} from '@/api/convpb.js';
import {clientOptions} from '@/api/grpcweb.js';
import {pullResource, setValue, trackAction} from '@/api/resource.js';
import {periodFromObject} from '@/api/sc/types/period.js';
import {AllocationApiPromiseClient} from '@smart-core-os/sc-bos-ui-gen/proto/allocation_grpc_web_pb.d.ts';
import {ListAllocationHistoryRequest, PullAllocationsRequest} from '@smart-core-os/sc-bos-ui-gen/proto/allocation_pb.d.ts';
import {AllocationHistoryPromiseClient} from '@smart-core-os/sc-bos-ui-gen/proto/allocation_grpc_web_pb.d.ts';
/**
 * @param {Partial<PullAllocationsRequest.AsObject>} request
 * @param {ResourceValue<Allocation.AsObject>} resource
 */
export function pullAllocations(request, resource) {
  pullResource('Allocation.pullAllocations', resource, endpoint => {
    const api = apiClient(endpoint);
    const stream = api.pullAllocations(pullAllocationsRequestFromObject(request));
    stream.on('data', msg => {
      const changes = msg.getChangesList();
      for (const change of changes) {
        setValue(resource, change.getAllocation().toObject());
      }
    });
    return stream;
  });
}

export function listAllocationsHistory(request, tracker) {
  return trackAction('Allocation.listAllocationsHistory', tracker ?? {}, (endpoint) => {
    const api = historyClient(endpoint);
    return api.listAllocationHistory(listAllocationHistoryRequestFromObject(request));
  });
}

/**
 * @param {AllocationRecord | AllocationRecord.AsObject} obj
 * @returns {AllocationRecord.AsObject & {recordTime: Date|undefined}}
 */
export function allocationRecordToObject(obj) {
  if (!obj) return undefined;
  if (typeof obj.toObject === 'function') obj = obj.toObject();
  if (obj.recordTime) obj.recordTime = timestampToDate(obj.recordTime);
  return obj;
}

/**
 * @param {string} endpoint
 * @return {AllocationApiPromiseClient}
 */
function apiClient(endpoint) {
  return new AllocationApiPromiseClient(endpoint, null, clientOptions());
}

/**
 * @param {string} endpoint
 * @return {AllocationHistoryPromiseClient}
 */
function historyClient(endpoint) {
  return new AllocationHistoryPromiseClient(endpoint, null, clientOptions());
}

/**
 * @param obj
 * @return {PullAllocationsRequest|undefined}
 */
function pullAllocationsRequestFromObject(obj) {
  if (!obj) return undefined;
  const dst = new PullAllocationsRequest();
  setProperties(dst, obj, 'name', 'updatesOnly');
  dst.setReadMask(fieldMaskFromObject(obj.readMask));
  return dst;
}

/**
 * @param {Partial<ListAllocationHistoryRequest.AsObject>} obj
 * @return {ListAllocationHistoryRequest|undefined}
 */
function listAllocationHistoryRequestFromObject(obj) {
  if (!obj) return undefined;
  const dst = new ListAllocationHistoryRequest();
  setProperties(dst, obj, 'name', 'pageSize', 'pageToken', 'orderBy');
  dst.setReadMask(fieldMaskFromObject(obj.readMask));
  dst.setPeriod(periodFromObject(obj.period));
  return dst;
}