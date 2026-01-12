import {fieldMaskFromObject, setProperties} from '@/api/convpb';
import {clientOptions} from '@/api/grpcweb';
import {pullResource, setValue, trackAction} from '@/api/resource';
import {StatusApiPromiseClient} from '@smart-core-os/sc-bos-ui-gen/proto/smartcore/bos/status/v1/status_grpc_web_pb';
import {GetCurrentStatusRequest, PullCurrentStatusRequest} from '@smart-core-os/sc-bos-ui-gen/proto/smartcore/bos/status/v1/status_pb';


/**
 * @param {Partial<PullCurrentStatusRequest.AsObject>} request
 * @param {ResourceValue<StatusLog.AsObject, PullCurrentStatusResponse>} resource
 */
export function pullCurrentStatus(request, resource) {
  pullResource('Status.pullCurrentStatus', resource, endpoint => {
    const api = apiClient(endpoint);
    const stream = api.pullCurrentStatus(pullCurrentStatusRequestFromObject(request));
    stream.on('data', msg => {
      const changes = msg.getChangesList();
      for (const change of changes) {
        setValue(resource, change.getCurrentStatus().toObject());
      }
    });
    return stream;
  });
}

/**
 * @param {Partial<GetCurrentStatusRequest.AsObject>} request
 * @param {ActionTracker<StatusLog.AsObject>} [tracker]
 * @return {Promise<StatusLog.AsObject>}
 */
export function getCurrentStatus(request, tracker) {
  return trackAction('Status.getCurrentStatus', tracker ?? {}, endpoint => {
    const api = apiClient(endpoint);
    return api.getCurrentStatus(getCurrentStatusRequestFromObject(request));
  });
}

/**
 * @param {string} endpoint
 * @return {StatusApiPromiseClient}
 */
function apiClient(endpoint) {
  return new StatusApiPromiseClient(endpoint, null, clientOptions());
}

/**
 * @param {Partial<PullCurrentStatusRequest.AsObject>} obj
 * @return {PullCurrentStatusRequest|undefined}
 */
function pullCurrentStatusRequestFromObject(obj) {
  if (!obj) return undefined;

  const dst = new PullCurrentStatusRequest();
  setProperties(dst, obj, 'name', 'updatesOnly');
  dst.setReadMask(fieldMaskFromObject(obj.readMask));
  return dst;
}

/**
 * @param {Partial<GetCurrentStatusRequest.AsObject>} obj
 * @return {undefined|GetCurrentStatusRequest}
 */
function getCurrentStatusRequestFromObject(obj) {
  if (!obj) return undefined;

  const dst = new GetCurrentStatusRequest();
  setProperties(dst, obj, 'name');
  dst.setReadMask(fieldMaskFromObject(obj.readMask));
  return dst;
}
