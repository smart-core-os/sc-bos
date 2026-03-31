import deepEqual from 'fast-deep-equal';
import {toValue, watch} from 'vue';

/**
 * @typedef {import('@/api/resource').RemoteResource} RemoteResource
 */

/**
 * Converts a query like value into a Smart Core query object.
 * Queries typically look like {name: 'someName'}, so if the passed input is a string it will be converted to
 * {name: input}.
 *
 * @template {{name: string}} T
 * @param {MaybeRefOrGetter<string|T|null>} input - input value to be converted into a query object.
 * @return {T|null} input or {name: input} if input is a string
 */
export const toQueryObject = (input) => {
  const inputValue = toValue(input);
  if (inputValue === null || inputValue === undefined) return null;
  if (typeof inputValue === 'string') return {name: inputValue};
  return inputValue;
};

/**
 * Sets the name of the request if it is not already set.
 *
 * @template {{name: string}} T
 * @param {T} req
 * @param {MaybeRefOrGetter<string>} name
 * @return {T}
 */
export const setRequestName = (req, name) => {
  const nameValue = toValue(name);
  const needsName = nameValue === null || nameValue === undefined;
  if (needsName && !Object.hasOwn(req, 'name')) {
    throw new Error('name is required as part of request');
  }
  if (!Object.hasOwn(req, 'name')) {
    req.name = nameValue;
  }
  return req;
};

/**
 * Calls apiCall each time the query changes, tracking and managing stop cleanup.
 *
 * @template T
 * @param {MaybeRefOrGetter<T>} query - object representing the request to the API
 * @param {MaybeRefOrGetter<boolean>} [paused] - boolean representing whether the data stream is paused
 * @param {(req: T) => () => {}} apiCall - a function that starts a bg task and returns a cleanup function
 * @example
 * watchResource(
 *   () => toValue(toQueryObject(query)),
 *   () => toValue(paused),
 *   (req) => {
 *     pullAirTemperature(req, resource);
 *     return () => closeResource(resource);
 *   }
 * );
 */
export const watchResource = (query, paused = false, apiCall) => {
  let stop = null;

  watch(
      [() => toValue(query), () => toValue(paused)],
      (newSource, oldSource) => {
        // Check if the props have changed
        const oldNewEqual = deepEqual(newSource, oldSource);

        // If the props haven't changed, do nothing
        if (oldNewEqual) return;
        // Separate the array values into readable variables
        const req = newSource[0];
        const paused = newSource[1];

        // If the props have changed (either the name or the paused state or both), close the old resources
        // and empty the array
        if (stop) stop();
        stop = null; // free up memory if needed
        if (!paused && req) { // If not paused and there is a request, pull new stop
          stop = apiCall(req);
        }
      },
      {immediate: true, deep: true, flush: 'sync'}
  );
};
