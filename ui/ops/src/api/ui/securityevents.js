import {
  fieldMaskFromObject,
  setProperties
} from '@/api/convpb.js';
import {clientOptions} from '@/api/grpcweb.js';
import {pullResource, setCollection, trackAction} from '@/api/resource';
import {SecurityEventApiPromiseClient} from '@smart-core-os/sc-bos-ui-gen/proto/smartcore/bos/securityevent/v1/security_event_grpc_web_pb';
import {
  ListSecurityEventsRequest, PullSecurityEventsRequest
} from '@smart-core-os/sc-bos-ui-gen/proto/smartcore/bos/securityevent/v1/security_event_pb';

/**
 * @param {Partial<ListSecurityEventsRequest.AsObject>} request
 * @param {ActionTracker<ListSecurityEventsResponse.AsObject>} [tracker]
 * @return {Promise<ListSecurityEventsResponse.AsObject>}
 */
export function listSecurityEvents(request, tracker) {
  return trackAction('SecurityEvents.listSecurityEvents', tracker ?? {}, endpoint => {
    const api = apiClient(endpoint);
    return api.listSecurityEvents(listSecurityEventsRequestFromObject(request));
  });
}

/**
 * @param {Partial<PullSecurityEventsRequest.AsObject>} request
 * @param {ResourceCollection<Alert.AsObject, PullSecurityEventsResponse>} resource
 */
export function pullSecurityEvents(request, resource) {
  pullResource('SecurityEvents.pullSecurityEvents', resource, endpoint => {
    const api = apiClient(endpoint);
    const stream = api.pullSecurityEvents(pullSecurityEventsRequestFromObject(request));
    stream.on('data', msg => {
      const changes = msg.getChangesList();
      for (const change of changes) {
        setCollection(resource, change, v => v.id);
      }
    });
    return stream;
  });
}

/**
 * @param {string} endpoint
 * @return {SecurityEventApiPromiseClient}
 */
function apiClient(endpoint) {
  return new SecurityEventApiPromiseClient(endpoint, clientOptions());
}

/**
 * @param {Partial<ListSecurityEventsRequest.AsObject>} obj
 * @return {ListSecurityEventsRequest}
 */
function listSecurityEventsRequestFromObject(obj) {
  if (!obj) return undefined;
  const dst = new ListSecurityEventsRequest();
  setProperties(dst, obj, 'name', 'pageToken', 'pageSize');
  dst.setReadMask(fieldMaskFromObject(obj.readMask));
  return dst;
}

/**
 * @param {Partial<PullSecurityEventsRequest.AsObject>} obj
 * @return {PullSecurityEventsRequest|undefined}
 */
function pullSecurityEventsRequestFromObject(obj) {
  if (!obj) return undefined;
  const dst = new PullSecurityEventsRequest();
  setProperties(dst, obj, 'name');
  dst.setReadMask(fieldMaskFromObject(obj.readMask));
  return dst;
}
