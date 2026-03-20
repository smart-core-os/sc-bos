import {fieldMaskFromObject, setProperties} from '@/api/convpb';
import {clientOptions} from '@/api/grpcweb';
import {pullResource, setValue} from '@/api/resource';
import {FanSpeedApiPromiseClient} from '@smart-core-os/sc-bos-ui-gen/proto/smartcore/bos/fanspeed/v1/fan_speed_grpc_web_pb';
import {FanSpeed, PullFanSpeedRequest} from '@smart-core-os/sc-bos-ui-gen/proto/smartcore/bos/fanspeed/v1/fan_speed_pb';

/**
 * @param {Partial<PullFanSpeedRequest.AsObject>} request
 * @param {ResourceValue<FanSpeed.AsObject, PullFanSpeedResponse>} resource
 */
export function pullFanSpeed(request, resource) {
  pullResource('FanSpeed.pullFanSpeed', resource, endpoint => {
    const api = apiClient(endpoint);
    const stream = api.pullFanSpeed(pullFanSpeedRequestFromObject(request));
    stream.on('data', msg => {
      const changes = msg.getChangesList();
      for (const change of changes) {
        setValue(resource, change.getFanSpeed().toObject());
      }
    });
    return stream;
  });
}

/**
 * A map from id to name for FanSpeed.Direction.
 *
 * @type {Object<number, string>}
 */
export const directionNames = Object.entries(FanSpeed.Direction).reduce((all, [name, id]) => {
  all[id] = name;
  return all;
}, {});

/**
 * @param {string} endpoint
 * @return {FanSpeedApiPromiseClient}
 */
function apiClient(endpoint) {
  return new FanSpeedApiPromiseClient(endpoint, null, clientOptions());
}

/**
 * @param {Partial<PullFanSpeedRequest.AsObject>} obj
 * @return {PullFanSpeedRequest|undefined}
 */
function pullFanSpeedRequestFromObject(obj) {
  if (!obj) return undefined;

  const dst = new PullFanSpeedRequest();
  setProperties(dst, obj, 'name', 'updatesOnly');
  dst.setReadMask(fieldMaskFromObject(obj.readMask));
  return dst;
}
