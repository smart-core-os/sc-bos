import {fieldMaskFromObject, setProperties, timestampToDate} from '@/api/convpb';
import {clientOptions} from '@/api/grpcweb';
import {pullResource, setCollection, setValue, trackAction} from '@/api/resource';
import {periodFromObject} from '@/api/sc/types/period';
import {HealthApiPromiseClient, HealthHistoryPromiseClient} from '@smart-core-os/sc-bos-ui-gen/proto/health_grpc_web_pb';
import {
  GetHealthCheckRequest,
  HealthCheck,
  ListHealthChecksRequest,
  ListHealthCheckHistoryRequest,
  PullHealthChecksRequest,
  PullHealthCheckRequest
} from '@smart-core-os/sc-bos-ui-gen/proto/health_pb';

/**
 * @param {Partial<ListHealthChecksRequest.AsObject>} request
 * @param {ActionTracker<ListHealthChecksResponse.AsObject>} [tracker]
 * @return {Promise<ListHealthChecksResponse.AsObject>}
 */
export function listHealthChecks(request, tracker) {
  return trackAction('Health.listHealthChecks', tracker ?? {}, endpoint => {
    const api = apiClient(endpoint);
    return api.listHealthChecks(listHealthChecksRequestFromObject(request));
  });
}

/**
 * @param {Partial<PullHealthChecksRequest.AsObject>} request
 * @param {ResourceCollection<HealthCheck.AsObject, PullHealthChecksResponse>} resource
 */
export function pullHealthChecks(request, resource) {
  pullResource('Health.pullHealthChecks', resource, endpoint => {
    const api = apiClient(endpoint);
    const stream = api.pullHealthChecks(pullHealthChecksRequestFromObject(request));
    stream.on('data', msg => {
      const changes = msg.getChangesList();
      for (const change of changes) {
        setCollection(resource, change, (hc) => hc.id);
      }
    });
    return stream;
  });
}

/**
 * @param {Partial<GetHealthCheckRequest.AsObject>} request
 * @param {ActionTracker<HealthCheck.AsObject>} [tracker]
 * @return {Promise<HealthCheck.AsObject>}
 */
export function getHealthCheck(request, tracker) {
  return trackAction('Health.getHealthCheck', tracker ?? {}, endpoint => {
    const api = apiClient(endpoint);
    return api.getHealthCheck(getHealthCheckRequestFromObject(request));
  });
}

/**
 * @param {Partial<PullHealthCheckRequest.AsObject>} request
 * @param {ResourceValue<HealthCheck.AsObject, PullHealthCheckResponse>} resource
 */
export function pullHealthCheck(request, resource) {
  pullResource('Health.pullHealthCheck', resource, endpoint => {
    const api = apiClient(endpoint);
    const stream = api.pullHealthCheck(pullHealthCheckRequestFromObject(request));
    stream.on('data', msg => {
      const changes = msg.getChangesList();
      for (const change of changes) {
        setValue(resource, change.getHealthCheck().toObject());
      }
    });
    return stream;
  });
}

/**
 * @param {Partial<ListHealthCheckHistoryRequest.AsObject>} request
 * @param {ActionTracker<ListHealthCheckHistoryResponse.AsObject>} [tracker]
 * @return {Promise<ListHealthCheckHistoryResponse.AsObject>}
 */
export function listHealthCheckHistory(request, tracker) {
  return trackAction('HealthHistory.listHealthCheckHistory', tracker ?? {}, endpoint => {
    const api = historyClient(endpoint);
    return api.listHealthCheckHistory(listHealthCheckHistoryRequestFromObject(request));
  });
}

/**
 * @param {HealthCheckRecord | HealthCheckRecord.AsObject} obj
 * @return {HealthCheckRecord.AsObject & {recordTime: Date|undefined}}
 */
export function healthCheckRecordToObject(obj) {
  if (!obj) return undefined;
  if (typeof obj.toObject === 'function') obj = obj.toObject();
  if (obj.recordTime) obj.recordTime = timestampToDate(obj.recordTime);
  return obj;
}

/**
 * @param {HealthCheck.Reliability.State} state
 * @return {string}
 */
export function reliabilityStateToString(state) {
  switch (state) {
    case HealthCheck.Reliability.State.STATE_UNSPECIFIED:
      return 'Unknown';
    case HealthCheck.Reliability.State.RELIABLE:
      return 'Reliable';
    case HealthCheck.Reliability.State.UNRELIABLE:
      return 'Unreliable';
    case HealthCheck.Reliability.State.CONN_TRANSIENT_FAILURE:
      return 'Connection Issues';
    case HealthCheck.Reliability.State.SEND_FAILURE:
      return 'Send Failure';
    case HealthCheck.Reliability.State.NO_RESPONSE:
      return 'No Response';
    case HealthCheck.Reliability.State.BAD_RESPONSE:
      return 'Bad Response';
    case HealthCheck.Reliability.State.NOT_FOUND:
      return 'Not Found';
    case HealthCheck.Reliability.State.PERMISSION_DENIED:
      return 'Permission Denied';
    default:
      return 'Unknown';
  }
}

