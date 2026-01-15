import {
  convertProperties,
  fieldMaskFromObject,
  setProperties,
  timestampFromObject,
  timestampToDate
} from '@/api/convpb.js';
import {clientOptions} from '@/api/grpcweb.js';
import {pullResource, setValue} from '@/api/resource';
import {trackAction} from '@/api/resource.js';
import {TenantApiPromiseClient} from '@smart-core-os/sc-bos-ui-gen/proto/smartcore/bos/tenant/v1/tenants_grpc_web_pb';
import {
  AddTenantZonesRequest,
  CreateSecretRequest,
  CreateTenantRequest,
  DeleteSecretRequest,
  DeleteTenantRequest,
  GetTenantRequest,
  ListSecretsRequest,
  ListTenantsRequest,
  PullTenantRequest,
  RemoveTenantZonesRequest,
  Secret,
  Tenant,
  UpdateTenantRequest
} from '@smart-core-os/sc-bos-ui-gen/proto/smartcore/bos/tenant/v1/tenants_pb';

/**
 * @param {Partial<ListTenantsRequest.AsObject>} request
 * @param {ActionTracker<ListTenantsResponse.AsObject>} [tracker]
 * @return {Promise<ListTenantsResponse.AsObject>}
 */
export function listTenants(request, tracker) {
  return trackAction('Tenant.listTenants', tracker ?? {}, endpoint => {
    const api = client(endpoint);
    return api.listTenants(new ListTenantsRequest());
  });
}

/**
 * @param {Partial<PullTenantRequest.AsObject>} request
 * @param {ResourceValue<Tenant.AsObject, PullTenantResponse>} resource
 */
export function pullTenant(request, resource) {
  pullResource('Tenant.pullTenant', resource, endpoint => {
    const api = client(endpoint);
    const stream = api.pullTenant(pullTenantRequestFromObject(request));
    stream.on('data', msg => {
      const changes = msg.getChangesList();
      for (const change of changes) {
        setValue(resource, change.getTenant().toObject());
      }
    });
    return stream;
  });
}

/**
 *
 * @param {Partial<CreateTenantRequest.AsObject>} request
 * @param {ActionTracker<CreateTenantRequest.AsObject>} [tracker]
 * @return {Promise<Tenant.AsObject>}
 */
export function createTenant(request, tracker) {
  return trackAction('Tenant.createTenant', tracker ?? {}, endpoint => {
    const api = client(endpoint);
    return api.createTenant(createTenantRequestFromObject(request));
  });
}

/**
 *
 * @param {Partial<UpdateTenantRequest.AsObject>} request
 * @param {ActionTracker<CreateTenantRequest.AsObject>} [tracker]
 * @return {Promise<Tenant.AsObject>}
 */
export function updateTenant(request, tracker) {
  return trackAction('Tenant.updateTenant', tracker ?? {}, endpoint => {
    const api = client(endpoint);
    return api.updateTenant(updateTenantRequestFromObject(request));
  });
}

/**
 *
 * @param {Partial<DeleteTenantRequest.AsObject>} obj
 * @param {ActionTracker<DeleteTenantRequest.AsObject>} [tracker]
 * @return {Promise<DeleteTenantResponse>}
 */
export function deleteTenant(obj, tracker) {
  return trackAction('Tenant.deleteTenant', tracker ?? {}, endpoint => {
    const api = client(endpoint);
    return api.deleteTenant(deleteTenantRequestFromObject(obj));
  });
}

/**
 *
 * @param {Partial<GetTenantRequest.AsObject>} request
 * @param {ActionTracker<GetTenantRequest.AsObject>} [tracker]
 * @return {Promise<Tenant.AsObject>}
 */
export function getTenant(request, tracker) {
  const id = String(request.id);
  if (!id) throw new Error('request.id must be specified');
  return trackAction('Tenant.getTenant', tracker ?? {}, endpoint => {
    const api = client(endpoint);
    return api.getTenant(new GetTenantRequest().setId(id));
  });
}

/**
 *
 * @param {Partial<ListSecretsRequest.AsObject>} request
 * @param {ActionTracker<ListSecretsRequest.AsObject>} [tracker]
 * @return {Promise<ListSecretsResponse.AsObject>}
 */
export function listSecrets(request, tracker) {
  const tenantId = String(request.tenantId);
  if (!tenantId) throw new Error('request.tenantId must be specified');
  return trackAction('Tenant.listSecrets', tracker ?? {}, endpoint => {
    const api = client(endpoint);
    return api.listSecrets(new ListSecretsRequest().setFilter(`tenant.id=${tenantId}`));
  });
}

/**
 *
 * @param {Partial<CreateSecretRequest.AsObject>} request
 * @param {ActionTracker<CreateSecretRequest.AsObject>} [tracker]
 * @return {Promise<Secret.AsObject>}
 */
export function createSecret(request, tracker) {
  const secret = request.secret;
  if (!secret) throw new Error('request.secret must be specified');
  return trackAction('Tenant.createSecret', tracker ?? {}, endpoint => {
    const api = client(endpoint);
    return api.createSecret(new CreateSecretRequest().setSecret(secretFromObject(secret)));
  });
}

