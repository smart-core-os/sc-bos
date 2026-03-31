import {fieldMaskFromObject, setProperties} from '@/api/convpb';
import {clientOptions} from '@/api/grpcweb';
import {pullResource, setValue, trackAction} from '@/api/resource';
import {ResourceUseApiPromiseClient} from '@smart-core-os/sc-bos-ui-gen/proto/smartcore/bos/resourceuse/v1/resource_use_grpc_web_pb';
import {GetResourceUseRequest, PullResourceUseRequest} from '@smart-core-os/sc-bos-ui-gen/proto/smartcore/bos/resourceuse/v1/resource_use_pb';


/**
 * @param {Partial<PullResourceUseRequest.AsObject>} request
 * @param {ResourceValue<ResourceUse.AsObject, PullResourceUseResponse>} resource
 */
export function pullResourceUse(request, resource) {
  pullResource('ResourceUseApi.pullResourceUse', resource, endpoint => {
    const api = apiClient(endpoint);
    const stream = api.pullResourceUse(pullRequestFromObject(request));
    stream.on('data', msg => {
      const changes = msg.getChangesList();
      for (const change of changes) {
        setValue(resource, change.getResourceUse().toObject());
      }
    });
    return stream;
  });
}

/**
 * @param {Partial<GetResourceUseRequest.AsObject>} request
 * @param {ActionTracker<ResourceUse.AsObject>} [tracker]
 * @return {Promise<ResourceUse.AsObject>}
 */
export function getResourceUse(request, tracker) {
  return trackAction('ResourceUseApi.getResourceUse', tracker ?? {}, endpoint => {
    const api = apiClient(endpoint);
    return api.getResourceUse(getRequestFromObject(request));
  });
}

/**
 * @param {string} endpoint
 * @return {ResourceUseApiPromiseClient}
 */
function apiClient(endpoint) {
  return new ResourceUseApiPromiseClient(endpoint, null, clientOptions());
}

/**
 * @param {Partial<PullResourceUseRequest.AsObject>} obj
 * @return {PullResourceUseRequest|undefined}
 */
function pullRequestFromObject(obj) {
  if (!obj) return undefined;
  const dst = new PullResourceUseRequest();
  setProperties(dst, obj, 'name', 'updatesOnly');
  dst.setReadMask(fieldMaskFromObject(obj.readMask));
  return dst;
}

/**
 * @param {Partial<GetResourceUseRequest.AsObject>} obj
 * @return {GetResourceUseRequest|undefined}
 */
function getRequestFromObject(obj) {
  if (!obj) return undefined;
  const dst = new GetResourceUseRequest();
  setProperties(dst, obj, 'name');
  dst.setReadMask(fieldMaskFromObject(obj.readMask));
  return dst;
}
