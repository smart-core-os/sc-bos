import {fieldMaskFromObject, setProperties, timestampToDate} from '@/api/convpb';
import {clientOptions} from '@/api/grpcweb';
import {pullResource, setCollection, setValue, trackAction} from '@/api/resource';
import {ServicesApiPromiseClient} from '@smart-core-os/sc-bos-ui-gen/proto/smartcore/bos/services/v1/services_grpc_web_pb';
import {
  ConfigureServiceRequest,
  GetServiceRequest,
  ListServicesRequest,
  PullServiceMetadataRequest,
  PullServiceRequest,
  PullServicesRequest,
  StartServiceRequest,
  StopServiceRequest
} from '@smart-core-os/sc-bos-ui-gen/proto/smartcore/bos/services/v1/services_pb';
import {GetMetadataRequest} from '@smart-core-os/sc-api-grpc-web/traits/metadata_pb';

/**
 * @param {Partial<GetServiceRequest.AsObject>} request
 * @param {ActionTracker<Service.AsObject>?} tracker
 * @return {Promise<Service.AsObject>}
 */
export function getService(request, tracker) {
  return trackAction('ServicesApi.GetService', tracker ?? {}, endpoint => {
    const api = apiClient(endpoint);
    return api.getService(getServiceRequestFromObject(request));
  });
}

/**
 * @param {Partial<PullServiceRequest.AsObject>} request - must have an id, should have a name
 * @param {ResourceValue<Service.AsObject, PullServiceResponse>} resource
 */
export function pullService(request, resource) {
  pullResource('Services.PullService', resource, endpoint => {
    const api = apiClient(endpoint);
    const stream = api.pullService(pullServiceRequestFromObject(request));
    stream.on('data', msg => {
      const changes = msg.getChangesList();
      changes.forEach(change => {
        setValue(resource, change.getService().toObject());
      });
    });
    return stream;
  });
}

/**
 * @param {Partial<GetMetadataRequest.AsObject>} request
 * @param {ActionTracker<ServiceMetadata.AsObject>} [tracker]
 * @return {Promise<ServiceMetadata.AsObject>}
 */
export function getServiceMetadata(request, tracker) {
  const name = String(request.name);
  if (!name) throw new Error('request.name must be specified');
  return trackAction('Services.GetServiceMetadata', tracker ?? {}, endpoint => {
    const api = apiClient(endpoint);
    return api.getServiceMetadata(getServiceMetadataRequestFromObject(request));
  });
}

/**
 * @param {Partial<PullServiceMetadataRequest.AsObject>} request
 * @param {ResourceValue<ServiceMetadata.AsObject, ServiceMetadata>} resource
 */
