import {clientOptions} from '@/api/grpcweb.js';
import {trackAction} from '@/api/resource.js';
import {LogApiPromiseClient} from '@smart-core-os/sc-bos-ui-gen/proto/smartcore/bos/log/v1/log_grpc_web_pb';
import {
  GetDownloadLogUrlRequest,
  GetLogLevelRequest,
  LogLevel,
  PullLogLevelRequest,
  PullLogMessagesRequest,
  PullLogMetadataRequest,
  UpdateLogLevelRequest
} from '@smart-core-os/sc-bos-ui-gen/proto/smartcore/bos/log/v1/log_pb';

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
  stream.on('data', msg => {
    for (const change of msg.getChangesList()) {
      onMessages(change.getMessagesList().map(m => m.toObject()));
    }
  });
  if (onError) stream.on('error', onError);
  return stream;
}

/**
 * @param {Partial<import('@smart-core-os/sc-bos-ui-gen/proto/smartcore/bos/log/v1/log_pb').GetLogLevelRequest.AsObject>} request
 * @param {Object} [tracker]
 * @return {Promise<import('@smart-core-os/sc-bos-ui-gen/proto/smartcore/bos/log/v1/log_pb').LogLevel.AsObject>}
 */
export function getLogLevel(request, tracker) {
  return trackAction('Log.getLogLevel', tracker ?? {}, endpoint => {
    const api = new LogApiPromiseClient(endpoint, null, clientOptions());
    const req = new GetLogLevelRequest();
    if (request?.name) req.setName(request.name);
    return api.getLogLevel(req);
  });
}

/**
 * @param {string} endpoint
 * @param {Partial<import('@smart-core-os/sc-bos-ui-gen/proto/smartcore/bos/log/v1/log_pb').PullLogLevelRequest.AsObject>} request
 * @param {function(import('@smart-core-os/sc-bos-ui-gen/proto/smartcore/bos/log/v1/log_pb').LogLevel.AsObject): void} onLevel
 * @param {function(Error): void} [onError]
 * @return {import('grpc-web').ClientReadableStream}
 */
export function pullLogLevel(endpoint, request, onLevel, onError) {
  const api = new LogApiPromiseClient(endpoint, null, clientOptions());
  const req = new PullLogLevelRequest();
  if (request?.name) req.setName(request.name);
  const stream = api.pullLogLevel(req);
  stream.on('data', msg => {
    for (const change of msg.getChangesList()) {
      if (change.hasLogLevel()) {
        onLevel(change.getLogLevel().toObject());
      }
    }
  });
  if (onError) stream.on('error', onError);
  return stream;
}

/**
 * @param {Partial<{name: string, level: number}>} request
 * @param {Object} [tracker]
 * @return {Promise<import('@smart-core-os/sc-bos-ui-gen/proto/smartcore/bos/log/v1/log_pb').LogLevel.AsObject>}
 */
export function updateLogLevel(request, tracker) {
  return trackAction('Log.updateLogLevel', tracker ?? {}, endpoint => {
    const api = new LogApiPromiseClient(endpoint, null, clientOptions());
    const req = new UpdateLogLevelRequest();
    if (request?.name) req.setName(request.name);
    if (request?.level != null) {
      const ll = new LogLevel();
      ll.setLevel(request.level);
      req.setLogLevel(ll);
    }
    return api.updateLogLevel(req);
  });
}

/**
 * @param {string} endpoint
 * @param {Partial<import('@smart-core-os/sc-bos-ui-gen/proto/smartcore/bos/log/v1/log_pb').PullLogMetadataRequest.AsObject>} request
 * @param {function(import('@smart-core-os/sc-bos-ui-gen/proto/smartcore/bos/log/v1/log_pb').LogMetadata.AsObject): void} onMetadata
 * @param {function(Error): void} [onError]
 * @return {import('grpc-web').ClientReadableStream}
 */
export function pullLogMetadata(endpoint, request, onMetadata, onError) {
  const api = new LogApiPromiseClient(endpoint, null, clientOptions());
  const req = new PullLogMetadataRequest();
  if (request?.name) req.setName(request.name);
  const stream = api.pullLogMetadata(req);
  stream.on('data', msg => {
    for (const change of msg.getChangesList()) {
      if (change.hasLogMetadata()) {
        onMetadata(change.getLogMetadata().toObject());
      }
    }
  });
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
