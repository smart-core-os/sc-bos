import {fieldMaskFromObject, setProperties} from '@/api/convpb';
import {clientOptions} from '@/api/grpcweb.js';
import {pullResource, setValue, trackAction} from '@/api/resource.js';
import {OpenCloseApiPromiseClient, OpenCloseInfoPromiseClient} from '@smart-core-os/sc-bos-ui-gen/proto/smartcore/bos/openclose/v1/open_close_grpc_web_pb';
import {
  DescribePositionsRequest,
  OpenClosePosition,
  OpenClosePositions,
  PullOpenClosePositionsRequest,
  UpdateOpenClosePositionsRequest
} from '@smart-core-os/sc-bos-ui-gen/proto/smartcore/bos/openclose/v1/open_close_pb';

/**
 * @param {Partial<PullOpenClosePositionsRequest.AsObject>} request
 * @param {ResourceValue<OpenClosePositions.AsObject, PullOpenClosePositionsResponse>} resource
 */
export function pullOpenClosePositions(request, resource) {
  pullResource('OpenClose.pullOpenClosePositions', resource, endpoint => {
    const api = apiClient(endpoint);
    const stream = api.pullPositions(pullOpenClosePositionsRequestFromObject(request));
    stream.on('data', msg => {
      const changes = msg.getChangesList();
      for (const change of changes) {
        setValue(resource, change.getOpenClosePosition().toObject());
      }
    });
    return stream;
  });
}

/**
 * @param {Partial<UpdateOpenClosePositionsRequest.AsObject>} request
 * @param {ActionTracker<OpenClosePositions.AsObject>} [tracker]
 * @return {Promise<OpenClosePositions.AsObject>}
 */
export function updateOpenClosePositions(request, tracker) {
  return trackAction('OpenClose.updatePositions', tracker ?? {}, endpoint => {
    const api = apiClient(endpoint);
    return api.updatePositions(updateOpenClosePositionsRequestFromObject(request));
  });
}

/**
 * @param {Partial<DescribePositionsRequest.AsObject>} request
 * @param {ActionTracker<PositionsSupport.AsObject>} [tracker]
 * @return {Promise<PositionsSupport.AsObject>}
 */
export function describeOpenClosePositions(request, tracker) {
  return trackAction('OpenCloseInfo.describePositions', tracker ?? {}, endpoint => {
    const api = infoClient(endpoint);
    return api.describePositions(describePositionsRequestFromObject(request));
  });
}

/**
 * @param {string} endpoint
 * @return {OpenCloseApiPromiseClient}
 */
function apiClient(endpoint) {
  return new OpenCloseApiPromiseClient(endpoint, null, clientOptions());
}

/**
 * @param {string} endpoint
 * @return {OpenCloseInfoPromiseClient}
 */
function infoClient(endpoint) {
  return new OpenCloseInfoPromiseClient(endpoint, null, clientOptions());
}

/**
 * @param {Partial<PullOpenClosePositionsRequest.AsObject>} obj
 * @return {undefined|PullOpenClosePositionsRequest}
 */
function pullOpenClosePositionsRequestFromObject(obj) {
  if (!obj) return undefined;

  const dst = new PullOpenClosePositionsRequest();
  setProperties(dst, obj, 'name', 'excludeTweening', 'updatesOnly');
  dst.setReadMask(fieldMaskFromObject(obj.readMask));
  return dst;
}

/**
 * @param {Partial<UpdateOpenClosePositionsRequest.AsObject>} obj
 * @return {undefined|UpdateOpenClosePositionsRequest}
 */
function updateOpenClosePositionsRequestFromObject(obj) {
  if (!obj) return undefined;

  const dst = new UpdateOpenClosePositionsRequest();
  setProperties(dst, obj, 'name', 'delta');
  dst.setStates(openClosePositionsFromObject(obj.states));
  dst.setUpdateMask(fieldMaskFromObject(obj.updateMask));
  return dst;
}

/**
 * @param {Partial<OpenClosePositions.AsObject>} obj
 * @return {undefined|OpenClosePositions}
 */
function openClosePositionsFromObject(obj) {
  if (!obj) return undefined;

  const dst = new OpenClosePositions();
  if (obj.statesList) {
    dst.setStatesList(obj.statesList.map(openClosePositionFromObject).filter(Boolean));
  }
  if (obj.preset) {
    const preset = new OpenClosePositions.Preset();
    setProperties(preset, obj.preset, 'name', 'title');
    dst.setPreset(preset);
  }
  return dst;
}

/**
 * @param {Partial<OpenClosePosition.AsObject>} obj
 * @return {undefined|OpenClosePosition}
 */
function openClosePositionFromObject(obj) {
  if (!obj) return undefined;

  const dst = new OpenClosePosition();
  setProperties(dst, obj, 'openPercent', 'targetOpenPercent', 'direction', 'resistance');
  return dst;
}

/**
 * @param {Partial<DescribePositionsRequest.AsObject>} obj
 * @return {undefined|DescribePositionsRequest}
 */
function describePositionsRequestFromObject(obj) {
  if (!obj) return undefined;

  const dst = new DescribePositionsRequest();
  setProperties(dst, obj, 'name');
  return dst;
}