/**
 *
 * @param {Partial<DeleteSecretRequest.AsObject>} request
 * @param {ActionTracker<DeleteSecretRequest.AsObject>} [tracker]
 * @return {Promise<DeleteSecretResponse.AsObject>}
 */
export function deleteSecret(request, tracker) {
  const secretId = request.id;
  if (!secretId) throw new Error('request.id must be specified');
  return trackAction('Tenant.deleteSecret', tracker ?? {}, endpoint => {
    const api = client(endpoint);
    return api.deleteSecret(new DeleteSecretRequest().setId(secretId));
  });
}

/**
 * @param {Partial<AddTenantZonesRequest.AsObject>} request
 * @param {ActionTracker<AddTenantZonesRequest.AsObject>} [tracker]
 * @return {Promise<Tenant.AsObject>}
 */
export function addTenantZones(request, tracker) {
  return trackAction('Tenant.addTenantZones', tracker ?? {}, endpoint => {
    const api = client(endpoint);
    return api.addTenantZones(addTenantZonesRequestFromObject(request));
  });
}

/**
 * @param {Partial<RemoveTenantZonesRequest.AsObject>} request
 * @param {ActionTracker<RemoveTenantZonesRequest.AsObject>} [tracker]
 * @return {Promise<Tenant.AsObject>}
 */
export function removeTenantZones(request, tracker) {
  return trackAction('Tenant.removeTenantZones', tracker ?? {}, endpoint => {
    const api = client(endpoint);
    return api.removeTenantZones(removeTenantZonesRequestFromObject(request));
  });
}

/**
 * @param {string} endpoint
 * @return {TenantApiPromiseClient}
 */
function client(endpoint) {
  return new TenantApiPromiseClient(endpoint, null, clientOptions());
}

/**
 * @param {Partial<PullTenantRequest.AsObject>} obj
 * @return {undefined|PullTenantRequest}
 */
function pullTenantRequestFromObject(obj) {
  if (!obj) return undefined;

  const dst = new PullTenantRequest();
  setProperties(dst, obj, 'id');
  return dst;
}

/**
 * @param {Partial<CreateTenantRequest.AsObject>} obj
 * @return {CreateTenantRequest}
 */
function createTenantRequestFromObject(obj) {
  if (!obj) return undefined;

  const req = new CreateTenantRequest();
  req.setTenant(tenantFromObject(obj.tenant));
  return req;
}

/**
 * @param {Partial<UpdateTenantRequest.AsObject>} obj
 * @return {UpdateTenantRequest}
 */
function updateTenantRequestFromObject(obj) {
  if (!obj) return undefined;

  const req = new UpdateTenantRequest();
  req.setTenant(tenantFromObject(obj.tenant));
  req.setUpdateMask(fieldMaskFromObject(obj.updateMask));
  return req;
}

/**
 *
 * @param {Partial<DeleteTenantRequest.AsObject>} obj
 * @return {DeleteTenantRequest}
 */
function deleteTenantRequestFromObject(obj) {
  if (!obj) return undefined;

  const req = new DeleteTenantRequest();
  setProperties(req, obj, 'id');
  return req;
}

/**
 *
 * @param {Partial<Secret.AsObject>} obj
 * @return {Secret}
 */
function secretFromObject(obj) {
  if (!obj) return undefined;
  const dst = new Secret();
  setProperties(dst, obj, 'id', 'secret', 'etag', 'note', 'secretHash');
  convertProperties(dst, obj, timestampFromObject, 'expireTime', 'firstUseTime', 'lastUseTime');
  dst.setTenant(tenantFromObject(obj.tenant));
  return dst;
}

/**
 * @param {Secret|Secret.AsObject|null} s
 * @return {Secret.AsObject&{createTime?: Date, expireTime?:Date, lastUseTime?:Date, firstUseTime?:Date} | null}
 */
export function secretToObject(s) {
  if (!s) return null;

  const res = {...s};
  for (const prop of ['createTime', 'expireTime', 'firstUseTime', 'lastUseTime']) {
    if (s[prop]) {
      res[prop] = timestampToDate(s[prop]);
    }
  }
  return res;
}

/**
 * @param {Partial<Tenant.AsObject>} obj
 * @return {Tenant}
 */
function tenantFromObject(obj) {
  if (!obj) return undefined;
  const dst = new Tenant();
  setProperties(dst, obj, 'id', 'title', 'etag', 'zoneNamesList');
  dst.setCreateTime(timestampFromObject(obj.createTime));
  return dst;
}

/**
 * @param {Partial<AddTenantZonesRequest.AsObject>} obj
 * @return {undefined|AddTenantZonesRequest}
 */
function addTenantZonesRequestFromObject(obj) {
  if (!obj) return undefined;
  const dst = new AddTenantZonesRequest();
  setProperties(dst, obj, 'tenantId', 'addZoneNamesList');
  return dst;
}

/**
 * @param {Partial<RemoveTenantZonesRequest.AsObject>} obj
 * @return {undefined|RemoveTenantZonesRequest}
 */
function removeTenantZonesRequestFromObject(obj) {
  if (!obj) return undefined;
  const dst = new RemoveTenantZonesRequest();
  setProperties(dst, obj, 'tenantId', 'removeZoneNamesList');
  return dst;
}
