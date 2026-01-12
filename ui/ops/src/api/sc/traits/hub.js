import {setProperties} from '@/api/convpb';
import {clientOptions} from '@/api/grpcweb';
import {pullResource, setCollection, trackAction} from '@/api/resource';
import {HubApiPromiseClient} from '@smart-core-os/sc-bos-ui-gen/proto/smartcore/bos/hub/v1/hub_grpc_web_pb';
import {
  EnrollHubNodeRequest,
  ForgetHubNodeRequest,
  HubNode,
  InspectHubNodeRequest,
  ListHubNodesRequest,
  PullHubNodesRequest,
  TestHubNodeRequest
} from '@smart-core-os/sc-bos-ui-gen/proto/smartcore/bos/hub/v1/hub_pb';

/**
 *
 * @param {ActionTracker<ListHubNodesResponse.AsObject>} [tracker]
 * @return {Promise<ListHubNodesResponse.AsObject>}
 */
export function listHubNodes(tracker) {
  return trackAction('Hub.listHubNodes', tracker ?? {}, endpoint => {
    const api = apiClient(endpoint);
    return api.listHubNodes(new ListHubNodesRequest());
  });
}

/**
 *
 * @param {Partial<PullHubNodesRequest.AsObject>} req
 * @param {ResourceCollection<HubNode.AsObject, PullHubNodesResponse>} resource
 */
export function pullHubNodes(req, resource) {
  pullResource('Hub.pullHubNodes', resource, endpoint => {
    const api = apiClient(endpoint);
    const stream = api.pullHubNodes(pullHubNodesRequestFromObject(req));
    stream.on('data', msg => {
      const changes = msg.getChangesList();
      for (const change of changes) {
        setCollection(resource, change, v => v.address);
      }
    });
    return stream;
  });
}


/**
 *
 * @param {Partial<EnrollHubNodeRequest.AsObject>} request
 * @param {ActionTracker<HubNode.AsObject>} [tracker]
 * @return {Promise<HubNode.AsObject>}
 */
export function enrollHubNode(request, tracker) {
  return trackAction('Hub.enrollHubNode', tracker ?? {}, endpoint => {
    const api = apiClient(endpoint);
    const hubNode = enrollHubNodeRequestFromObject(request);
    return api.enrollHubNode(hubNode);
  });
}

/**
 *
 * @param {Partial<ForgetHubNodeRequest.AsObject>} request
 * @param {ActionTracker<ForgetHubNodeResponse.AsObject>} [tracker]
 * @return {Promise<ForgetHubNodeResponse.AsObject>}
 */
export function forgetHubNode(request, tracker) {
  return trackAction('Hub.forgetHubNode', tracker ?? {}, endpoint => {
    const api = apiClient(endpoint);
    const hubNode = forgetHubNodeRequestFromObject(request);
    return api.forgetHubNode(hubNode);
  });
}

/**
 * @param {Partial<TestHubNodeRequest.AsObject>} request
 * @param {ActionTracker<TestHubNodeResponse.AsObject>} [tracker]
 * @return {Promise<TestHubNodeResponse.AsObject>}
 */
export function testHubNode(request, tracker) {
  return trackAction('Hub.testHubNode', tracker ?? {}, endpoint => {
    const api = apiClient(endpoint);
    return api.testHubNode(testHubNodeRequestFromObject(request));
  });
}

/**
 * @param {Partial<InspectHubNodeRequest.AsObject>} request
 * @param {ActionTracker<HubNode.AsObject>} [tracker]
 * @return {Promise<HubNode.AsObject>}
 */
export function inspectHubNode(request, tracker) {
  return trackAction('Hub.inspectHubNode', tracker ?? {}, endpoint => {
    const api = apiClient(endpoint);
    return api.inspectHubNode(inspectHubNodeRequestFromObject(request));
  });
}

/**
 * @param {string} endpoint
 * @return {HubApiPromiseClient}
 */
function apiClient(endpoint) {
  return new HubApiPromiseClient(endpoint, null, clientOptions());
}

// --------------------------- //
// ----- Test Hub Node ----- //
/**
 *
 * @param {Partial<TestHubNodeRequest.AsObject>} obj
 * @return {TestHubNodeRequest|undefined}
 */
function testHubNodeRequestFromObject(obj) {
  if (!obj) return undefined;

  const dst = new TestHubNodeRequest();
  dst.setAddress(obj.address);
  return dst;
}

/**
 *
 * @param {Partial<InspectHubNodeRequest.AsObject>} obj
 * @return {InspectHubNodeRequest|undefined}
 */
function inspectHubNodeRequestFromObject(obj) {
  if (!obj) return undefined;

  const dst = new InspectHubNodeRequest();
  dst.setNode(hubNodeFromObject(obj.node));
  return dst;
}

// --------------------------- //

// --------------------------- //
// ----- Enroll Hub Node ----- //
/**
 * @param {Partial<EnrollHubNodeRequest.AsObject>} obj
 * @return {HubNode.AsObject|undefined}
 */
function enrollHubNodeRequestFromObject(obj) {
  if (!obj) return undefined;

  const dst = new EnrollHubNodeRequest();
  setProperties(dst, obj, 'publicCertsList');
  dst.setNode(hubNodeFromObject(obj.node));
  return dst;
}


/**
 * @param {Partial<EnrollHubNodeRequest.AsObject>} obj
 * @return {HubNode.AsObject|undefined}
 */
function hubNodeFromObject(obj) {
  if (!obj) return undefined;

  const dst = new HubNode();
  setProperties(dst, obj, 'address', 'name', 'description');

  return dst;
}

// --------------------------- //


// --------------------------- //
// ----- Forget Hub Node ----- //
/**
 * @param {Partial<HubNode.AsObject>} obj
 * @return {ForgetHubNodeRequest|undefined}
 */
function forgetHubNodeRequestFromObject(obj) {
  if (!obj) return undefined;

  const dst = new ForgetHubNodeRequest();
  setProperties(dst, obj, 'address', 'allowMissing');

  return dst;
}

/**
 * @param {Partial<PullHubNodesRequest.AsObject>} obj
 * @return {undefined|PullHubNodesRequest}
 */
function pullHubNodesRequestFromObject(obj) {
  if (!obj) return undefined;

  const dst = new PullHubNodesRequest();
  setProperties(dst, obj, 'updatesOnly');

  return dst;
}

