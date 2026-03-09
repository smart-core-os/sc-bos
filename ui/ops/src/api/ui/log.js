import {clientOptions} from '@/api/grpcweb.js';
import {trackAction} from '@/api/resource.js';
import {LogApiPromiseClient} from '@smart-core-os/sc-bos-ui-gen/proto/smartcore/bos/log/v1/log_grpc_web_pb';
import {GetDownloadLogUrlRequest, PullLogMessagesRequest} from '@smart-core-os/sc-bos-ui-gen/proto/smartcore/bos/log/v1/log_pb';

/**
 * Opens a server-streaming connection for log messages.
 * Caller is responsible for calling stream.cancel() on cleanup.
 *
 * @param {string} endpoint
 * @param {Partial<PullLogMessagesRequest.AsObject>} request
 * @param {function(import('@smart-core-os/sc-bos-ui-gen/proto/smartcore/bos/log/v1/log_pb').LogMessage.AsObject[]): void} onMessages
 * @param {function(Error): void} [onError]
 * @return {import('grpc-web').ClientReadableStream}
 */
export function pullLogMessages(endpoint, request, onMessages, onError) {
  const api = new LogApiPromiseClient(endpoint, null, clientOptions());
  const req = new PullLogMessagesRequest();
  if (request?.name) req.setName(request.name);
  if (request?.initialCount) req.setInitialCount(request.initialCount);
  if (request?.updatesOnly) req.setUpdatesOnly(request.updatesOnly);
  const stream = api.pullLogMessages(req);
  stream.on('data', msg => onMessages(msg.getMessagesList().map(m => m.toObject())));
  if (onError) stream.on('error', onError);
  return stream;
}

/**
 * @param {Partial<GetDownloadLogUrlRequest.AsObject>} request
 * @param {ActionTracker<GetDownloadLogUrlResponse.AsObject>} [tracker]
 * @return {Promise<GetDownloadLogUrlResponse.AsObject>}
 */
export function getDownloadLogUrl(request, tracker) {
  return trackAction('Log.getDownloadLogUrl', tracker ?? {}, endpoint => {
    const api = new LogApiPromiseClient(endpoint, null, clientOptions());
    const req = new GetDownloadLogUrlRequest();
    if (request?.name) req.setName(request.name);
    if (request?.includeRotated) req.setIncludeRotated(request.includeRotated);
    return api.getDownloadLogUrl(req);
  });
}
