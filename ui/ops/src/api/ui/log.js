import {clientOptions} from '@/api/grpcweb.js';
import {pullResource, setValue, trackAction} from '@/api/resource.js';
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

const apiClient = (endpoint) => new LogApiPromiseClient(endpoint, null, clientOptions());

/**
 * Opens a server-streaming connection for log messages, with automatic retry.
 * Each incoming batch is delivered as resource.value (an array of LogMessage.AsObject).
 * The caller should accumulate batches; resource.value is the latest batch, not the full history.
 * Call closeResource(resource) to stop the stream.
 *
 * @param {Partial<PullLogMessagesRequest.AsObject>} request
 * @param {import('@/api/resource.js').ResourceValue} resource
 */
export function pullLogMessages(request, resource) {
  pullResource('Log.pullLogMessages', resource, endpoint => {
    const api = apiClient(endpoint);
    const req = new PullLogMessagesRequest();
    if (request?.name) req.setName(request.name);
    if (request?.initialCount) req.setInitialCount(request.initialCount);
    if (request?.updatesOnly) req.setUpdatesOnly(request.updatesOnly);
    const stream = api.pullLogMessages(req);
    stream.on('data', (msg) => {
      for (const change of msg.getChangesList()) {
        setValue(resource, change.getMessagesList().map(m => m.toObject()));
      }
    });
    return stream;
  });
}

/**
 * @param {Partial<import('@smart-core-os/sc-bos-ui-gen/proto/smartcore/bos/log/v1/log_pb').GetLogLevelRequest.AsObject>} request
 * @param {Object} [tracker]
 * @return {Promise<import('@smart-core-os/sc-bos-ui-gen/proto/smartcore/bos/log/v1/log_pb').LogLevel.AsObject>}
 */
export function getLogLevel(request, tracker) {
  return trackAction('Log.getLogLevel', tracker ?? {}, endpoint => {
    const api = apiClient(endpoint);
    const req = new GetLogLevelRequest();
    if (request?.name) req.setName(request.name);
    return api.getLogLevel(req);
  });
}

/**
 * Opens a server-streaming connection for log level changes, with automatic retry.
 * resource.value is set to the current level number (or null if unknown).
 * Call closeResource(resource) to stop the stream.
 *
 * @param {Partial<import('@smart-core-os/sc-bos-ui-gen/proto/smartcore/bos/log/v1/log_pb').PullLogLevelRequest.AsObject>} request
 * @param {import('@/api/resource.js').ResourceValue} resource
 */
export function pullLogLevel(request, resource) {
  pullResource('Log.pullLogLevel', resource, endpoint => {
    const api = apiClient(endpoint);
    const req = new PullLogLevelRequest();
    if (request?.name) req.setName(request.name);
    const stream = api.pullLogLevel(req);
    stream.on('data', (msg) => {
      for (const change of msg.getChangesList()) {
        setValue(resource, change.getLogLevel()?.getLevel() ?? null);
      }
    });
    return stream;
  });
}

/**
 * @param {Partial<{name: string, level: number}>} request
 * @param {Object} [tracker]
 * @return {Promise<import('@smart-core-os/sc-bos-ui-gen/proto/smartcore/bos/log/v1/log_pb').LogLevel.AsObject>}
 */
export function updateLogLevel(request, tracker) {
  return trackAction('Log.updateLogLevel', tracker ?? {}, endpoint => {
    const api = apiClient(endpoint);
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
 * Opens a server-streaming connection for log metadata changes, with automatic retry.
 * resource.value is set to the current LogMetadata.AsObject.
 * Call closeResource(resource) to stop the stream.
 *
 * @param {Partial<import('@smart-core-os/sc-bos-ui-gen/proto/smartcore/bos/log/v1/log_pb').PullLogMetadataRequest.AsObject>} request
 * @param {import('@/api/resource.js').ResourceValue} resource
 */
export function pullLogMetadata(request, resource) {
  pullResource('Log.pullLogMetadata', resource, endpoint => {
    const api = apiClient(endpoint);
    const req = new PullLogMetadataRequest();
    if (request?.name) req.setName(request.name);
    const stream = api.pullLogMetadata(req);
    stream.on('data', msg => {
      for (const change of msg.getChangesList()) {
        if (change.hasLogMetadata()) {
          setValue(resource, change.getLogMetadata().toObject());
        }
      }
    });
    return stream;
  });
}

/**
 * @param {Partial<GetDownloadLogUrlRequest.AsObject>} request
 * @param {ActionTracker<GetDownloadLogUrlResponse.AsObject>} [tracker]
 * @return {Promise<GetDownloadLogUrlResponse.AsObject>}
 */
export function getDownloadLogUrl(request, tracker) {
  return trackAction('Log.getDownloadLogUrl', tracker ?? {}, endpoint => {
    const api = apiClient(endpoint);
    const req = new GetDownloadLogUrlRequest();
    if (request?.name) req.setName(request.name);
    if (request?.includeRotated) req.setIncludeRotated(request.includeRotated);
    return api.getDownloadLogUrl(req);
  });
}
