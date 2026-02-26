import {timestampFromObject} from '@/api/convpb';
import {Period} from '@smart-core-os/sc-api-grpc-web/types/time/period_pb';

/**
 * @param {Partial<Period.AsObject>} obj
 * @return {Period|undefined}
 */
export function periodFromObject(obj) {
  if (!obj) return undefined;
  const dst = new Period();
  dst.setEndTime(timestampFromObject(obj.endTime));
  dst.setStartTime(timestampFromObject(obj.startTime));
  return dst;
}
