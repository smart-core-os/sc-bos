import {fieldMaskFromObject, setProperties} from '@/api/convpb';
import {clientOptions} from '@/api/grpcweb.js';
import {pullResource, setValue, trackAction} from '@/api/resource.js';
import {OnOffApiPromiseClient} from '@smart-core-os/sc-bos-ui-gen/proto/smartcore/bos/onoff/v1/on_off_grpc_web_pb';
import {GetOnOffRequest, PullOnOffRequest} from '@smart-core-os/sc-bos-ui-gen/proto/smartcore/bos/onoff/v1/on_off_pb';

/**
 * @param {Partial<PullOnOffRequest.AsObject>} request
 * @param {ResourceValue<OnOff.AsObject, PullOnOffResponse>} resource
 */
export function pullOnOff(request, resource) {
  pullResource('OnOff.pullOnOff', resource, endpoint => {
    const api = apiClient(endpoint);
    const stream = api.pullOnOff(pullOnOffRequestFromObject(request));
    stream.on('data', msg => {
      const changes = msg.getChangesList();
      for (const change of changes) {
        setValue(resource, change.getOnOff().toObject());
      }
    });
    return stream;
  });
}

/**
 * @param {Partial<GetOnOffRequest.AsObject>} request
 * @param {ActionTracker<OnOff.AsObject>} [tracker]
 * @return {Promise<OnOff.AsObject>}
 */
export function getOnOff(request, tracker) {
  return trackAction('OnOff.getOnOff', tracker ?? {}, endpoint => {
    const api = apiClient(endpoint);
    return api.getOnOff(getOnOffRequestFromObject(request));
  });
}

/**
 * @param {string} endpoint
 * @return {OnOffApiPromiseClient}
 */
function apiClient(endpoint) {
  return new OnOffApiPromiseClient(endpoint, null, clientOptions());
}

/**
 * @param {Partial<PullOnOffRequest.AsObject>} obj
 * @return {undefined|PullOnOffRequest}
 */
function pullOnOffRequestFromObject(obj) {
  if (!obj) return undefined;

  const dst = new PullOnOffRequest();
  setProperties(dst, obj, 'name', 'updatesOnly');
  dst.setReadMask(fieldMaskFromObject(obj.readMask));
  return dst;
}

/**
 * @param {Partial<GetOnOffRequest.AsObject>} obj
 * @return {GetOnOffRequest|undefined}
 */
function getOnOffRequestFromObject(obj) {
  if (!obj) return undefined;

  const dst = new GetOnOffRequest();
  setProperties(dst, obj, 'name');
  dst.setReadMask(fieldMaskFromObject(obj.readMask));
  return dst;
}
