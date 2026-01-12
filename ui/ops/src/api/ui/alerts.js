import {
  convertProperties,
  fieldMaskFromObject,
  setProperties,
  timestampFromObject,
  timestampToDate
} from '@/api/convpb.js';
import {clientOptions} from '@/api/grpcweb.js';
import {pullResource, setCollection, setValue, trackAction} from '@/api/resource';
import {AlertAdminApiPromiseClient, AlertApiPromiseClient} from '@smart-core-os/sc-bos-ui-gen/proto/smartcore/bos/alert/v1/alerts_grpc_web_pb';
import {
  AcknowledgeAlertRequest,
  Alert,
  CreateAlertRequest,
  GetAlertMetadataRequest,
  ListAlertsRequest,
  PullAlertMetadataRequest,
  PullAlertsRequest
} from '@smart-core-os/sc-bos-ui-gen/proto/smartcore/bos/alert/v1/alerts_pb';

/**
 * @param {Partial<ListAlertsRequest.AsObject>} request
 * @param {ActionTracker<ListAlertsResponse.AsObject>} [tracker]
 * @return {Promise<ListAlertsResponse.AsObject>}
 */
export function listAlerts(request, tracker) {
  return trackAction('Alerts.listAlerts', tracker ?? {}, endpoint => {
    const api = apiClient(endpoint);
    return api.listAlerts(listAlertsRequestFromObject(request));
  });
}

/**
 * @param {Partial<PullAlertsRequest.AsObject>} request
 * @param {ResourceCollection<Alert.AsObject, PullAlertsResponse>} resource
 */