/**
 * @param {HealthCheck.Check.State} state
 * @return {string}
 */
export function checkStateToString(state) {
  switch (state) {
    case HealthCheck.Check.State.STATE_UNSPECIFIED:
      return 'Unknown';
    case HealthCheck.Check.State.NORMAL:
      return 'Normal';
    case HealthCheck.Check.State.ABNORMAL:
      return 'Abnormal';
    case HealthCheck.Check.State.LOW:
      return 'Low';
    case HealthCheck.Check.State.HIGH:
      return 'High';
    default:
      return 'Unknown';
  }
}

/**
 * @param {HealthCheck.OccupantImpact} impact
 * @return {string}
 */
export function occupantImpactToString(impact) {
  switch (impact) {
    case HealthCheck.OccupantImpact.OCCUPANT_IMPACT_UNSPECIFIED:
      return 'Unknown';
    case HealthCheck.OccupantImpact.NO_OCCUPANT_IMPACT:
      return 'No Impact';
    case HealthCheck.OccupantImpact.COMFORT:
      return 'Comfort';
    case HealthCheck.OccupantImpact.HEALTH:
      return 'Health';
    case HealthCheck.OccupantImpact.LIFE:
      return 'Life Safety';
    default:
      return 'Unknown';
  }
}

/**
 * @param {HealthCheck.EquipmentImpact} impact
 * @return {string}
 */
export function equipmentImpactToString(impact) {
  switch (impact) {
    case HealthCheck.EquipmentImpact.EQUIPMENT_IMPACT_UNSPECIFIED:
      return 'Unknown';
    case HealthCheck.EquipmentImpact.NO_EQUIPMENT_IMPACT:
      return 'No Impact';
    case HealthCheck.EquipmentImpact.WARRANTY:
      return 'Warranty';
    case HealthCheck.EquipmentImpact.LIFESPAN:
      return 'Lifespan';
    case HealthCheck.EquipmentImpact.FUNCTION:
      return 'Function';
    default:
      return 'Unknown';
  }
}

/**
 * @param {string} endpoint
 * @return {HealthApiPromiseClient}
 */
function apiClient(endpoint) {
  return new HealthApiPromiseClient(endpoint, null, clientOptions());
}

/**
 * @param {string} endpoint
 * @return {HealthHistoryPromiseClient}
 */
function historyClient(endpoint) {
  return new HealthHistoryPromiseClient(endpoint, null, clientOptions());
}

/**
 * @param {Partial<ListHealthChecksRequest.AsObject>} obj
 * @return {ListHealthChecksRequest|undefined}
 */
function listHealthChecksRequestFromObject(obj) {
  if (!obj) return undefined;

  const dst = new ListHealthChecksRequest();
  setProperties(dst, obj, 'name', 'pageSize', 'pageToken');
  dst.setReadMask(fieldMaskFromObject(obj.readMask));
  return dst;
}

/**
 * @param {Partial<PullHealthChecksRequest.AsObject>} obj
 * @return {PullHealthChecksRequest|undefined}
 */
function pullHealthChecksRequestFromObject(obj) {
  if (!obj) return undefined;

  const dst = new PullHealthChecksRequest();
  setProperties(dst, obj, 'name', 'updatesOnly');
  dst.setReadMask(fieldMaskFromObject(obj.readMask));
  return dst;
}

/**
 * @param {Partial<GetHealthCheckRequest.AsObject>} obj
 * @return {GetHealthCheckRequest|undefined}
 */
function getHealthCheckRequestFromObject(obj) {
  if (!obj) return undefined;

  const dst = new GetHealthCheckRequest();
  setProperties(dst, obj, 'name', 'id');
  dst.setReadMask(fieldMaskFromObject(obj.readMask));
  return dst;
}

/**
 * @param {Partial<PullHealthCheckRequest.AsObject>} obj
 * @return {PullHealthCheckRequest|undefined}
 */
function pullHealthCheckRequestFromObject(obj) {
  if (!obj) return undefined;

  const dst = new PullHealthCheckRequest();
  setProperties(dst, obj, 'name', 'id', 'updatesOnly');
  dst.setReadMask(fieldMaskFromObject(obj.readMask));
  return dst;
}

/**
 * @param {Partial<ListHealthCheckHistoryRequest.AsObject>} obj
 * @return {ListHealthCheckHistoryRequest|undefined}
 */
function listHealthCheckHistoryRequestFromObject(obj) {
  if (!obj) return undefined;

  const dst = new ListHealthCheckHistoryRequest();
  setProperties(dst, obj, 'name', 'id', 'pageToken', 'pageSize', 'orderBy');
  dst.setReadMask(fieldMaskFromObject(obj.readMask));
  dst.setPeriod(periodFromObject(obj.period));
  return dst;
}
