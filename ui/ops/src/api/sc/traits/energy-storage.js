import {fieldMaskFromObject, setProperties} from '@/api/convpb';
import {clientOptions} from '@/api/grpcweb';
import {pullResource, setValue, trackAction} from '@/api/resource';
import {EnergyStorageApiPromiseClient} from '@smart-core-os/sc-bos-ui-gen/proto/smartcore/bos/energystorage/v1/energy_storage_grpc_web_pb';
import {GetEnergyLevelRequest, PullEnergyLevelRequest, ChargeRequest} from '@smart-core-os/sc-bos-ui-gen/proto/smartcore/bos/energystorage/v1/energy_storage_pb';

/**
 * @param {Partial<PullEnergyLevelRequest.AsObject>} request
 * @param {ResourceValue<EnergyLevel.AsObject, PullEnergyLevelResponse>} resource
 */
export function pullEnergyLevel(request, resource) {
  pullResource('EnergyStorage.pullEnergyLevel', resource, endpoint => {
    const api = apiClient(endpoint);
    const stream = api.pullEnergyLevel(pullEnergyLevelRequestFromObject(request));
    stream.on('data', msg => {
      const changes = msg.getChangesList();
      for (const change of changes) {
        setValue(resource, change.getEnergyLevel().toObject());
      }
    });
    return stream;
  });
}

/**
 * @param {Partial<GetEnergyLevelRequest.AsObject>} request
 * @param {ActionTracker<EnergyLevel.AsObject>} [tracker]
 * @return {Promise<EnergyLevel.AsObject>}
 */
export function getEnergyLevel(request, tracker) {
  return trackAction('EnergyStorage.getEnergyLevel', tracker ?? {}, endpoint => {
    const api = new EnergyStorageApiPromiseClient(endpoint, null, clientOptions());
    return api.getEnergyLevel(getEnergyLevelRequestFromObject(request));
  });
}

/**
 * @param {Partial<ChargeRequest.AsObject>} request
 * @param {ActionTracker<ChargeResponse.AsObject>} [tracker]
 * @return {Promise<ChargeResponse.AsObject>}
 */
export function charge(request, tracker) {
  return trackAction('EnergyStorage.charge', tracker ?? {}, endpoint => {
    const api = new EnergyStorageApiPromiseClient(endpoint, null, clientOptions());
    return api.charge(chargeRequestFromObject(request));
  });
}

/**
 * @param {string} endpoint
 * @return {EnergyStorageApiPromiseClient}
 */
function apiClient(endpoint) {
  return new EnergyStorageApiPromiseClient(endpoint, null, clientOptions());
}

/**
 * @param {Partial<PullEnergyLevelRequest.AsObject>} obj
 * @return {PullEnergyLevelRequest|undefined}
 */
function pullEnergyLevelRequestFromObject(obj) {
  if (!obj) return undefined;

  const dst = new PullEnergyLevelRequest();
  setProperties(dst, obj, 'name', 'updatesOnly');
  dst.setReadMask(fieldMaskFromObject(obj.readMask));
  return dst;
}

/**
 * @param {Partial<GetEnergyLevelRequest.AsObject>} obj
 * @return {undefined|GetEnergyLevelRequest}
 */
function getEnergyLevelRequestFromObject(obj) {
  if (!obj) return undefined;

  const dst = new GetEnergyLevelRequest();
  setProperties(dst, obj, 'name');
  dst.setReadMask(fieldMaskFromObject(obj.readMask));
  return dst;
}

/**
 * @param {Partial<ChargeRequest.AsObject>} obj
 * @return {undefined|ChargeRequest}
 */
function chargeRequestFromObject(obj) {
  if (!obj) return undefined;

  const dst = new ChargeRequest();
  setProperties(dst, obj, 'name', 'charge');
  return dst;
}

