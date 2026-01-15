import {fieldMaskFromObject, setProperties} from '@/api/convpb';
import {clientOptions} from '@/api/grpcweb.js';
import {trackAction} from '@/api/resource';
import {pullResource, setValue} from '@/api/resource.js';
import {TemperatureApiPromiseClient} from '@smart-core-os/sc-bos-ui-gen/proto/smartcore/bos/temperature/v1/temperature_grpc_web_pb';
import {
  PullTemperatureRequest,
  Temperature,
  UpdateTemperatureRequest
} from '@smart-core-os/sc-bos-ui-gen/proto/smartcore/bos/temperature/v1/temperature_pb';
import {Temperature as TemperatureUnit} from '@smart-core-os/sc-api-grpc-web/types/unit_pb';

/**
 * @param {Partial<PullTemperatureRequest.AsObject>} request
 * @param {ResourceValue<Temperature.AsObject, PullTemperatureResponse>} resource
 */
export function pullTemperature(request, resource) {
  pullResource('Temperature.pullTemperature', resource, endpoint => {
    const api = apiClient(endpoint);
    const stream = api.pullTemperature(pullTemperatureRequestFromObject(request));
    stream.on('data', msg => {
      const changes = msg.getChangesList();
      for (const change of changes) {
        setValue(resource, change.getTemperature().toObject());
      }
    });
    return stream;
  });
}

/**
 *
 * @param {Partial<UpdateTemperatureRequest.AsObject>} request
 * @param {ActionTracker<Temperature.AsObject>} [tracker]
 * @return {Promise<Temperature.AsObject>}
 */
export function updateTemperature(request, tracker) {
  return trackAction('Temperature.updateTemperature', tracker ?? {}, endpoint => {
    const api = apiClient(endpoint);
    return api.updateTemperature(updateTemperatureRequestFromObject(request));
  });
}

/**
 * @param {string} endpoint
 * @return {TemperatureApiPromiseClient}
 */
function apiClient(endpoint) {
  return new TemperatureApiPromiseClient(endpoint, null, clientOptions());
}

/**
 * @param {Partial<PullTemperatureRequest.AsObject>} obj
 * @return {PullTemperatureRequest|undefined}
 */
function pullTemperatureRequestFromObject(obj) {
  if (!obj) return undefined;

  const req = new PullTemperatureRequest();
  setProperties(req, obj, 'name', 'updatesOnly');
  req.setReadMask(fieldMaskFromObject(obj.readMask));
  return req;
}

/**
 * @param {Partial<UpdateTemperatureRequest.AsObject>} obj
 * @return {UpdateTemperatureRequest}
 */
function updateTemperatureRequestFromObject(obj) {
  if (!obj) return undefined;

  const req = new UpdateTemperatureRequest();
  setProperties(req, obj, 'name', 'delta');
  req.setTemperature(temperatureFromObject(obj.temperature));
  req.setUpdateMask(fieldMaskFromObject(obj.updateMask));
  return req;
}

/**
 * @param {Partial<Temperature.AsObject>} obj
 * @return {Temperature}
 */
function temperatureFromObject(obj) {
  if (!obj) return undefined;

  const temp = new Temperature();
  temp.setSetPoint(temperatureUnitFromObject(obj.setPoint));
  temp.setMeasured(temperatureUnitFromObject(obj.measured));
  return temp;
}

/**
 * @param {Partial<TemperatureUnit.AsObject>} obj
 * @return {TemperatureUnit|undefined}
 */
function temperatureUnitFromObject(obj) {
  if (!obj) return undefined;

  const t = new TemperatureUnit();
  setProperties(t, obj, 'valueCelsius');
  return t;
}

/**
 *
 * @param {Partial<TemperatureUnit.AsObject>} value
 * @return {string}
 */
export function temperatureToString(value) {
  if (Object.hasOwn(value, 'valueCelsius')) {
    return value.valueCelsius.toFixed(1) + '°C';
  }
  return '-';
}

