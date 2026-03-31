import {timestampFromObject} from '@/api/convpb';
import {Period} from '@smart-core-os/sc-bos-ui-gen/proto/smartcore/bos/types/time/v1/period_pb';

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
