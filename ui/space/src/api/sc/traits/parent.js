import {fieldMaskFromObject, setProperties} from '@/api/convpb';
import {clientOptions} from '@/api/grpcweb.js';
import {pullResource, setCollection} from '@/api/resource.js';
import {ParentApiPromiseClient} from '@smart-core-os/sc-bos-ui-gen/proto/smartcore/bos/parent/v1/parent_grpc_web_pb';
import {PullChildrenRequest} from '@smart-core-os/sc-bos-ui-gen/proto/smartcore/bos/parent/v1/parent_pb';

/**
 * @param {Partial<PullChildrenRequest.AsObject>} request
 * @param {ResourceCollection<Child.AsObject, PullChildrenResponse>} resources
 */
export function pullChildren(request, resources) {
  pullResource('Parent.pullChildren', resources, endpoint => {
    const api = apiClient(endpoint);
    const stream = api.pullChildren(pullChildrenRequestFromObject(request));
    stream.on('data', msg => {
      const changes = msg.getChangesList();
      for (const change of changes) {
        setCollection(resources, change, child => child.name);
      }
    });
    return stream;
  });
}

/**
 * @param {string} endpoint
 * @return {ParentApiPromiseClient}
 */
function apiClient(endpoint) {
  return new ParentApiPromiseClient(endpoint, null, clientOptions());
}

/**
 * @param {Partial<PullChildrenRequest.AsObject>} obj
 * @return {undefined|PullChildrenRequest}
 */
function pullChildrenRequestFromObject(obj) {
  if (!obj) return undefined;

  const dst = new PullChildrenRequest();
  setProperties(dst, obj, 'name', 'updatesOnly');
  dst.setReadMask(fieldMaskFromObject(obj.readMask));
  return dst;
}
