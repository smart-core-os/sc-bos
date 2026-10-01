import {fieldMaskFromObject, setProperties} from '@/api/convpb.js';
import {clientOptions} from '@/api/grpcweb.js';
import {pullResource, setValue} from '@/api/resource.js';
import {AirTemperatureApiPromiseClient} from '@smart-core-os/sc-bos-ui-gen/proto/smartcore/bos/airtemperature/v1/air_temperature_grpc_web_pb';
import {PullAirTemperatureRequest} from '@smart-core-os/sc-bos-ui-gen/proto/smartcore/bos/airtemperature/v1/air_temperature_pb';

/**
 * @param {Partial<PullAirTemperatureRequest.AsObject>} request
 * @param {ResourceValue<AirTemperature.AsObject, PullAirTemperatureResponse>} resource
 */
export function pullAirTemperature(request, resource) {
  pullResource('AirTemperature.pullAirTemperature', resource, endpoint => {
    const api = new AirTemperatureApiPromiseClient(endpoint, null, clientOptions());
    const stream = api.pullAirTemperature(pullAirTemperatureRequestFromObject(request));
    stream.on('data', msg => {
      const changes = msg.getChangesList();
      for (const change of changes) {
        setValue(resource, change.getAirTemperature().toObject());
      }
    });
    return stream;
  });
}

/**
 * @param {Partial<PullAirTemperatureRequest.AsObject>} obj
 * @return {PullAirTemperatureRequest|undefined}
 */
function pullAirTemperatureRequestFromObject(obj) {
  if (!obj) return undefined;
  const dst = new PullAirTemperatureRequest();
  setProperties(dst, obj, 'name', 'updatesOnly');
  dst.setReadMask(fieldMaskFromObject(obj.readMask));
  return dst;
}