export function pullAlerts(request, resource) {
  pullResource('Alerts.pullAlerts', resource, endpoint => {
    const api = apiClient(endpoint);
    const stream = api.pullAlerts(pullAlertsRequestFromObject(request));
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
 * @param {Partial<GetAlertMetadataRequest.AsObject>} request
 * @param {ActionTracker<AlertMetadata.AsObject>} [tracker]
 * @return {Promise<AlertMetadata.AsObject>}
 */
export function getAlertMetadata(request, tracker) {
  return trackAction('Alerts.getAlertMetadata', tracker ?? {}, endpoint => {
    const api = apiClient(endpoint);
    return api.getAlertMetadata(getAlertMetadataRequestFromObject(request));
  });
}

/**
 * @param {Partial<PullAlertMetadataRequest.AsObject>} request
 * @param {ResourceValue<AlertMetadata.AsObject, PullAlertMetadataResponse>} resource
 */
export function pullAlertMetadata(request, resource) {
  pullResource('Alerts.pullAlertMetadata', resource, endpoint => {
    const api = apiClient(endpoint);
    const stream = api.pullAlertMetadata(pullAlertMetadataRequestFromObject(request));
    stream.on('data', msg => {
      const changes = msg.getChangesList();
      for (const change of changes) {
        setValue(resource, change.getMetadata().toObject());
      }
    });
    return stream;
  });
}

/**
 * @param {Partial<AcknowledgeAlertRequest.AsObject>} request
 * @param {ActionTracker<Alert.AsObject>} [tracker]
 * @return {Promise<Alert.AsObject>}
 */
export function acknowledgeAlert(request, tracker) {
  return trackAction('Alerts.acknowledgeAlert', tracker ?? {}, endpoint => {
    const api = apiClient(endpoint);
    return api.acknowledgeAlert(acknowledgeAlertRequestFromObject(request));
  });
}

/**
 * @param {Partial<AcknowledgeAlertRequest.AsObject>} request
 * @param {ActionTracker<Alert.AsObject>} [tracker]
 * @return {Promise<Alert.AsObject>}
 */
export function unacknowledgeAlert(request, tracker) {
  return trackAction('Alerts.unacknowledgeAlert', tracker ?? {}, endpoint => {
    const api = apiClient(endpoint);
    return api.unacknowledgeAlert(acknowledgeAlertRequestFromObject(request));
  });
}

/**
 * @param {Partial<CreateAlertRequest.AsObject>} request
 * @param {ActionTracker<CreateAlertRequest.AsObject>} [tracker]
 * @return {Promise<Alert.AsObject>}
 */
export function createAlert(request, tracker) {
  return trackAction('Alerts.createAlert', tracker ?? {}, endpoint => {
    const api = adminClient(endpoint);
    return api.createAlert(createAlertRequestFromObject(request));
  });
}

/**
 * @param {Alert | Alert.AsObject} obj
 * @return {Alert.AsObject & {createTime: Date, updateTime: Date, acknowledgement?: {acknowledgeTime: Date}}|undefined}
 */
export function alertToObject(obj) {
  if (!obj) return undefined;
  if (typeof obj.toObject === 'function') obj = obj.toObject();

  if (obj.createTime) obj.createTime = timestampToDate(obj.createTime);
  if (obj.resolveTime) obj.resolveTime = timestampToDate(obj.resolveTime);
  if (obj.acknowledgement) {
    if (obj.acknowledgement.acknowledgeTime) obj.acknowledgement.acknowledgeTime = timestampToDate(obj.acknowledgement.acknowledgeTime);
  }
  return obj;
}

/**
 * @param {string} endpoint
 * @return {AlertApiPromiseClient}
 */
function apiClient(endpoint) {
  return new AlertApiPromiseClient(endpoint, null, clientOptions());
}

/**
 *
 * @param {string} endpoint
 * @return {AlertAdminApiPromiseClient}
 */
function adminClient(endpoint) {
  return new AlertAdminApiPromiseClient(endpoint, null, clientOptions());
}

/**
 * @param {Partial<ListAlertsRequest.AsObject>} obj
 * @return {ListAlertsRequest}
 */
function listAlertsRequestFromObject(obj) {
  if (!obj) return undefined;
  const dst = new ListAlertsRequest();
  setProperties(dst, obj, 'name', 'pageToken', 'pageSize');
  dst.setReadMask(fieldMaskFromObject(obj.readMask));
  dst.setQuery(alertQueryFromObject(obj.query));
  return dst;
}

/**
 * @param {Partial<PullAlertsRequest.AsObject>} obj
 * @return {PullAlertsRequest|undefined}
 */
function pullAlertsRequestFromObject(obj) {
  if (!obj) return undefined;
  const dst = new PullAlertsRequest();
  setProperties(dst, obj, 'name');
  dst.setReadMask(fieldMaskFromObject(obj.readMask));
  dst.setQuery(alertQueryFromObject(obj.query));
  return dst;
}

/**
 * @param {Partial<GetAlertMetadataRequest.AsObject>} obj
 * @return {GetAlertMetadataRequest|undefined}
 */
function getAlertMetadataRequestFromObject(obj) {
  if (!obj) return undefined;
  const dst = new GetAlertMetadataRequest();
  setProperties(dst, obj, 'name');
  dst.setReadMask(fieldMaskFromObject(obj.readMask));
  return dst;
}

/**
 * @param {Partial<PullAlertMetadataRequest.AsObject>} obj
 * @return {PullAlertMetadataRequest|undefined}
 */
function pullAlertMetadataRequestFromObject(obj) {
  if (!obj) return undefined;
  const dst = new PullAlertMetadataRequest();
  setProperties(dst, obj, 'name', 'updatesOnly');
  dst.setReadMask(fieldMaskFromObject(obj.readMask));
  return dst;
}

/**
 * @param {Partial<Alert.Query.AsObject>} obj
 * @return {Alert.Query|undefined}
 */
function alertQueryFromObject(obj) {
  if (!obj) return undefined;
  const dst = new Alert.Query();
  setProperties(dst, obj, 'floor', 'zone', 'subsystem', 'severity', 'source',
      'severityNotBelow', 'severityNotAbove',
      'acknowledged', 'resolved');
  convertProperties(dst, obj, timestampFromObject,
      'createdNotBefore', 'createdNotAfter',
      'resolvedNotBefore', 'resolvedNotAfter');
  return dst;
}

/**
 *
 * @param {Partial<AcknowledgeAlertRequest.AsObject>} obj
 * @return {AcknowledgeAlertRequest|undefined}
 */
function acknowledgeAlertRequestFromObject(obj) {
  if (!obj) return undefined;
  const dst = new AcknowledgeAlertRequest();
  setProperties(dst, obj, 'name', 'id', 'allowMissing', 'allowAcknowledged');
  dst.setAuthor(alertAuthorFromObject(obj.author));
  return dst;
}

/**
 * @param {Partial<Alert.Acknowledgement.Author.AsObject>} obj
 * @return {undefined|Alert.Acknowledgement.Author}
 */
function alertAuthorFromObject(obj) {
  if (!obj) return undefined;
  const dst = new Alert.Acknowledgement.Author();
  setProperties(dst, obj, 'id', 'displayName', 'email');
  return dst;
}

/**
 *
 * @param {Partial<CreateAlertRequest.AsObject>} obj
 * @return {CreateAlertRequest}
 */
function createAlertRequestFromObject(obj) {
  if (!obj) return undefined;

  const request = new CreateAlertRequest();
  request.setAlert(alertFromObject(obj.alert));
  return request;
}

/**
 *
 * @param {Partial<Alert.AsObject>} obj
 * @return {Alert}
 */
function alertFromObject(obj) {
  if (!obj) return undefined;

  const alert = new Alert();
  setProperties(alert, obj, 'description', 'source', 'floor', 'zone', 'severity', 'subsystem');
  return alert;
}
