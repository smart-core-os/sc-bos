import {fieldMaskFromObject, setProperties} from '@/api/convpb';
import {clientOptions} from '@/api/grpcweb.js';
import {pullResource, setValue} from '@/api/resource.js';
import {OpenCloseApiPromiseClient} from '@smart-core-os/sc-bos-ui-gen/proto/smartcore/bos/openclose/v1/open_close_grpc_web_pb';
import {PullOpenClosePositionsRequest} from '@smart-core-os/sc-bos-ui-gen/proto/smartcore/bos/openclose/v1/open_close_pb';

/**
 * @param {Partial<PullOpenClosePositionsRequest.AsObject>} request
 * @param {ResourceValue<OpenClosePositions.AsObject, PullOpenClosePositionsResponse>} resource
 */
export function pullOpenClosePositions(request, resource) {
  pullResource('OpenClose.pullOpenClosePositions', resource, endpoint => {
    const api = apiClient(endpoint);
    const stream = api.pullPositions(pullOpenClosePositionsRequestFromObject(request));
    stream.on('data', msg => {
      const changes = msg.getChangesList();
      for (const change of changes) {
        setValue(resource, change.getOpenClosePosition().toObject());
      }
    });
    return stream;
  });
}

/**
 * @param {string} endpoint
 * @return {OpenCloseApiPromiseClient}
 */
function apiClient(endpoint) {
  return new OpenCloseApiPromiseClient(endpoint, null, clientOptions());
}

/**
 * @param {Partial<PullOpenClosePositionsRequest.AsObject>} obj
 * @return {undefined|PullOpenClosePositionsRequest}
 */
function pullOpenClosePositionsRequestFromObject(obj) {
  if (!obj) return undefined;

  const dst = new PullOpenClosePositionsRequest();
  setProperties(dst, obj, 'name', 'excludeTweening', 'updatesOnly');
  dst.setReadMask(fieldMaskFromObject(obj.readMask));
  return dst;
}
