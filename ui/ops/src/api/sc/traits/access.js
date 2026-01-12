import {fieldMaskFromObject, setProperties} from '@/api/convpb';
import {clientOptions} from '@/api/grpcweb';
import {pullResource, setValue, trackAction} from '@/api/resource';
import {AccessApiPromiseClient} from '@smart-core-os/sc-bos-ui-gen/proto/smartcore/bos/access/v1/access_grpc_web_pb';
import {AccessAttempt, GetLastAccessAttemptRequest, PullAccessAttemptsRequest} from '@smart-core-os/sc-bos-ui-gen/proto/smartcore/bos/access/v1/access_pb';

/**
 * @param {Partial<PullAccessAttemptsRequest.AsObject>} request
 * @param {ResourceValue<AccessAttempt.AsObject, PullAccessAttemptsResponse>} resource
 */
export function pullAccessAttempts(request, resource) {
  pullResource('Access.pullAccessAttempts', resource, endpoint => {
    const api = apiClient(endpoint);
    const stream = api.pullAccessAttempts(pullAccessAttemptsRequestFromObject(request));
    stream.on('data', msg => {
      const changes = msg.getChangesList();
      for (const change of changes) {
        setValue(resource, change.getAccessAttempt().toObject());
      }
    });
    return stream;
  });
}

/**
 * @param {Partial<GetLastAccessAttemptRequest.AsObject>} request
 * @param {ActionTracker<AccessAttempt.AsObject>} [tracker]
 * @return {Promise<AccessAttempt.AsObject>}
 */
export function getLastAccessAttempt(request, tracker) {
  return trackAction('Access.getLastAccessAttempt', tracker ?? {}, endpoint => {
    const api = apiClient(endpoint);
    return api.getLastAccessAttempt(getLastAccessAttemptRequestFromObject(request));
  });
}

/**
 * A map from id to name for AccessAttempt.Grant.
 *
 * @type {Object<number, string>}
 */
export const grantNamesByID = Object.entries(AccessAttempt.Grant).reduce((all, [name, id]) => {
  all[id] = name;
  return all;
}, {});

/**
 * @param {string} endpoint
 * @return {AccessApiPromiseClient}
 */
function apiClient(endpoint) {
  return new AccessApiPromiseClient(endpoint, null, clientOptions());
}

/**
 * @param {Partial<PullAccessAttemptsRequest.AsObject>} obj
 * @return {PullAccessAttemptsRequest|undefined}
 */
function pullAccessAttemptsRequestFromObject(obj) {
  if (!obj) return undefined;

  const dst = new PullAccessAttemptsRequest();
  setProperties(dst, obj, 'name', 'updatesOnly');
  dst.setReadMask(fieldMaskFromObject(obj.readMask));
  return dst;
}

/**
 * @param {Partial<GetLastAccessAttemptRequest.AsObject>} obj
 * @return {undefined|GetLastAccessAttemptRequest}
 */
function getLastAccessAttemptRequestFromObject(obj) {
  if (!obj) return undefined;

  const dst = new GetLastAccessAttemptRequest();
  setProperties(dst, obj, 'name');
  dst.setReadMask(fieldMaskFromObject(obj.readMask));
  return dst;
}
