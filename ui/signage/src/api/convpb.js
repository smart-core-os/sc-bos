import {parseISO} from 'date-fns';
import {Duration} from 'google-protobuf/google/protobuf/duration_pb';
import {FieldMask} from 'google-protobuf/google/protobuf/field_mask_pb';
import {Timestamp} from 'google-protobuf/google/protobuf/timestamp_pb';

/**
 * Copy props from src into dst. Src is a JS object and dst is a protobuf message.
 * This function calls setSomeProp for each prop, which should be camelCase versions of the proto property names.
 *
 * @param {import('google-protobuf').Message} dst
 * @param {Object} src
 * @param {string} props
 */
export function setProperties(dst, src, ...props) {
  convertProperties(dst, src, v => v, ...props);
}

/**
 * @function ConvFunc
 * @param {*} v
 * @return *
 */

/**
 * Copy props from src into dst calling conv on each. Src is a JS object and dst is a protobuf message.
 * This function calls setSomeProp for each prop, which should be camelCase versions of the proto property names.
 *
 * @param {import('google-protobuf').Message} dst
 * @param {Object} src
 * @param {ConvFunc} conv
 * @param {string} props
 */
export function convertProperties(dst, src, conv, ...props) {
  for (const prop of props) {
    if (Object.hasOwn(src, prop)) {
      dst['set' + prop[0].toUpperCase() + prop.substring(1)](conv(src[prop]));
    }
  }
}

/**
 * Convert a js object representing a Timestamp into a protobuf Timestamp.
 *
 * @param {Timestamp.AsObject|string|Date} obj
 * @return {Timestamp}
 */
export function timestampFromObject(obj) {
  if (!obj) return undefined;
  if (typeof obj === 'string') return timestampFromObject(parseISO(obj));
  if (obj instanceof Date) return Timestamp.fromDate(obj);

  return new Timestamp()
      .setSeconds(obj.seconds)
      .setNanos(obj.nanos);
}

/**
 * @param {google_protobuf_timestamp_pb.Timestamp|google_protobuf_timestamp_pb.Timestamp.AsObject} ts
 * @return {Date}
 */
export function timestampToDate(ts) {
  if (!ts) return undefined;
  if (ts instanceof Timestamp) return ts.toDate();
  if (Object.hasOwn(ts, 'nanos') && Object.hasOwn(ts, 'seconds')) {
    return timestampToDate(new Timestamp().setSeconds(ts.seconds).setNanos(ts.seconds));
  }

  // be kind
  if (ts instanceof Date) return ts;
  throw new Error('cannot convert ' + ts + ' to Date, unknown format');
}

/**
 * Covert a JS object representing a Duration into a protobuf Duration.
 *
 * @param {Duration.AsObject} obj
 * @return {Duration}
 */
export function durationFromObject(obj) {
  if (!obj) return undefined;

  const dst = new Duration();
  setProperties(dst, obj, 'seconds', 'nanos');
  return dst;
}

/**
 * Covert a JS object representing a FieldMask into a protobuf FieldMask.
 *
 * @param {FieldMask.AsObject} obj
 * @return {FieldMask|undefined}
 */
export function fieldMaskFromObject(obj) {
  if (!obj) return undefined;

  const dst = new FieldMask();
  setProperties(dst, obj, 'pathsList');
  return dst;
}
