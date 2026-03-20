import {fieldMaskFromObject, setProperties} from '@/api/convpb.js';
import {clientOptions} from '@/api/grpcweb.js';
import {pullResource, setCollection, trackAction} from '@/api/resource.js';
import {PublicationApiPromiseClient} from '@smart-core-os/sc-bos-ui-gen/proto/smartcore/bos/publication/v1/publication_grpc_web_pb';
import {
  AcknowledgePublicationRequest,
  CreatePublicationRequest,
  Publication,
  PullPublicationsRequest,
  UpdatePublicationRequest
} from '@smart-core-os/sc-bos-ui-gen/proto/smartcore/bos/publication/v1/publication_pb';

/**
 * @param {Partial<PullPublicationsRequest.AsObject>} request
 * @param {ResourceCollection<Publication.AsObject, OpenClosePositions>} resource
 */
export function pullPublications(request, resource) {
  pullResource('Publication.pullPublications', resource, endpoint => {
    const api = apiClient(endpoint);
    const stream = api.pullPublications(pullPublicationsRequestFromObject(request));
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
 * @param {Partial<CreatePublicationRequest.AsObject>} request
 * @param {ActionTracker<Publication.AsObject>} [tracker]
 * @return {Promise<Publication.AsObject>}
 */
export function createPublication(request, tracker) {
  return trackAction('Publication.createPublication', tracker ?? {}, endpoint => {
    const api = apiClient(endpoint);
    return api.createPublication(createPublicationRequestFromObject(request));
  });
}

/**
 * @param {Partial<UpdatePublicationRequest.AsObject>} request
 * @param {ActionTracker<Publication.AsObject>} [tracker]
 * @return {Promise<Publication.AsObject>}
 */
export async function updatePublication(request, tracker) {
  return trackAction('Publication.updatePublication', tracker ?? {}, endpoint => {
    const api = apiClient(endpoint);
    return api.updatePublication(updatePublicationRequestFromObject(request));
  });
}

/**
 * @param {Partial<AcknowledgePublicationRequest.AsObject>} request
 * @param {ActionTracker<Publication.AsObject>} [tracker]
 * @return {Promise<Publication.AsObject>}
 */
export async function acknowledgePublication(request, tracker) {
  return trackAction('Publication.acknowledgePublication', tracker ?? {}, endpoint => {
    const api = apiClient(endpoint);
    return api.acknowledgePublication(acknowledgePublicationRequestFromObject(request));
  });
}

/**
 * @param {string} endpoint
 * @return {PublicationApiPromiseClient}
 */
function apiClient(endpoint) {
  return new PublicationApiPromiseClient(endpoint, null, clientOptions());
}

/**
 * @param {Partial<PullPublicationsRequest.AsObject>} obj
 * @return {undefined|PullPublicationsRequest}
 */
function pullPublicationsRequestFromObject(obj) {
  if (!obj) return undefined;

  const dst = new PullPublicationsRequest();
  setProperties(dst, obj, 'name', 'updatesOnly');
  dst.setReadMask(fieldMaskFromObject(obj.readMask));
  return dst;
}

/**
 * @param {Partial<CreatePublicationRequest.AsObject>} obj
 * @return {CreatePublicationRequest|undefined}
 */
function createPublicationRequestFromObject(obj) {
  if (!obj) return undefined;

  const dst = new CreatePublicationRequest();
  setProperties(dst, obj, 'name');
  dst.setPublication(publicationFromObject(obj.publication));
  return dst;
}

/**
 * @param {Partial<UpdatePublicationRequest.AsObject>} obj
 * @return {undefined|UpdatePublicationRequest}
 */
function updatePublicationRequestFromObject(obj) {
  if (!obj) return undefined;

  const dst = new UpdatePublicationRequest();
  setProperties(dst, obj, 'name', 'version');
  dst.setUpdateMask(fieldMaskFromObject(obj.updateMask));
  dst.setPublication(publicationFromObject(obj.publication));
  return dst;
}

/**
 * @param {Partial<UpdatePublicationRequest.AsObject>} obj
 * @return {AcknowledgePublicationRequest|undefined}
 */
function acknowledgePublicationRequestFromObject(obj) {
  if (!obj) return undefined;

  const dst = new AcknowledgePublicationRequest();
  setProperties(dst, obj, 'name', 'id', 'version', 'receipt', 'receiptRejectedReason', 'allowAcknowledged');
  return dst;
}

/**
 * @param {Partial<Publication.AsObject>} obj
 * @return {undefined|Publication}
 */
export function publicationFromObject(obj) {
  if (!obj) return undefined;

  const publication = new Publication();
  setProperties(publication, obj, 'id', 'body', 'mediaType', 'version');
  publication.setAudience(audienceFromObject(obj.audience));
  return publication;
}

/**
 * @param {Partial<Publication.Audience.AsObject>} obj
 * @return {undefined|Publication.Audience}
 */
export function audienceFromObject(obj) {
  if (!obj) return undefined;
  const dst = new Publication.Audience();
  setProperties(dst, obj, 'name', 'receipt', 'receiptRejectedReason');
  return dst;
}
