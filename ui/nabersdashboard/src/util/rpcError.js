import {StatusCode} from 'grpc-web';

/** gRPC status codes by number, so a failure reason can be named. */
const STATUS_NAMES = Object.fromEntries(
  Object.entries(StatusCode).map(([name, code]) => [code, name])
);

/**
 * A short, readable reason for a failed meter read.
 *
 * gRPC-web errors routinely carry an empty `message`, so the status code is
 * often the only thing worth showing.
 *
 * @param {*} e
 * @return {string}
 */
export function describeRpcError(e) {
  if (e?.code === StatusCode.NOT_FOUND) return 'no history recorded';
  const name = STATUS_NAMES[e?.code];
  return name ? name.toLowerCase().replace(/_/g, ' ') : (e?.message || 'read failed');
}
