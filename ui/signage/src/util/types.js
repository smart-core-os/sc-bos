// isObject: Checks if the given value is a non-null object and not any other type like Array.
export const isObject = (value) => {
  return value !== null && typeof value === 'object' && value.constructor === Object;
};

// isArray: Determines if the value is a non-null array.
export const isArray = (value) => {
  return value !== null && Array.isArray(value);
};

// isString: Checks if the value is a string. This includes both string literals and String objects.
export const isString = (value) => {
  return value !== null && (typeof value === 'string' || value instanceof String);
};

// isNumber: Verifies that the value is a number and not NaN or Infinity.
export const isNumber = (value) => {
  return value !== null && typeof value === 'number' && isFinite(value);
};

// isFunction: Checks if the value is a function, including both named and anonymous functions.
export const isFunction = (value) => {
  return value !== null && typeof value === 'function';
};

// isBoolean: Determines if the value is a boolean (true or false).
export const isBoolean = (value) => {
  return value !== null && typeof value === 'boolean';
};

// isNull: Specifically checks if the value is null.
export const isNull = (value) => {
  return value === null;
};

// isUndefined: Determines if the value is undefined, indicating an uninitialized variable.
export const isUndefined = (value) => {
  return typeof value === 'undefined';
};

/**
 * Returns true if the value is null or undefined.
 *
 * @param {any} value
 * @return {boolean}
 */
export const isNullOrUndef = (value) => {
  return isNull(value) || isUndefined(value);
};

// isDate: Validates whether the value is a Date object and not an invalid date (NaN).
export const isDate = (value) => {
  return value instanceof Date && !isNaN(value);
};

// isRegExp: Checks if the value is a regular expression object.
export const isRegExp = (value) => {
  return value !== null && value instanceof RegExp;
};

// isSymbol: Verifies if the value is a Symbol, a unique and immutable primitive introduced in ES6.
export const isSymbol = (value) => {
  return typeof value === 'symbol';
};

// isBigInt: Checks if the value is a BigInt, a type of data for representing integers larger than 2^53.
export const isBigInt = (value) => {
  return typeof value === 'bigint';
};

// isValueAvailable: Determines if the value is 'available' by checking
// that it's neither null, an empty array, nor an empty object.
export const isValueAvailable = (value) => {
  return value && !(Array.isArray(value) && value.length === 0) &&
      !(value.constructor === Object && Object.keys(value).length === 0);
};

