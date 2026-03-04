import {StatusCode} from 'grpc-web';

/**
 *
 * @param {string} message
 * @return {string}
 */
export function formatErrorMessage(message) {
  // Ensure that message is a string
  if (typeof message !== 'string') {
    // Handle non-string message appropriately
    // For example, return a default error message or convert message to a string if possible
    return `Invalid error message format: ${message}`;
  }

  // Finding the start of the description and extracting it
  const descStart = message.indexOf('desc = ');
  const description = message.substring(descStart).split('"')[1];

  if (!description) {
    // Handle missing description appropriately
    // For example, return a default error message or return the original message
    return message;
  }


  // Breaking down the description into parts
  const descParts = description.split(': ');
  // The last part is usually the main error detail
  const specificError = descParts[descParts.length - 1].trim();

  // Formatting the message into a more structured and readable sentence
  return `${specificError}.`;
}

/**
 * Return whether the given error looks like it's caused by a network error.
 *
 * @param {ResourceError | RpcError | any} err
 * @return {boolean}
 */
export function isNetworkError(err) {
  if (!err) return false;
  if (err.error) err = err.error;
  return err?.code === StatusCode.UNKNOWN;
}


