import deepEqual from 'fast-deep-equal';
import {toValue, watch} from 'vue';

/**
 * Converts a query like value into a Smart Core query object.
 *
 * @template {{name: string}} T
 * @param {MaybeRefOrGetter<string|T|null>} input
 * @return {T|null}
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
 * @param {MaybeRefOrGetter<T>} query
 * @param {MaybeRefOrGetter<boolean>} [paused]
 * @param {(req: T) => () => {}} apiCall
 */
export const watchResource = (query, paused = false, apiCall) => {
  let stop = null;

  watch(
      [() => toValue(query), () => toValue(paused)],
      (newSource, oldSource) => {
        const oldNewEqual = deepEqual(newSource, oldSource);

        if (oldNewEqual) return;
        const req = newSource[0];
        const paused = newSource[1];

        if (stop) stop();
        stop = null;
        if (!paused && req) {
          stop = apiCall(req);
        }
      },
      {immediate: true, deep: true, flush: 'sync'}
  );
};