export function pullServiceMetadata(request, resource) {
  pullResource('Services.PullServiceMetadata', resource, endpoint => {
    const api = apiClient(endpoint);
    const stream = api.pullServiceMetadata(pullServiceMetadataRequestFromObject(request));
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
 * @param {Partial<ListServicesRequest.AsObject>} request
 * @param {ActionTracker<ListServicesRequest.AsObject>} [tracker]
 * @return {Promise<ListServicesResponse.AsObject>}
 */
export function listServices(request, tracker) {
  return trackAction('Services.ListServices', tracker ?? {}, endpoint => {
    const api = apiClient(endpoint);
    return api.listServices(listServicesRequestFromObject(request));
  });
}

/**
 *
 * @param {Partial<PullServicesRequest.AsObject>} request
 * @param {ResourceCollection<Service.AsObject, Service>} resource
 */
export function pullServices(request, resource) {
  pullResource('Services.PullServices', resource, endpoint => {
    const api = apiClient(endpoint);
    const stream = api.pullServices(pullServicesRequestFromObject(request));
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
 * @param {Partial<ConfigureServiceRequest.AsObject>} request
 * @param {ActionTracker<Service.AsObject>} [tracker]
 * @return {Promise<Service.AsObject>}
 */
export function configureService(request, tracker) {
  return trackAction('ServicesApi.ConfigureService', tracker ?? {}, endpoint => {
    const api = apiClient(endpoint);
    return api.configureService(configureServiceRequestFromObject(request));
  });
}

/**
 * @param {Partial<StartServiceRequest.AsObject>} request
 * @param {ActionTracker<Service.AsObject>} [tracker]
 * @return {Promise<Service.AsObject>}
 */
export function startService(request, tracker) {
  return trackAction('ServicesApi.StartService', tracker ?? {}, endpoint => {
    const api = apiClient(endpoint);
    return api.startService(startServiceRequestFromObject(request));
  });
}

/**
 * @param {Partial<StopServiceRequest.AsObject>} request
 * @param {ActionTracker<Service.AsObject>} [tracker]
 * @return {Promise<Service.AsObject>}
 */
export function stopService(request, tracker) {
  return trackAction('ServicesApi.stopService', tracker ?? {}, endpoint => {
    const api = apiClient(endpoint);
    return api.stopService(stopServiceRequestFromObject(request));
  });
}

/**
 * @typedef {Service.AsObject & {
 *   lastInactiveTime: Date,
 *   lastActiveTime: Date,
 *   lastLoadingStartTime: Date,
 *   lastLoadingEndTime: Date,
 *   lastErrorTime: Date,
 *   lastConfigTime: Date,
 *   nextAttemptTime: Date,
 * }} ServiceAsObject
 */

/**
 * @param {Service | Service.AsObject | null | undefined} service
 * @return {ServiceAsObject}
 */
export function serviceToObject(service) {
  if (!service) return undefined;
  if (Object.hasOwn(service, 'toObject')) service = service.toObject();
  const obj = {...service};
  const dates = [
    'lastInactiveTime', 'lastActiveTime',
    'lastLoadingStartTime', 'lastLoadingEndTime',
    'lastErrorTime', 'lastConfigTime', 'nextAttemptTime'
  ];
  for (const key of dates) {
    if (obj[key]) obj[key] = timestampToDate(obj[key]);
  }
  return obj;
}

/**
 * @param {string} endpoint
 * @return {ServicesApiPromiseClient}
 */
function apiClient(endpoint) {
  return new ServicesApiPromiseClient(endpoint, null, clientOptions());
}

/**
 * @param {Partial<GetServiceRequest.AsObject>} obj
 * @return {GetServiceRequest|undefined}
 */
function getServiceRequestFromObject(obj) {
  if (!obj) return undefined;
  const req = new GetServiceRequest();
  setProperties(req, obj, 'name', 'id');
  req.setReadMask(fieldMaskFromObject(obj.readMask));
  return req;
}

/**
 * @param {Partial<PullServicesRequest.AsObject>} obj
 * @return {PullServicesRequest|undefined}
 */
function pullServiceRequestFromObject(obj) {
  if (!obj) return undefined;
  const req = new PullServiceRequest();
  setProperties(req, obj, 'name', 'id', 'updatesOnly');
  req.setReadMask(fieldMaskFromObject(obj.readMask));
  return req;
}

/**
 * @param {Partial<GetMetadataRequest.AsObject>} obj
 * @return {GetMetadataRequest}
 */
function getServiceMetadataRequestFromObject(obj) {
  if (!obj) return undefined;
  const req = new GetMetadataRequest();
  setProperties(req, obj, 'name', 'id');
  req.setReadMask(fieldMaskFromObject(obj.readMask));
  return req;
}

/**
 * @param {Partial<PullServiceMetadataRequest.AsObject>} obj
 * @return {PullServicesRequest}
 */
function pullServiceMetadataRequestFromObject(obj) {
  if (!obj) return undefined;
  const req = new PullServiceMetadataRequest();
  setProperties(req, obj, 'name', 'id', 'updatesOnly');
  req.setReadMask(fieldMaskFromObject(obj.readMask));
  return req;
}

/**
 * @param {Partial<ListServicesRequest.AsObject>} obj
 * @return {ListServicesRequest}
 */
function listServicesRequestFromObject(obj) {
  if (!obj) return undefined;
  const req = new ListServicesRequest();
  setProperties(req, obj, 'name', 'pageToken', 'pageSize');
  req.setReadMask(fieldMaskFromObject(obj.readMask));
  return req;
}

/**
 * @param {Partial<PullServicesRequest.AsObject>} obj
 * @return {PullServicesRequest}
 */
function pullServicesRequestFromObject(obj) {
  if (!obj) return undefined;
  const req = new PullServicesRequest();
  setProperties(req, obj, 'name', 'updatesOnly');
  req.setReadMask(fieldMaskFromObject(obj.readMask));
  return req;
}

/**
 * @param {Partial<ConfigureServiceRequest.AsObject>} obj
 * @return {ConfigureServiceRequest}
 */
function configureServiceRequestFromObject(obj) {
  if (!obj) return undefined;
  const req = new ConfigureServiceRequest();
  setProperties(req, obj, 'name', 'id', 'configRaw');
  return req;
}

/**
 * @param {Partial<StartServiceRequest.AsObject>} obj
 * @return {StartServiceRequest}
 */
function startServiceRequestFromObject(obj) {
  if (!obj) return undefined;
  const req = new StartServiceRequest();
  setProperties(req, obj, 'name', 'id', 'allowActive');
  return req;
}

/**
 * @param {Partial<StopServiceRequest.AsObject>} obj
 * @return {StopServiceRequest}
 */
function stopServiceRequestFromObject(obj) {
  if (!obj) return undefined;
  const req = new StopServiceRequest();
  setProperties(req, obj, 'name', 'id', 'allowActive');
  return req;
}


export const ServiceNames = {
  Automations: 'automations',
  Drivers: 'drivers',
  Systems: 'systems',
  Zones: 'zones'
};
