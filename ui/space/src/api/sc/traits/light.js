import {fieldMaskFromObject, setProperties} from '@/api/convpb.js';
import {clientOptions} from '@/api/grpcweb.js';
import {pullResource, setValue, trackAction} from '@/api/resource.js';
import {tweenFromObject} from '@/api/sc/types/tween.js';
import {LightApiPromiseClient, LightInfoPromiseClient} from '@smart-core-os/sc-bos-ui-gen/proto/smartcore/bos/light/v1/light_grpc_web_pb';
import {
  Brightness,
  DescribeBrightnessRequest,
  GetBrightnessRequest,
  LightPreset,
  PullBrightnessRequest,
  UpdateBrightnessRequest
} from '@smart-core-os/sc-bos-ui-gen/proto/smartcore/bos/light/v1/light_pb';

/**
 * @param {Partial<PullBrightnessRequest.AsObject>} request
 * @param {ResourceValue<Brightness.AsObject, PullBrightnessResponse>} resource
 */
export function pullBrightness(request, resource) {
  pullResource('Light.pullBrightness', resource, endpoint => {
    const api = apiClient(endpoint);
    const stream = api.pullBrightness(pullBrightnessRequestFromObject(request));
    stream.on('data', msg => {
      const changes = msg.getChangesList();
      for (const change of changes) {
        setValue(resource, change.getBrightness().toObject());
      }
    });
    return stream;
  });
}

/**
 * @param {Partial<GetBrightnessRequest.AsObject>} request
 * @param {ActionTracker<Brightness.AsObject>} [tracker]
 * @return {Promise<Brightness.AsObject>}
 */
export function getBrightness(request, tracker) {
  return trackAction('Light.getBrightness', tracker ?? {}, endpoint => {
    const api = apiClient(endpoint);
    return api.getBrightness(getBrightnessRequestFromObject(request));
  });
}

/**
 * @param {Partial<UpdateBrightnessRequest.AsObject>} request
 * @param {ActionTracker<Brightness.AsObject>} [tracker]
 * @return {Promise<Brightness.AsObject>}
 */
export function updateBrightness(request, tracker) {
  return trackAction('Light.updateBrightness', tracker ?? {}, endpoint => {
    const api = apiClient(endpoint);
    return api.updateBrightness(updateBrightnessRequestFromObject(request));
  });
}

/**
 *
 * @param {DescribeBrightnessRequest.AsObject} request
 * @param {ActionTracker<BrightnessSupport>} [tracker]
 * @return {Promise<BrightnessSupport>}
 */
export function describeBrightness(request, tracker) {
  return trackAction('LightInfo.describeBrightness', tracker ?? {}, endpoint => {
    const api = infoClient(endpoint);
    return api.describeBrightness(describeBrightnessRequestFromObject(request));
  });
}

/**
 * @param {string} endpoint
 * @return {LightApiPromiseClient}
 */
function apiClient(endpoint) {
  return new LightApiPromiseClient(endpoint, null, clientOptions());
}

/**
 * @param {string} endpoint
 * @return {LightInfoPromiseClient}
 */
function infoClient(endpoint) {
  return new LightInfoPromiseClient(endpoint, null, clientOptions());
}

/**
 * @param {Partial<PullBrightnessRequest.AsObject>} obj
 * @return {PullBrightnessRequest|undefined}
 */
function pullBrightnessRequestFromObject(obj) {
  if (!obj) return undefined;

  const dst = new PullBrightnessRequest();
  setProperties(dst, obj, 'name', 'excludeRamping', 'updatesOnly');
  dst.setReadMask(fieldMaskFromObject(obj.readMask));
  return dst;
}

/**
 *
 * @param {Partial<GetBrightnessRequest.AsObject>} obj
 * @return {GetBrightnessRequest}
 */
export function getBrightnessRequestFromObject(obj) {
  if (!obj) return undefined;

  const dst = new GetBrightnessRequest();
  setProperties(dst, obj, 'name');
  dst.setUpdateMask(fieldMaskFromObject(obj.readMask));
  return dst;
}

/**
 *
 * @param {Partial<UpdateBrightnessRequest.AsObject>} obj
 * @return {UpdateBrightnessRequest}
 */
export function updateBrightnessRequestFromObject(obj) {
  if (!obj) return undefined;

  const dst = new UpdateBrightnessRequest();
  setProperties(dst, obj, 'name', 'delta');
  dst.setBrightness(brightnessFromObject(obj.brightness));
  dst.setUpdateMask(fieldMaskFromObject(obj.updateMask));
  return dst;
}

/**
 * Convert a JS object representation of Brightness into a protobuf Brightness object.
 *
 * @param {Partial<Brightness.AsObject>} obj
 * @return {Brightness}
 */
export function brightnessFromObject(obj) {
  if (!obj) return undefined;

  const brightness = new Brightness();
  setProperties(brightness, obj, 'levelPercent', 'targetLevelPercent');
  brightness.setPreset(lightPresetFromObject(obj.preset));
  brightness.setTargetPreset(lightPresetFromObject(obj.targetPreset));
  brightness.setBrightnessTween(tweenFromObject(obj.brightnessTween));
  return brightness;
}

/**
 * Convert a JS object representation of LightPreset into a protobuf LightPreset object.
 *
 * @param {Partial<LightPreset.AsObject>} obj
 * @return {LightPreset|null}
 */
export function lightPresetFromObject(obj) {
  if (!obj) return undefined;

  const dst = new LightPreset();
  setProperties(dst, obj, 'name', 'title');
  return dst;
}

/**
 * Convert a JS object representation of DescribeBrightnessRequest into a protobuf DescribeBrightnessRequest object.
 *
 * @param {Partial<DescribeBrightnessRequest.AsObject>} obj
 * @return {DescribeBrightnessRequest|null}
 */
export function describeBrightnessRequestFromObject(obj) {
  if (!obj) return undefined;

  const dst = new DescribeBrightnessRequest();
  setProperties(dst, obj, 'name');
  return dst;
}
