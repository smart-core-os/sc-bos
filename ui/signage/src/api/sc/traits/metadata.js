import {fieldMaskFromObject, setProperties} from '@/api/convpb';
import {clientOptions} from '@/api/grpcweb';
import {trackAction} from '@/api/resource';
import {pullResource, setValue} from '@/api/resource.js';
import {MetadataApiPromiseClient} from '@smart-core-os/sc-api-grpc-web/traits/metadata_grpc_web_pb';
import {GetMetadataRequest, PullMetadataRequest} from '@smart-core-os/sc-api-grpc-web/traits/metadata_pb';


/**
 * @param {Partial<PullMetadataRequest.AsObject>} request
 * @param {ResourceValue<Metadata.AsObject, PullMetadataResponse>} resource
 */
export function pullMetadata(request, resource) {
  pullResource('Metadata.pullMetadata', resource, endpoint => {
    const api = apiClient(endpoint);
    const stream = api.pullMetadata(pullMetadataRequestFromObject(request));
    stream.on('data', msg => {
      const changes = msg.getChangesList();
      for (const change of changes) {
        setValue(resource, change.getMetadata().toObject());
      }
    });
    return stream;
  });
}

/**
 * @param {Partial<GetMetadataRequest.AsObject>} request
 * @param {ActionTracker<Metadata.AsObject>} [tracker]
 * @return {Promise<Metadata.AsObject>}
 */
export function getMetadata(request, tracker) {
  return trackAction('Metadata.getMetadata', tracker ?? {}, endpoint => {
    const api = apiClient(endpoint);
    return api.getMetadata(getMetadataRequestFromObject(request));
  });
}

/**
 * @param {string} endpoint
 * @return {MetadataApiPromiseClient}
 */
function apiClient(endpoint) {
  return new MetadataApiPromiseClient(endpoint, null, clientOptions());
}

/**
 * @param {Partial<PullMetadataRequest.AsObject>} obj
 * @return {PullMetadataRequest}
 */
function pullMetadataRequestFromObject(obj) {
  if (!obj) return undefined;

  const req = new PullMetadataRequest();
  setProperties(req, obj, 'name', 'updatesOnly');
  req.setReadMask(fieldMaskFromObject(obj.readMask));
  return req;
}

/**
 * @param {Partial<GetMetadataRequest.AsObject>} obj
 * @return {GetMetadataRequest}
 */
function getMetadataRequestFromObject(obj) {
  if (!obj) return undefined;

  const req = new GetMetadataRequest();
  setProperties(req, obj, 'name');
  req.setReadMask(fieldMaskFromObject(obj.readMask));
  return req;
}
