import {durationFromObject, setProperties} from '@/api/convpb.js';
import {Tween} from '@smart-core-os/sc-api-grpc-web/types/tween_pb';

/**
 * Convert a JS object representing a Tween into a protobuf Tween.
 *
 * @param {Partial<Tween.AsObject>} obj
 * @return {Tween}
 */
export function tweenFromObject(obj) {
  if (!obj) return undefined;

  const dst = new Tween();
  setProperties(dst, obj, 'progress');
  dst.setTotalDuration(durationFromObject(obj.totalDuration));
  return dst;
}
