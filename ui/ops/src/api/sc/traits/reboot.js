import {fieldMaskFromObject, setProperties} from '@/api/convpb';
import {clientOptions} from '@/api/grpcweb.js';
import {pullResource, setValue, trackAction} from '@/api/resource.js';
import {RebootApiPromiseClient} from '@smart-core-os/sc-bos-ui-gen/proto/smartcore/bos/reboot/v1/reboot_grpc_web_pb';
import {PullRebootStateRequest, RebootRequest} from '@smart-core-os/sc-bos-ui-gen/proto/smartcore/bos/reboot/v1/reboot_pb';

/**
 * @param {Partial<PullRebootStateRequest.AsObject>} request
 * @param {ResourceValue<RebootState.AsObject, PullRebootStateResponse>} resource
 */
export function pullRebootState(request, resource) {
  pullResource('Reboot.pullRebootState', resource, endpoint => {
    const api = apiClient(endpoint);
    const stream = api.pullRebootState(pullRebootStateRequestFromObject(request));
    stream.on('data', msg => {
      const changes = msg.getChangesList();
      for (const change of changes) {
        setValue(resource, change.getRebootState().toObject());
      }
    });
    return stream;
  });
}

/**
 * @param {Partial<RebootRequest.AsObject>} request
 * @param {ActionTracker<RebootResponse.AsObject>} [tracker]
 * @return {Promise<RebootResponse.AsObject>}
 */
export function reboot(request, tracker) {
  return trackAction('Reboot.reboot', tracker ?? {}, endpoint => {
    const api = apiClient(endpoint);
    return api.reboot(rebootRequestFromObject(request));
  });
}

/**
 * @param {string} endpoint
 * @return {RebootApiPromiseClient}
 */
function apiClient(endpoint) {
  return new RebootApiPromiseClient(endpoint, null, clientOptions());
}

/**
 * @param {Partial<PullRebootStateRequest.AsObject>} obj
 * @return {PullRebootStateRequest|undefined}
 */
function pullRebootStateRequestFromObject(obj) {
  if (!obj) return undefined;
  const dst = new PullRebootStateRequest();
  setProperties(dst, obj, 'name', 'updatesOnly');
  dst.setReadMask(fieldMaskFromObject(obj.readMask));
  return dst;
}

/**
 * @param {Partial<RebootRequest.AsObject>} obj
 * @return {RebootRequest|undefined}
 */
function rebootRequestFromObject(obj) {
  if (!obj) return undefined;
  const dst = new RebootRequest();
  setProperties(dst, obj, 'name', 'reason');
  return dst;
}
