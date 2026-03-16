import {fieldMaskFromObject, setProperties} from '@/api/convpb';
import {clientOptions} from '@/api/grpcweb.js';
import {pullResource, setValue, trackAction} from '@/api/resource.js';
import {Actor} from '@smart-core-os/sc-bos-ui-gen/proto/smartcore/bos/actor/v1/actor_pb';
import {BootApiPromiseClient} from '@smart-core-os/sc-bos-ui-gen/proto/smartcore/bos/boot/v1/boot_grpc_web_pb';
import {PullBootStateRequest, RebootRequest} from '@smart-core-os/sc-bos-ui-gen/proto/smartcore/bos/boot/v1/boot_pb';

/**
 * @param {Partial<PullBootStateRequest.AsObject>} request
 * @param {ResourceValue<BootState.AsObject, PullBootStateResponse>} resource
 */
export function pullBootState(request, resource) {
  pullResource('Boot.pullBootState', resource, endpoint => {
    const api = apiClient(endpoint);
    const stream = api.pullBootState(pullBootStateRequestFromObject(request));
    stream.on('data', msg => {
      const changes = msg.getChangesList();
      for (const change of changes) {
        setValue(resource, change.getBootState().toObject());
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
  return trackAction('Boot.reboot', tracker ?? {}, endpoint => {
    const api = apiClient(endpoint);
    return api.reboot(rebootRequestFromObject(request));
  });
}

/**
 * @param {string} endpoint
 * @return {BootApiPromiseClient}
 */
function apiClient(endpoint) {
  return new BootApiPromiseClient(endpoint, null, clientOptions());
}

/**
 * @param {Partial<PullBootStateRequest.AsObject>} obj
 * @return {PullBootStateRequest|undefined}
 */
function pullBootStateRequestFromObject(obj) {
  if (!obj) return undefined;
  const dst = new PullBootStateRequest();
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
  dst.setActor(actorFromObject(obj.actor));
  return dst;
}

/**
 * @param {Partial<Actor.AsObject>} obj
 * @return {Actor|undefined}
 */
function actorFromObject(obj) {
  if (!obj) return undefined;
  const dst = new Actor();
  setProperties(dst, obj, 'displayName', 'email');
  return dst;
}
