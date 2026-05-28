import {fieldMaskFromObject, setProperties} from '@/api/convpb.js';
import {clientOptions} from '@/api/grpcweb.js';
import {pullResource, setValue} from '@/api/resource.js';
import {DataRetentionApiPromiseClient} from
  '@smart-core-os/sc-bos-ui-gen/proto/smartcore/bos/dataretention/v1/data_retention_grpc_web_pb';
import {PullDataRetentionRequest} from
  '@smart-core-os/sc-bos-ui-gen/proto/smartcore/bos/dataretention/v1/data_retention_pb';


/**
 * @param {Partial<PullDataRetentionRequest.AsObject>} request
 * @param {ResourceValue<DataRetention.AsObject, PullDataRetentionResponse>} resource
 */
export function pullDataRetention(request, resource) {
  pullResource('DataRetentionApi.pullDataRetention', resource, endpoint => {
    const api = apiClient(endpoint);
    const stream = api.pullDataRetention(pullRequestFromObject(request));
    stream.on('data', msg => {
      for (const change of msg.getChangesList()) {
        setValue(resource, change.getDataRetention().toObject());
      }
    });
    return stream;
  });
}

/**
 * @param {string} endpoint
 * @return {DataRetentionApiPromiseClient}
 */
function apiClient(endpoint) {
  return new DataRetentionApiPromiseClient(endpoint, null, clientOptions());
}

/**
 * @param {Partial<PullDataRetentionRequest.AsObject>} obj
 * @return {PullDataRetentionRequest|undefined}
 */
function pullRequestFromObject(obj) {
  if (!obj) return undefined;
  const dst = new PullDataRetentionRequest();
  setProperties(dst, obj, 'name', 'updatesOnly');
  dst.setReadMask(fieldMaskFromObject(obj.readMask));
  return dst;
}
