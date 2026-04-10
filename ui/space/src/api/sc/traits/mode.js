import {fieldMaskFromObject, setProperties} from '@/api/convpb';
import {clientOptions} from '@/api/grpcweb.js';
import {trackAction} from '@/api/resource';
import {pullResource, setValue} from '@/api/resource.js';
import {ModeApiPromiseClient, ModeInfoPromiseClient} from '@smart-core-os/sc-bos-ui-gen/proto/smartcore/bos/mode/v1/mode_grpc_web_pb';
import {
  DescribeModesRequest,
  ModeValues,
  ModeValuesRelative,
  PullModeValuesRequest,
  UpdateModeValuesRequest
} from '@smart-core-os/sc-bos-ui-gen/proto/smartcore/bos/mode/v1/mode_pb';

/**
 * @param {Partial<PullModeValuesRequest.AsObject>} request
 * @param {ResourceValue<ModeValues.AsObject, PullModeValuesResponse>} resource
 */
export function pullModeValues(request, resource) {
  pullResource('Mode.pullModeValues', resource, endpoint => {
    const api = apiClient(endpoint);
    const stream = api.pullModeValues(pullModeValuesRequestFromObject(request));
    stream.on('data', msg => {
      const changes = msg.getChangesList();
      for (const change of changes) {
        setValue(resource, change.getModeValues().toObject());
      }
    });
    return stream;
  });
}

/**
 *
 * @param {Partial<UpdateModeValuesRequest.AsObject>} request
 * @param {ActionTracker<ModeValues.AsObject>} [tracker]
 * @return {Promise<ModeValues.AsObject>}
 */
export function updateModeValues(request, tracker) {
  return trackAction('ModeValues.updateModeValues', tracker ?? {}, endpoint => {
    const api = apiClient(endpoint);
    return api.updateModeValues(updateModeValuesRequestFromObject(request));
  });
}

/**
 *
 * @param {Partial<DescribeModesRequest.AsObject>} request
 * @param {ActionTracker<ModesSupport.AsObject>} [tracker]
 * @return {Promise<ModesSupport.AsObject>}
 */
export function describeModes(request, tracker) {
  return trackAction('ModeSettings.describeModes', tracker ?? {}, endpoint => {
    const api = infoClient(endpoint);
    return api.describeModes(describeModesRequestFromObject(request));
  });
}

/**
 * @param {string} endpoint
 * @return {ModeApiPromiseClient}
 */
function apiClient(endpoint) {
  return new ModeApiPromiseClient(endpoint, null, clientOptions());
}

/**
 * @param {string} endpoint
 * @return {ModeInfoPromiseClient}
 */
function infoClient(endpoint) {
  return new ModeInfoPromiseClient(endpoint, null, clientOptions());
}

/**
 * @param {Partial<PullModeValuesRequest.AsObject>} obj
 * @return {PullModeValuesRequest|undefined}
 */
function pullModeValuesRequestFromObject(obj) {
  if (!obj) return undefined;

  const req = new PullModeValuesRequest();
  setProperties(req, obj, 'name', 'updatesOnly');
  req.setReadMask(fieldMaskFromObject(obj.readMask));
  return req;
}

/**
 * @param {Partial<UpdateModeValuesRequest.AsObject>} obj
 * @return {UpdateModeValuesRequest}
 */
function updateModeValuesRequestFromObject(obj) {
  if (!obj) return undefined;

  const req = new UpdateModeValuesRequest();
  setProperties(req, obj, 'name');
  req.setModeValues(modeValuesFromObject(obj.modeValues));
  req.setRelative(modeValuesRelativeFromObject(obj.relative));
  req.setUpdateMask(fieldMaskFromObject(obj.updateMask));
  return req;
}

/**
 * @param {Partial<DescribeModesRequest.AsObject>} obj
 * @return {DescribeModesRequest|undefined}
 */
function describeModesRequestFromObject(obj) {
  if (!obj) return undefined;

  const req = new DescribeModesRequest();
  setProperties(req, obj, 'name');
  return req;
}

/**
 * @param {Partial<ModeValues.AsObject>} obj
 * @return {undefined|ModeValues}
 */
function modeValuesFromObject(obj) {
  if (!obj) return undefined;

  const state = new ModeValues();
  if (Array.isArray(obj.valuesMap)) {
    for (const [k, v] of obj.valuesMap) {
      state.getValuesMap().set(k, v);
    }
  } else if (typeof obj.valuesMap === 'object') {
    for (const [k, v] of Object.entries(obj.valuesMap)) {
      state.getValuesMap().set(k, v);
    }
  }
  return state;
}

/**
 * @param {Partial<ModeValuesRelative.AsObject>} obj
 * @return {undefined|ModeValuesRelative}
 */
function modeValuesRelativeFromObject(obj) {
  if (!obj) return undefined;

  const state = new ModeValuesRelative();
  if (Array.isArray(obj.valuesMap)) {
    for (const [k, v] of obj.valuesMap) {
      state.getValuesMap().set(k, v);
    }
  } else if (typeof obj.valuesMap === 'object') {
    for (const [k, v] of Object.entries(obj.valuesMap)) {
      state.getValuesMap().set(k, v);
    }
  }
  return state;
}
