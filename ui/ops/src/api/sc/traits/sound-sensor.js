import {fieldMaskFromObject, setProperties, timestampToDate} from '@/api/convpb.js';
import {clientOptions} from '@/api/grpcweb.js';
import {pullResource, setValue, trackAction} from '@/api/resource.js';
import {periodFromObject} from '@/api/sc/types/period';
import {SoundSensorHistoryPromiseClient} from '@smart-core-os/sc-bos-ui-gen/proto/sound_sensor_history_grpc_web_pb';
import {ListSoundLevelHistoryRequest} from '@smart-core-os/sc-bos-ui-gen/proto/sound_sensor_history_pb';
import {SoundSensorApiPromiseClient, SoundSensorInfoPromiseClient} from '@smart-core-os/sc-bos-ui-gen/proto/sound_sensor_grpc_web_pb';
import {DescribeSoundLevelRequest, PullSoundLevelRequest} from '@smart-core-os/sc-bos-ui-gen/proto/sound_sensor_pb';

/**
 * @param {Partial<PullSoundLevelRequest.AsObject>} request
 * @param {ResourceValue<SoundLevel.AsObject, PullSoundLevelResponse>} resource
 */
export function pullSoundLevel(request, resource) {
 pullResource('SoundSensorApi.pullSoundLevel', resource, (endpoint) => {
   const api = apiClient(endpoint);
   const stream = api.pullSoundLevel(pullSoundLevelRequestFromObject(request));
   stream.on('data', (msg) => {
    const changes = msg.getChangesList();
    for (const change of changes) {
     setValue(resource, change.getSoundLevel().toObject());
    }
   });
   return stream;
  });
}

/**
 *
 * @param {Partial<DescribeSoundLevelRequest.AsObject>} request
 * @param {ActionTracker<SoundLevelSupport.AsObject>} [tracker]
 * @return {Promise<SoundLevelSupport.AsObject>}
 */
export function describeSoundLevel(request, tracker) {
  return trackAction('SoundLevelInfo.DescribeSoundLevel', tracker ?? {}, (endpoint) => {
   const api = infoClient(endpoint);
   return api.describeSoundLevel(describeSoundLevelRequestFromObject(request));
  });
}

/**
 *
 * @param {Partial<ListSoundLevelHistoryRequest.AsObject>} request
 * @param {ActionTracker<ListSoundLevelHistoryResponse.AsObject>} tracker
 * @return {Promise<ListSoundLevelHistoryResponse.AsObject>}
 */
export function listSoundLevelHistory(request, tracker) {
  return trackAction('SoundSensorHistory.listSoundLevelHistory', tracker ?? {}, (endpoint) => {
    const api = historyClient(endpoint);
    return api.listSoundLevelHistory(listSoundLevelHistoryRequestFromObject(request));
  });
}

/**
 * @param {SoundLevelRecord | SoundLevelRecord.AsObject} obj
 * @return {SoundLevelRecord.AsObject & {recordTime: Date|undefined}}
 */
export function soundLevelRecordToObject(obj) {
  if (!obj) return undefined;
  if (typeof obj.toObject === 'function') obj = obj.toObject();
  if (obj.recordTime) obj.recordTime = timestampToDate(obj.recordTime);
  return obj;
}

/**
 * @param {string} endpoint
 * @return {SoundSensorApiPromiseClient}
 */
function apiClient(endpoint) {
  return new SoundSensorApiPromiseClient(endpoint, null, clientOptions());
}

/**
 * @param {string} endpoint
 * @return {SoundSensorInfoPromiseClient}
 */
function infoClient(endpoint) {
  return new SoundSensorInfoPromiseClient(endpoint, null, clientOptions());
}

/**
 *
 * @param {string} endpoint
 * @return {SoundSensorHistoryPromiseClient}
 */
function historyClient(endpoint) {
  return new SoundSensorHistoryPromiseClient(endpoint, null, clientOptions());
}

/**
 * @param {Partial<PullSoundLevelRequest.AsObject>} obj
 * @return {PullSoundLevelRequest|undefined}
 */
function pullSoundLevelRequestFromObject(obj) {
  if (!obj) return undefined;
  const dst = new PullSoundLevelRequest();
  setProperties(dst, obj, 'name', 'updatesOnly');
  dst.setReadMask(fieldMaskFromObject(obj.readMask));
  return dst;
}

/**
 * @param {Partial<DescribeSoundLevelRequest.AsObject>} obj
 * @return {undefined|DescribeSoundLevelRequest}
 */
function describeSoundLevelRequestFromObject(obj) {
  if (!obj) return undefined;
  const dst = new DescribeSoundLevelRequest();
  setProperties(dst, obj, 'name');
  return dst;
}

/**
 * @param {Partial<ListSoundLevelHistoryRequest.AsObject>} obj
 * @return {ListSoundLevelHistoryRequest|undefined}
 */
function listSoundLevelHistoryRequestFromObject(obj) {
  if (!obj) return undefined;
  const dst = new ListSoundLevelHistoryRequest();
  setProperties(dst, obj, 'name', 'pageToken', 'pageSize', 'orderBy');
  dst.setReadMask(fieldMaskFromObject(obj.readMask));
  dst.setPeriod(periodFromObject(obj.period));
  return dst;
}
