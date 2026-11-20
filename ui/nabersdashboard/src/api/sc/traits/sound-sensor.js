import {fieldMaskFromObject, setProperties} from '@/api/convpb.js';
import {clientOptions} from '@/api/grpcweb.js';
import {pullResource, setValue} from '@/api/resource.js';
import {SoundSensorApiPromiseClient} from '@smart-core-os/sc-bos-ui-gen/proto/smartcore/bos/soundsensor/v1/sound_sensor_grpc_web_pb';
import {PullSoundLevelRequest} from '@smart-core-os/sc-bos-ui-gen/proto/smartcore/bos/soundsensor/v1/sound_sensor_pb';

/**
 * @param {Partial<PullSoundLevelRequest.AsObject>} request
 * @param {ResourceValue<SoundLevel.AsObject, PullSoundLevelResponse>} resource
 */
export function pullSoundSensor(request, resource) {
  pullResource('SoundSensor.pullSoundLevel', resource, endpoint => {
    const api = new SoundSensorApiPromiseClient(endpoint, null, clientOptions());
    const stream = api.pullSoundLevel(pullSoundLevelRequestFromObject(request));
    stream.on('data', msg => {
      const changes = msg.getChangesList();
      for (const change of changes) {
        setValue(resource, change.getSoundLevel().toObject());
      }
    });
    return stream;
  });
}

/**
 * @param {Partial<PullSoundLevelRequest.AsObject>} obj
 * @return {PullSoundLevelRequest|undefined}
 */
function pullSoundLevelRequestFromObject(obj) {
  if (!obj) return undefined;
  const dst = new PullSoundLevelRequest();
  setProperties(dst, obj, 'name', 'updatesOnly');
  dst.setReadMask(fieldMaskFromObject(obj.readMask));
  return dst;
}
