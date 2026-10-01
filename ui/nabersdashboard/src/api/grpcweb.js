import {StatusCode} from 'grpc-web';
import {useAuthStore} from '@/stores/auth.js';

/**
 * On auth failure, re-fetch the token so the next retry attempt gets a fresh token.
 *
 * @param {string} reason
 */
function handleLogout(reason) {
  console.warn('Auth error, re-fetching token:', reason);
  useAuthStore().fetchToken();
}

/**
 * Returns the current token from the auth store.
 *
 * @return {Promise<string|null>}
 */
function refreshToken() {
  return Promise.resolve(useAuthStore().token || null);
}

/**
 * @param {import('grpc-web').GrpcWebClientBaseOptions} [options]
 * @return {import('grpc-web').GrpcWebClientBaseOptions}
 */
export function clientOptions(options = {}) {
  const handleError = (e) => {
    switch (e.code) {
      case StatusCode.PERMISSION_DENIED:
        handleLogout('Permission denied');
        break;
      case StatusCode.UNAUTHENTICATED:
        handleLogout('Unauthenticated');
        break;
    }
  };

  const addRequestHeader = (request, invoker) => {
    return refreshToken().then(token => {
      if (token) {
        request.getMetadata()['Authorization'] = `Bearer ${token}`;
      }
      return request;
    }).then(request => {
      return invoker(request);
    });
  };

  return {
    ...options,
    unaryInterceptors: [
      ...(options.unaryInterceptors || []),
      {
        intercept(request, invoker) {
          return addRequestHeader(request, invoker).catch(e => {
            handleError(e);
            throw e;
          });
        }
      }],
    streamInterceptors: [
      ...(options.streamInterceptors || []),
      {
        intercept(request, invoker) {
          const s = new DelayedClientReadableStream(addRequestHeader(request, invoker));

          s.on('error', (err) => {
            handleError(err);
          });

          return s;
        }
      }
    ]
  };
}

/**
 * A ClientReadableStream that wraps a promise of a ClientReadableStream.
 *
 * @augments {ClientReadableStream}
 */
class DelayedClientReadableStream {
  /**
   * @param {Promise<ClientReadableStream>} other
   */
  constructor(other) {
    this.other = other;
  }

  on(eventType, callback) {
    this.other.then(o => o.on(eventType, callback));
    return this;
  }

  removeListener(eventType, callback) {
    this.other.then(o => o.removeListener(eventType, callback));
  }

  cancel() {
    this.other.then(o => o.cancel());
  }
}
