import {fieldMaskFromObject, setProperties} from '@/api/convpb.js';
import {clientOptions} from '@/api/grpcweb.js';
import {pullResource, setValue} from '@/api/resource.js';
import {BrightnessSensorApiPromiseClient} from '@smart-core-os/sc-bos-ui-gen/proto/smartcore/bos/brightnesssensor/v1/brightness_sensor_grpc_web_pb';
import {PullAmbientBrightnessRequest} from '@smart-core-os/sc-bos-ui-gen/proto/smartcore/bos/brightnesssensor/v1/brightness_sensor_pb';

/**
 * @param {Partial<PullAmbientBrightnessRequest.AsObject>} request
 * @param {ResourceValue<AmbientBrightness.AsObject, PullAmbientBrightnessResponse>} resource
 */
export function pullBrightnessSensor(request, resource) {
  pullResource('BrightnessSensor.pullAmbientBrightness', resource, endpoint => {
    const api = new BrightnessSensorApiPromiseClient(endpoint, null, clientOptions());
    const stream = api.pullAmbientBrightness(pullAmbientBrightnessRequestFromObject(request));
    stream.on('data', msg => {
      const changes = msg.getChangesList();
      for (const change of changes) {
        setValue(resource, change.getAmbientBrightness().toObject());
      }
    });
    return stream;
  });
}

/**
 * @param {Partial<PullAmbientBrightnessRequest.AsObject>} obj
 * @return {PullAmbientBrightnessRequest|undefined}
 */
function pullAmbientBrightnessRequestFromObject(obj) {
  if (!obj) return undefined;
  const dst = new PullAmbientBrightnessRequest();
  setProperties(dst, obj, 'name', 'updatesOnly');
  dst.setReadMask(fieldMaskFromObject(obj.readMask));
  return dst;
}
