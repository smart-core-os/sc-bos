import {fieldMaskFromObject, setProperties} from '@/api/convpb';
import {clientOptions} from '@/api/grpcweb';
import {pullResource, setValue, trackAction} from '@/api/resource';
import {EnterLeaveSensorApiPromiseClient} from '@smart-core-os/sc-api-grpc-web/traits/enter_leave_sensor_grpc_web_pb';
import {
  GetEnterLeaveEventRequest,
  PullEnterLeaveEventsRequest,
  ResetEnterLeaveTotalsRequest
} from '@smart-core-os/sc-api-grpc-web/traits/enter_leave_sensor_pb';

/**
 * @param {Partial<PullEnterLeaveEventsRequest.AsObject>} request
 * @param {ResourceValue<EnterLeaveEvent.AsObject, PullEnterLeaveEventsResponse>} resource
 */
export function pullEnterLeaveEvents(request, resource) {
  pullResource('EnterLeave.pullEnterLeaveEvents', resource, (endpoint) => {
    const api = apiClient(endpoint);
    const stream = api.pullEnterLeaveEvents(pullEnterLeaveEventsRequestFromObject(request));
    stream.on('data', (msg) => {
      const changes = msg.getChangesList();
      for (const change of changes) {
        setValue(resource, enterLeaveEventToObject(change.getEnterLeaveEvent()));
      }
    });
    return stream;
  });
}

/**
 * @param {Partial<GetEnterLeaveEventRequest.AsObject>} request
 * @param {ActionTracker<EnterLeaveEvent.AsObject>} [tracker]
 * @return {Promise<EnterLeaveEvent.AsObject>}
 */
export function getEnterLeaveEvent(request, tracker) {
  return trackAction('EnterLeaveSensor.getEnterLeaveEvent', tracker ?? {}, (endpoint) => {
    const api = apiClient(endpoint);
    return api.getEnterLeaveEvent(getEnterLeaveEventRequestFromObject(request));
  });
}

/**
 * @param {Partial<GetEnterLeaveEventRequest.AsObject>} request
 * @param {ActionTracker<ResetEnterLeaveTotalsResponse.AsObject>} [tracker]
 * @return {Promise<ResetEnterLeaveTotalsResponse.AsObject>}
 */
export function resetEnterLeaveTotals(request, tracker) {
  return trackAction('EnterLeaveSensor.resetEnterLeaveTotals', tracker ?? {}, (endpoint) => {
    const api = apiClient(endpoint);
    return api.resetEnterLeaveTotals(resetEnterLeaveTotalsRequestFromObject(request));
  });
}

/**
 * @param {string} endpoint
 * @return {EnterLeaveSensorApiPromiseClient}
 */
function apiClient(endpoint) {
  return new EnterLeaveSensorApiPromiseClient(endpoint, null, clientOptions());
}

/**
 * @param {Partial<PullEnterLeaveEventsRequest.AsObject>} obj
 * @return {PullEnterLeaveEventsRequest|undefined}
 */
function pullEnterLeaveEventsRequestFromObject(obj) {
  if (!obj) return undefined;

  const dst = new PullEnterLeaveEventsRequest();
  setProperties(dst, obj, 'name', 'updatesOnly');
  dst.setReadMask(fieldMaskFromObject(obj.readMask));
  return dst;
}

/**
 * @param {Partial<GetEnterLeaveEventRequest.AsObject>} obj
 * @return {undefined|GetEnterLeaveEventRequest}
 */
function getEnterLeaveEventRequestFromObject(obj) {
  if (!obj) return undefined;

  const dst = new GetEnterLeaveEventRequest();
  setProperties(dst, obj, 'name');
  dst.setReadMask(fieldMaskFromObject(obj.readMask));
  return dst;
}

/**
 * @param {Partial<ResetEnterLeaveTotalsRequest.AsObject>} obj
 * @return {ResetEnterLeaveTotalsRequest|undefined}
 */
function resetEnterLeaveTotalsRequestFromObject(obj) {
  if (!obj) return undefined;

  const dst = new ResetEnterLeaveTotalsRequest();
  setProperties(dst, obj, 'name');
  return dst;
}

/**
 *
 * @param {EnterLeaveEvent} obj
 * @return {EnterLeaveEvent.AsObject}
 */
function enterLeaveEventToObject(obj) {
  if (!obj) return undefined;

  const dst = obj.toObject();

  if (!obj.hasEnterTotal()) {
    dst.enterTotal = undefined;
  }
  if (!obj.hasLeaveTotal()) {
    dst.leaveTotal = undefined;
  }

  return dst;
}
