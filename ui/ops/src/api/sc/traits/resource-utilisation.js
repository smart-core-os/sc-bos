import {fieldMaskFromObject, setProperties} from '@/api/convpb';
import {clientOptions} from '@/api/grpcweb';
import {pullResource, setValue, trackAction} from '@/api/resource';
import {ResourceUtilisationApiPromiseClient} from '@smart-core-os/sc-bos-ui-gen/proto/smartcore/bos/resourceutilisation/v1/resource_utilisation_grpc_web_pb';
import {GetResourceUtilisationRequest, PullResourceUtilisationRequest} from '@smart-core-os/sc-bos-ui-gen/proto/smartcore/bos/resourceutilisation/v1/resource_utilisation_pb';


/**
 * @param {Partial<PullResourceUtilisationRequest.AsObject>} request
 * @param {ResourceValue<ResourceUtilisation.AsObject, PullResourceUtilisationResponse>} resource
 */
export function pullResourceUtilisation(request, resource) {
  pullResource('ResourceUtilisationApi.pullResourceUtilisation', resource, endpoint => {
    const api = apiClient(endpoint);
    const stream = api.pullResourceUtilisation(pullRequestFromObject(request));
    stream.on('data', msg => {
      const changes = msg.getChangesList();
      for (const change of changes) {
        setValue(resource, change.getResourceUtilisation().toObject());
      }
    });
    return stream;
  });
}

/**
 * @param {Partial<GetResourceUtilisationRequest.AsObject>} request
 * @param {ActionTracker<ResourceUtilisation.AsObject>} [tracker]
 * @return {Promise<ResourceUtilisation.AsObject>}
 */
export function getResourceUtilisation(request, tracker) {
  return trackAction('ResourceUtilisationApi.getResourceUtilisation', tracker ?? {}, endpoint => {
    const api = apiClient(endpoint);
    return api.getResourceUtilisation(getRequestFromObject(request));
  });
}

/**
 * @param {string} endpoint
 * @return {ResourceUtilisationApiPromiseClient}
 */
function apiClient(endpoint) {
  return new ResourceUtilisationApiPromiseClient(endpoint, null, clientOptions());
}

/**
 * @param {Partial<PullResourceUtilisationRequest.AsObject>} obj
 * @return {PullResourceUtilisationRequest|undefined}
 */
function pullRequestFromObject(obj) {
  if (!obj) return undefined;
  const dst = new PullResourceUtilisationRequest();
  setProperties(dst, obj, 'name', 'updatesOnly');
  dst.setReadMask(fieldMaskFromObject(obj.readMask));
  return dst;
}

/**
 * @param {Partial<GetResourceUtilisationRequest.AsObject>} obj
 * @return {GetResourceUtilisationRequest|undefined}
 */
function getRequestFromObject(obj) {
  if (!obj) return undefined;
  const dst = new GetResourceUtilisationRequest();
  setProperties(dst, obj, 'name');
  dst.setReadMask(fieldMaskFromObject(obj.readMask));
  return dst;
}
