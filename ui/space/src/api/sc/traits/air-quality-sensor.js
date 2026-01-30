import {fieldMaskFromObject, setProperties} from '@/api/convpb';
import {clientOptions} from '@/api/grpcweb.js';
import {pullResource, setValue, trackAction} from '@/api/resource.js';
import {periodFromObject} from '@/api/sc/types/period';
import {AirQualitySensorHistoryPromiseClient} from '@smart-core-os/sc-bos-ui-gen/proto/smartcore/bos/airqualitysensor/v1/air_quality_sensor_history_grpc_web_pb';
import {ListAirQualityHistoryRequest} from '@smart-core-os/sc-bos-ui-gen/proto/smartcore/bos/airqualitysensor/v1/air_quality_sensor_history_pb';
import {AirQualitySensorApiPromiseClient} from '@smart-core-os/sc-api-grpc-web/traits/air_quality_sensor_grpc_web_pb';
import {PullAirQualityRequest} from '@smart-core-os/sc-api-grpc-web/traits/air_quality_sensor_pb';

/**
 * @param {Partial<PullAirQualityRequest.AsObject>} request
 * @param {ResourceValue<AirQuality.AsObject, PullAirQualityResponse>} resource
 */
export function pullAirQualitySensor(request, resource) {
  pullResource('AirQualitySensor.pullAirQuality', resource, endpoint => {
    const api = apiClient(endpoint);
    const stream = api.pullAirQuality(pullAirQualityRequestFromObject(request));
    stream.on('data', msg => {
      const changes = msg.getChangesList();
      for (const change of changes) {
        setValue(resource, change.getAirQuality().toObject());
      }
    });
    return stream;
  });
}

/**
 *
 * @param {Partial<ListAirQualityHistoryRequest.AsObject>} request
 * @param {ActionTracker<ListAirQualityHistoryResponse.AsObject>} tracker
 * @return {Promise<ListAirQualityHistoryResponse.AsObject>}
 */
export function listAirQualitySensorHistory(request, tracker) {
  return trackAction('AirQualitySensorHistory.listAirQualitySensorHistory', tracker, (endpoint) => {
    const api = historyClient(endpoint);
    return api.listAirQualityHistory(listAirQualitySensorHistoryRequestFromObject(request));
  });
}

/**
 * @param {string} endpoint
 * @return {AirQualitySensorApiPromiseClient}
 */
function apiClient(endpoint) {
  return new AirQualitySensorApiPromiseClient(endpoint, null, clientOptions());
}

/**
 *
 * @param {string} endpoint
 * @return {AirQualitySensorHistoryPromiseClient}
 */
function historyClient(endpoint) {
  return new AirQualitySensorHistoryPromiseClient(endpoint, null, clientOptions());
}

/**
 * @param {Partial<PullAirQualityRequest.AsObject>} obj
 * @return {PullAirQualityRequest|undefined}
 */
function pullAirQualityRequestFromObject(obj) {
  if (!obj) return undefined;
  const dst = new PullAirQualityRequest();
  setProperties(dst, obj, 'name', 'updatesOnly');
  dst.setReadMask(fieldMaskFromObject(obj.readMask));
  return dst;
}

/**
 * @param {Partial<ListAirQualityHistoryRequest.AsObject>} obj
 * @return {ListAirQualityHistoryRequest|undefined}
 */
function listAirQualitySensorHistoryRequestFromObject(obj) {
  if (!obj) return undefined;
  const dst = new ListAirQualityHistoryRequest();
  setProperties(dst, obj, 'name', 'pageToken', 'pageSize');
  dst.setReadMask(fieldMaskFromObject(obj.readMask));
  dst.setPeriod(periodFromObject(obj.period));
  return dst;
}
