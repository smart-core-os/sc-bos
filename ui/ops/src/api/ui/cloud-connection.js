import {clientOptions} from '@/api/grpcweb.js';
import {pullResource, setValue, trackAction} from '@/api/resource.js';
import {
  CloudConnectionApiPromiseClient
} from '@smart-core-os/sc-bos-ui-gen/proto/smartcore/bos/ops/cloud/v1alpha/cloud_connection_grpc_web_pb';
import {
  GetCloudConnectionDefaultsRequest,
  PullCloudConnectionRequest,
  RegisterCloudConnectionRequest,
  TestCloudConnectionRequest,
  UnlinkCloudConnectionRequest
} from '@smart-core-os/sc-bos-ui-gen/proto/smartcore/bos/ops/cloud/v1alpha/cloud_connection_pb';

export const CloudErrCode = {
  INVALID_ENROLLMENT_CODE:    'invalid_enrollment_code',
  INVALID_CLIENT_CREDENTIALS: 'invalid_client_credentials',
  SERVER_UNREACHABLE:         'server_unreachable',
  CREDENTIAL_CHECK_FAILED:    'credential_check_failed',
  CONNECTION_FAILED:          'connection_failed',
};

export const CloudErrMessage = {
  [CloudErrCode.INVALID_ENROLLMENT_CODE]:    'Enrollment code is invalid, expired or already used',
  [CloudErrCode.INVALID_CLIENT_CREDENTIALS]: 'Invalid client ID or secret',
  [CloudErrCode.SERVER_UNREACHABLE]:         'Could not reach the server — check the server address',
  [CloudErrCode.CREDENTIAL_CHECK_FAILED]:    'Credential check failed — verify client ID, secret and server address',
  [CloudErrCode.CONNECTION_FAILED]:          'Could not connect to the server',
};

/**
 * @param {string} endpoint
 * @return {CloudConnectionApiPromiseClient}
 */
function apiClient(endpoint) {
  return new CloudConnectionApiPromiseClient(endpoint, null, clientOptions());
}

/**
 * @param {Partial<PullCloudConnectionRequest.AsObject>} request
 * @param {ResourceValue<import('@smart-core-os/sc-bos-ui-gen/proto/smartcore/bos/ops/cloud/v1alpha/cloud_connection_pb').CloudConnection.AsObject, any>} resource
 */
export function pullCloudConnection(request, resource) {
  pullResource('CloudConnectionApi.PullCloudConnection', resource, endpoint => {
    const api = apiClient(endpoint);
    const req = new PullCloudConnectionRequest();
    req.setName(request.name ?? '');
    req.setUpdatesOnly(request.updatesOnly ?? false);
    const stream = api.pullCloudConnection(req);
    stream.on('data', msg => {
      const changes = msg.getChangesList();
      for (const change of changes) {
        const conn = change.getCloudConnection();
        if (conn) setValue(resource, conn.toObject());
      }
    });
    return stream;
  });
}

/**
 * @param {{name: string, enrollmentCode?: {code: string, registerUrl?: string}, manual?: {clientId: string, clientSecret: string, bosapiRoot: string}}} request
 * @param {ActionTracker<any>} [tracker]
 * @return {Promise<import('@smart-core-os/sc-bos-ui-gen/proto/smartcore/bos/ops/cloud/v1alpha/cloud_connection_pb').RegisterCloudConnectionResponse.AsObject>}
 */
export function registerCloudConnection(request, tracker) {
  return trackAction('CloudConnectionApi.RegisterCloudConnection', tracker ?? {}, endpoint => {
    const api = apiClient(endpoint);
    const req = new RegisterCloudConnectionRequest();
    req.setName(request.name ?? '');
    if (request.enrollmentCode) {
      const ec = new RegisterCloudConnectionRequest.EnrollmentCode();
      ec.setCode(request.enrollmentCode.code ?? '');
      ec.setRegisterUrl(request.enrollmentCode.registerUrl ?? '');
      req.setEnrollmentCode(ec);
    } else if (request.manual) {
      const mc = new RegisterCloudConnectionRequest.ManualCredentials();
      mc.setClientId(request.manual.clientId ?? '');
      mc.setClientSecret(request.manual.clientSecret ?? '');
      mc.setBosapiRoot(request.manual.bosapiRoot ?? '');
      req.setManual(mc);
    }
    return api.registerCloudConnection(req);
  });
}

/**
 * @param {Partial<import('@smart-core-os/sc-bos-ui-gen/proto/smartcore/bos/ops/cloud/v1alpha/cloud_connection_pb').GetCloudConnectionDefaultsRequest.AsObject>} request
 * @param {ActionTracker<any>} [tracker]
 * @return {Promise<import('@smart-core-os/sc-bos-ui-gen/proto/smartcore/bos/ops/cloud/v1alpha/cloud_connection_pb').GetCloudConnectionDefaultsResponse.AsObject>}
 */
export function getCloudConnectionDefaults(request, tracker) {
  return trackAction('CloudConnectionApi.GetCloudConnectionDefaults', tracker ?? {}, endpoint => {
    const api = apiClient(endpoint);
    const req = new GetCloudConnectionDefaultsRequest();
    req.setName(request.name ?? '');
    return api.getCloudConnectionDefaults(req);
  });
}

/**
 * @param {Partial<import('@smart-core-os/sc-bos-ui-gen/proto/smartcore/bos/ops/cloud/v1alpha/cloud_connection_pb').TestCloudConnectionRequest.AsObject>} request
 * @param {ActionTracker<any>} [tracker]
 * @return {Promise<import('@smart-core-os/sc-bos-ui-gen/proto/smartcore/bos/ops/cloud/v1alpha/cloud_connection_pb').TestCloudConnectionResponse.AsObject>}
 */
export function testCloudConnection(request, tracker) {
  return trackAction('CloudConnectionApi.TestCloudConnection', tracker ?? {}, endpoint => {
    const api = apiClient(endpoint);
    const req = new TestCloudConnectionRequest();
    req.setName(request.name ?? '');
    return api.testCloudConnection(req);
  });
}

/**
 * @param {Partial<import('@smart-core-os/sc-bos-ui-gen/proto/smartcore/bos/ops/cloud/v1alpha/cloud_connection_pb').UnlinkCloudConnectionRequest.AsObject>} request
 * @param {ActionTracker<any>} [tracker]
 * @return {Promise<import('@smart-core-os/sc-bos-ui-gen/proto/smartcore/bos/ops/cloud/v1alpha/cloud_connection_pb').UnlinkCloudConnectionResponse.AsObject>}
 */
export function unlinkCloudConnection(request, tracker) {
  return trackAction('CloudConnectionApi.UnlinkCloudConnection', tracker ?? {}, endpoint => {
    const api = apiClient(endpoint);
    const req = new UnlinkCloudConnectionRequest();
    req.setName(request.name ?? '');
    return api.unlinkCloudConnection(req);
  });
}
