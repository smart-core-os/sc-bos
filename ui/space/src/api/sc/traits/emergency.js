import {fieldMaskFromObject, setProperties} from '@/api/convpb';
import {clientOptions} from '@/api/grpcweb';
import {pullResource, setValue, trackAction} from '@/api/resource';
import {EmergencyApiPromiseClient} from '@smart-core-os/sc-bos-ui-gen/proto/smartcore/bos/emergency/v1/emergency_grpc_web_pb';
import {GetEmergencyRequest, PullEmergencyRequest} from '@smart-core-os/sc-bos-ui-gen/proto/smartcore/bos/emergency/v1/emergency_pb';

/**
 * @param {Partial<PullEmergencyRequest.AsObject>} request
 * @param {ResourceValue<Emergency.AsObject, PullEmergencyResponse>} resource
 */
export function pullEmergency(request, resource) {
  pullResource('Emergency.pullEmergency', resource, (endpoint) => {
    const api = apiClient(endpoint);
    const stream = api.pullEmergency(pullEmergencyRequestFromObject(request));
    stream.on('data', (msg) => {
      const changes = msg.getChangesList();
      for (const change of changes) {
        setValue(resource, change.getEmergency().toObject());
      }
    });
    return stream;
  });
}

/**
 * @param {Partial<GetEmergencyRequest.AsObject>} request
 * @param {ActionTracker<Emergency.AsObject>} [tracker]
 * @return {Promise<Emergency.AsObject>}
 */
export function getEmergency(request, tracker) {
  return trackAction('Emergency.getEmergency', tracker ?? {}, (endpoint) => {
    const api = apiClient(endpoint);
    return api.getEmergency(getEmergencyRequestFromObject(request));
  });
}

/**
 * @param {string} endpoint
 * @return {EmergencyApiPromiseClient}
 */
function apiClient(endpoint) {
  return new EmergencyApiPromiseClient(endpoint, null, clientOptions());
}

/**
 * @param {Partial<PullEmergencyRequest.AsObject>} obj
 * @return {PullEmergencyRequest|undefined}
 */
function pullEmergencyRequestFromObject(obj) {
  if (!obj) return undefined;

  const dst = new PullEmergencyRequest();
  setProperties(dst, obj, 'name', 'updatesOnly');
  dst.setReadMask(fieldMaskFromObject(obj.readMask));
  return dst;
}

/**
 * @param {Partial<GetEmergencyRequest.AsObject>} obj
 * @return {undefined|GetEmergencyRequest}
 */
function getEmergencyRequestFromObject(obj) {
  if (!obj) return undefined;

  const dst = new GetEmergencyRequest();
  setProperties(dst, obj, 'name');
  dst.setReadMask(fieldMaskFromObject(obj.readMask));
  return dst;
}
