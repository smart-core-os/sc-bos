import {grpcWebEndpoint} from '@/api/config.js';

/**
 * Closes any open server streams associated with the given resource.
 *
 * @param {RemoteResource<any>} resource
 */
export function closeResource(resource) {
  if (resource?.stream?.cancel) resource.stream.cancel();
  if (resource?.stream?.close) resource.stream.close();
  if (resource?.value) resource.value = null;
  if (resource?.updateTime) resource.updateTime = null;
}

/**
 * Sets a successful value on the given resource and resets any error or loading state.
 *
 * @param {ResourceValue<V, M>} resource
 * @param {V} val
 * @template V,M
 */
export function setValue(resource, val) {
  resource.loading = false;
  resource.streamError = null;
  resource.value = val;
  resource.updateTime = new Date();
}

/**
 * Set properties on resource to indicate that an error occurred.
 *
 * @param {RemoteResource<any,any>} resource
 * @param {Error} err
 * @param {string} name
 */
export function setError(resource, err, name = '') {
  const rErr = /** @type {RemoteError} */ {
    name,
    error: err
  };
  resource.loading = false;
  resource.streamError = rErr;
  resource.updateTime = new Date();
}

/**
 * Execute a PullFoo type RPC against a remote service that follows Smart Core patterns.
 *
 * @param {string} logPrefix
 * @param {RemoteResource<O, T>} resource
 * @param {StreamFactory<T>} newStream
 * @template T,O
 */
export function pullResource(logPrefix, resource, newStream) {
  const doPull = (retryDelayMs = 1000) => {
    let retryCalled = false;
    const retry = () => {
      if (retryCalled) return;
      retryCalled = true;

      const handle = setTimeout(() => {
        const delay = Math.max(1000, Math.min(retryDelayMs * 2, 15 * 1000));
        doPull(delay);
      }, retryDelayMs);
      resource.stream = {
        cancel() {
          clearTimeout(handle);
        }
      };
    };

    const address = grpcWebEndpoint();
    Promise.resolve(address)
        .then((endpoint) => {
          const stream = newStream(endpoint);
          resource.stream = stream;
          stream.on('data', (r) => {
            retryDelayMs = 1000;
            resource.lastResponse = r;
          });
          stream.on('error', (err) => {
            setError(resource, err, logPrefix);
            retry();
          });
          stream.on('end', () => {
            retry();
          });
        })
        .catch((err) => {
          setError(resource, err, logPrefix);
          retry();
        });
  };

  doPull(0);
}

/**
 * Execute a non-streaming RPC against the globally configured endpoint.
 *
 * @param {string} logPrefix
 * @param {ActionTracker<V>} tracker
 * @param {Action<V, M>} action
 * @return {Promise<V>}
 * @template V, M
 */
export async function trackAction(logPrefix, tracker, action) {
  tracker.loading = true;
  const endpoint = await grpcWebEndpoint();
  try {
    const msg = await action(endpoint);
    const value = msg.toObject();
    tracker.response = value;
    tracker.error = null;
    return value;
  } catch (err) {
    const rErr = /** @type {RemoteError} */ {
      name: logPrefix,
      error: err
    };
    tracker.error = rErr;
    throw err;
  } finally {
    tracker.loading = false;
  }
}

/**
 * @return {ResourceValue<V, M>}
 * @template V,M
 */
export function newResourceValue() {
  return {
    loading: false,
    stream: null,
    streamError: null,
    updateTime: null,
    value: null
  };
}
