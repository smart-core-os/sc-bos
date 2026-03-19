import {fieldMaskFromObject, setProperties} from '@/api/convpb';
import {clientOptions} from '@/api/grpcweb';
import {pullResource, setValue, trackAction} from '@/api/resource';
import {StorageApiPromiseClient, StorageInfoPromiseClient} from
  '@smart-core-os/sc-bos-ui-gen/proto/smartcore/bos/storage/v1/storage_grpc_web_pb';
import {DescribeStorageRequest, PullStorageRequest} from
  '@smart-core-os/sc-bos-ui-gen/proto/smartcore/bos/storage/v1/storage_pb';


/**
 * @param {Partial<PullStorageRequest.AsObject>} request
 * @param {ResourceValue<Storage.AsObject, PullStorageResponse>} resource
 */
export function pullStorage(request, resource) {
  pullResource('StorageApi.pullStorage', resource, endpoint => {
    const api = apiClient(endpoint);
    const stream = api.pullStorage(pullRequestFromObject(request));
    stream.on('data', msg => {
      for (const change of msg.getChangesList()) {
        setValue(resource, change.getStorage().toObject());
      }
    });
    return stream;
  });
}

/**
 * @param {Partial<DescribeStorageRequest.AsObject>} request
 * @param {ActionTracker<StorageSupport.AsObject>} [tracker]
 * @return {Promise<StorageSupport.AsObject>}
 */
export function describeStorage(request, tracker) {
  return trackAction('StorageInfo.describeStorage', tracker ?? {}, endpoint => {
    return infoClient(endpoint).describeStorage(describeRequestFromObject(request));
  });
}

/**
 * @param {string} endpoint
 * @return {StorageApiPromiseClient}
 */
function apiClient(endpoint) {
  return new StorageApiPromiseClient(endpoint, null, clientOptions());
}

/**
 * @param {string} endpoint
 * @return {StorageInfoPromiseClient}
 */
function infoClient(endpoint) {
  return new StorageInfoPromiseClient(endpoint, null, clientOptions());
}

/**
 * @param {Partial<PullStorageRequest.AsObject>} obj
 * @return {PullStorageRequest|undefined}
 */
function pullRequestFromObject(obj) {
  if (!obj) return undefined;
  const dst = new PullStorageRequest();
  setProperties(dst, obj, 'name', 'updatesOnly');
  dst.setReadMask(fieldMaskFromObject(obj.readMask));
  return dst;
}

/**
 * @param {Partial<DescribeStorageRequest.AsObject>} obj
 * @return {DescribeStorageRequest|undefined}
 */
function describeRequestFromObject(obj) {
  if (!obj) return undefined;
  const dst = new DescribeStorageRequest();
  setProperties(dst, obj, 'name');
  return dst;
}
