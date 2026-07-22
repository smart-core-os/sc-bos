import {setProperties} from '@/api/convpb';
import {clientOptions} from '@/api/grpcweb';
import {trackAction} from '@/api/resource';
import {UdmiExportApiPromiseClient} from '@smart-core-os/sc-bos-ui-gen/proto/smartcore/bos/udmi/v1/udmi_grpc_web_pb';
import {ListExportedPointsRequest} from '@smart-core-os/sc-bos-ui-gen/proto/smartcore/bos/udmi/v1/udmi_pb';

/**
 * Lists the distinct messages a udmi automation has published, for a points list export.
 *
 * @param {Partial<ListExportedPointsRequest.AsObject>} request - must have a name (the udmi automation name)
 * @param {ActionTracker<ListExportedPointsResponse.AsObject>?} tracker
 * @return {Promise<ListExportedPointsResponse.AsObject>}
 */
export function listExportedPoints(request, tracker) {
  return trackAction('UdmiExportApi.ListExportedPoints', tracker ?? {}, endpoint => {
    const api = apiClient(endpoint);
    return api.listExportedPoints(listExportedPointsRequestFromObject(request));
  });
}

/**
 * @param {string} endpoint
 * @return {UdmiExportApiPromiseClient}
 */
function apiClient(endpoint) {
  return new UdmiExportApiPromiseClient(endpoint, null, clientOptions());
}

/**
 * @param {Partial<ListExportedPointsRequest.AsObject>} obj
 * @return {ListExportedPointsRequest|undefined}
 */
function listExportedPointsRequestFromObject(obj) {
  if (!obj) return undefined;
  const req = new ListExportedPointsRequest();
  setProperties(req, obj, 'name');
  return req;
}
