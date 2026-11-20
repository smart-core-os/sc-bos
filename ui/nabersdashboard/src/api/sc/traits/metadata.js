import {trackAction} from '@/api/resource.js';
import {clientOptions} from '@/api/grpcweb.js';
import {MetadataApiPromiseClient} from '@smart-core-os/sc-bos-ui-gen/proto/smartcore/bos/metadata/v1/metadata_grpc_web_pb';
import {GetMetadataRequest} from '@smart-core-os/sc-bos-ui-gen/proto/smartcore/bos/metadata/v1/metadata_pb';

/**
 * Fetch the device metadata for the given name.
 *
 * A meter's human-readable name, its location and its unique ref all live here
 * rather than in the dashboard's own config, which is what lets one config list
 * meters by Smart Core name and still present them as the building's own labels.
 *
 * @param {string} name
 * @return {Promise<Metadata.AsObject>}
 */
export async function getMetadata(name) {
  const tracker = {loading: false, response: null, error: null};
  const req = new GetMetadataRequest();
  req.setName(name);
  return trackAction('Metadata.getMetadata', tracker, (endpoint) => {
    const client = new MetadataApiPromiseClient(endpoint, null, clientOptions());
    return client.getMetadata(req);
  });
}
