import {fieldMaskFromObject, setProperties} from '@/api/convpb';
import {clientOptions} from '@/api/grpcweb';
import {pullResource, setValue, trackAction} from '@/api/resource';
import {ElectricApiPromiseClient} from '@smart-core-os/sc-bos-ui-gen/proto/smartcore/bos/electric/v1/electric_grpc_web_pb';
import {GetDemandRequest, PullDemandRequest} from '@smart-core-os/sc-bos-ui-gen/proto/smartcore/bos/electric/v1/electric_pb';

/**
 * @param {Partial<PullDemandRequest.AsObject>} request
 * @param {ResourceValue<ElectricDemand.AsObject, PullDemandResponse>} resource
 */
export function pullDemand(request, resource) {
  pullResource('Electric.pullDemand', resource, endpoint => {
    const api = apiClient(endpoint);
    const stream = api.pullDemand(pullDemandRequestFromObject(request));
    stream.on('data', msg => {
      const changes = msg.getChangesList();
      for (const change of changes) {
        setValue(resource, change.getDemand().toObject());
      }
    });
    return stream;
  });
}

/**
 * @param {Partial<GetDemandRequest.AsObject>} request
 * @param {ActionTracker<ElectricDemand.AsObject>} [tracker]
 * @return {Promise<ElectricDemand.AsObject>}
 */
export function getDemand(request, tracker) {
  return trackAction('Electric.getDemand', tracker ?? {}, endpoint => {
    const api = new ElectricApiPromiseClient(endpoint, null, clientOptions());
    return api.getDemand(getDemandRequestFromObject(request));
  });
}

/**
 * @param {string} endpoint
 * @return {ElectricApiPromiseClient}
 */
function apiClient(endpoint) {
  return new ElectricApiPromiseClient(endpoint, null, clientOptions());
}

/**
 * @param {Partial<PullDemandRequest.AsObject>} obj
 * @return {PullDemandRequest|undefined}
 */
function pullDemandRequestFromObject(obj) {
  if (!obj) return undefined;

  const dst = new PullDemandRequest();
  setProperties(dst, obj, 'name', 'updatesOnly');
  dst.setReadMask(fieldMaskFromObject(obj.readMask));
  return dst;
}

/**
 * @param {Partial<GetDemandRequest.AsObject>} obj
 * @return {undefined|GetDemandRequest}
 */
function getDemandRequestFromObject(obj) {
  if (!obj) return undefined;

  const dst = new GetDemandRequest();
  setProperties(dst, obj, 'name');
  dst.setReadMask(fieldMaskFromObject(obj.readMask));
  return dst;
}
