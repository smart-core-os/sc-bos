import {useAccountStore} from '@/stores/account';
import {StatusCode} from 'grpc-web';

/**
 * @param {import('grpc-web').GrpcWebClientBaseOptions} [options]
 * @return {import('grpc-web').GrpcWebClientBaseOptions}
 */
export function clientOptions(options = {}) {
  const account = useAccountStore();

  /**
   * Log the user out if we get a permission denied/unauthenticated error
   * (this going to clear the pinia state too)
   *
   * @param {string} reason
   */
  const handleLogout = (reason) => {
    account.logout(reason).catch(e => console.error('Logout failed', e));
  };

  /**
   * Log the user out if we get an auth error (this going to clear the pinia state too)
   *
   * @param {Error} e
   */
  const handleError = (e) => {
    switch (e.code) {
      case StatusCode.PERMISSION_DENIED:
        console.warn('Attempt to access a protected resource', e);
        break;
      case StatusCode.UNAUTHENTICATED:
        handleLogout('Unauthenticated');
        break;
    }
  };

  /**
   * Triggers a token refresh if necessary and returns the new token
   *
   * @return {Promise<string>}
   */
  const refreshToken = () => {
    return account.refreshToken().then(() => {
      return account.authenticationDetails.token;
    });
  };


  /**
   * @template T
   * @param {*} request
   * @param {function(*): T} invoker
   * @return {Promise<T>}
   */
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
 * A ClientReadableStream that wraps a promise of a ClientReadableStream. This is generally ok on the surface but some
 * subtle issue might arise if, for example, the close method throws an exception which won't be propagated to the
 * caller. Similarly we assume that p.then(a); p.then(b); invokes a then b in that order when p settles, otherwise some
 * calling code might not subscribe and unsubscribe is the right order.
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
