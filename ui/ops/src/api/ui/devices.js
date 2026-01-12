import {convertProperties, fieldMaskFromObject, setProperties, timestampFromObject} from '@/api/convpb.js';
import {clientOptions} from '@/api/grpcweb.js';
import {pullResource, setCollection, setValue} from '@/api/resource';
import {trackAction} from '@/api/resource.js';
import {periodFromObject} from '@/api/sc/types/period.js';
import {DevicesApiPromiseClient} from '@smart-core-os/sc-bos-ui-gen/proto/smartcore/bos/devices/v1/devices_grpc_web_pb';
import {
  Device,
  DevicesMetadata,
  GetDevicesMetadataRequest,
  GetDownloadDevicesUrlRequest,
  ListDevicesRequest,
  PullDevicesMetadataRequest,
  PullDevicesRequest
} from '@smart-core-os/sc-bos-ui-gen/proto/smartcore/bos/devices/v1/devices_pb';
import {Empty} from 'google-protobuf/google/protobuf/empty_pb.js';

/**
 * @param {Partial<ListDevicesRequest.AsObject>} request
 * @param {ActionTracker<ListDevicesResponse.AsObject>} [tracker]
 * @return {Promise<ListDevicesResponse.AsObject>}
 */
export function listDevices(request, tracker) {
  return trackAction('Devices.listDevices', tracker ?? {}, endpoint => {
    const api = apiClient(endpoint);
    return api.listDevices(listDevicesRequestFromObject(request));
  });
}

/**
 * @param {Partial<PullDevicesRequest.AsObject>} request
 * @param {ResourceCollection<Device.AsObject, PullDevicesResponse>} resource
 */
export function pullDevices(request, resource) {
  pullResource('Devices.pullDevices', resource, endpoint => {
    const api = apiClient(endpoint);
    const stream = api.pullDevices(pullDevicesRequestFromObject(request));
    stream.on('data', msg => {
      const changes = msg.getChangesList();
      for (const change of changes) {
        setCollection(resource, change, (d) => d.name);
      }
    });
    return stream;
  });
}

/**
 *
 * @param {Partial<GetDevicesMetadataRequest.AsObject>} request
 * @param {ActionTracker<GetDevicesMetadataRequest.AsObject>} [tracker]
 * @return {Promise<DevicesMetadata.AsObject>}
 */
export function getDevicesMetadata(request, tracker) {
  return trackAction('Devices.getDevicesMetadata', tracker ?? {}, endpoint => {
    const api = apiClient(endpoint);
    return api.getDevicesMetadata(getDevicesMetadataRequestFromObject(request));
  });
}

/**
 * @param {Partial<PullDevicesMetadataRequest.AsObject>} request
 * @param {ResourceValue<DevicesMetadata.AsObject, DevicesMetadata>} resource
 */
export function pullDevicesMetadata(request, resource) {
  pullResource('Devices.pullDevicesMetadata', resource, endpoint => {
    const api = apiClient(endpoint);
    const stream = api.pullDevicesMetadata(pullDevicesMetadataRequestFromObject(request));
    stream.on('data', msg => {
      const changes = msg.getChangesList();
      for (const change of changes) {
        setValue(resource, change.getDevicesMetadata().toObject());
      }
    });
    return stream;
  });
}

/**
 * @param {Partial<GetDownloadDevicesUrlRequest.AsObject>} request
 * @param {ActionTracker<DownloadDevicesUrl.AsObject>} [tracker]
 * @return {Promise<DownloadDevicesUrl.AsObject>}
 */
export function getDownloadDevicesUrl(request, tracker) {
  return trackAction('Devices.getDownloadDevicesUrl', tracker ?? {}, endpoint => {
    const api = apiClient(endpoint);
    return api.getDownloadDevicesUrl(getDownloadDevicesUrlRequestFromObject(request));
  });
}

/**
 * @param {string} endpoint
 * @return {DevicesApiPromiseClient}
 */
function apiClient(endpoint) {
  return new DevicesApiPromiseClient(endpoint, null, clientOptions());
}

/**
 * @param {Partial<ListDevicesRequest.AsObject>} obj
 * @return {undefined|ListDevicesRequest}
 */
function listDevicesRequestFromObject(obj) {
  if (!obj) return undefined;
  const dst = new ListDevicesRequest();
  setProperties(dst, obj, 'pageToken', 'pageSize');
  dst.setQuery(deviceQueryFromObject(obj.query));
  return dst;
}

/**
 * @param {Partial<PullDevicesRequest.AsObject>} obj
 * @return {undefined|PullDevicesRequest}
 */
function pullDevicesRequestFromObject(obj) {
  if (!obj) return undefined;
  const dst = new PullDevicesRequest();
  setProperties(dst, obj, 'updatesOnly');
  dst.setReadMask(fieldMaskFromObject(obj.readMask));
  dst.setQuery(deviceQueryFromObject(obj.query));
  return dst;
}

/**
 * @param {Partial<Device.Query.AsObject>} obj
 * @return {undefined|Device.Query}
 */
function deviceQueryFromObject(obj) {
  if (!obj) return undefined;
  const dst = new Device.Query();
  for (const item of (obj.conditionsList ?? [])) {
    dst.addConditions(deviceQueryConditionFromObject(item));
  }
  return dst;
}

/**
 * @param {Partial<Device.Query.Condition.AsObject>} obj
 * @return {undefined|Device.Query.Condition}
 */
function deviceQueryConditionFromObject(obj) {
  if (!obj) return undefined;
  const dst = new Device.Query.Condition();
  setProperties(dst, obj, 'field', 'matcher',
      'stringEqual', 'stringEqualFold',
      'stringContains', 'stringContainsFold',
      'nameDescendant', 'nameDescendantInc'
  );
  if (obj.stringIn) {
    dst.setStringIn(new Device.Query.StringList().setStringsList(obj.stringIn.stringsList));
  }
  if (obj.stringInFold) {
    dst.setStringInFold(new Device.Query.StringList().setStringsList(obj.stringInFold.stringsList));
  }
  if (obj.nameDescendantIn) {
    dst.setNameDescendantIn(new Device.Query.StringList().setStringsList(obj.nameDescendantIn.stringsList));
  }
  if (obj.nameDescendantIncIn) {
    dst.setNameDescendantIncIn(new Device.Query.StringList().setStringsList(obj.nameDescendantIncIn.stringsList));
  }
  convertProperties(dst, obj, timestampFromObject,
      'timestampEqual', 'timestampGt', 'timestampGte', 'timestampLt', 'timestampLte');

  if (obj.present) {
    dst.setPresent(new Empty())
  }
  dst.setMatches(deviceQueryFromObject(obj.matches));
  return dst;
}
/**
 *
 * @param {Partial<DevicesMetadata.Include.AsObject>} obj
 * @return {undefined|DevicesMetadata.Include}
 */
function devicesMetadataIncludeFromObject(obj) {
  if (!obj) return undefined;
  const dst = new DevicesMetadata.Include();
  dst.setFieldsList(obj.fieldsList);
  return dst;
}

/**
 *
 * @param {Partial<GetDevicesMetadataRequest.AsObject>} obj
 * @return {undefined|GetDevicesMetadataRequest}
 */
function getDevicesMetadataRequestFromObject(obj) {
  if (!obj) return undefined;
  const dst = new GetDevicesMetadataRequest();
  dst.setReadMask(fieldMaskFromObject(obj.readMask));
  dst.setIncludes(devicesMetadataIncludeFromObject(obj.includes));
  dst.setQuery(deviceQueryFromObject(obj.query));
  return dst;
}

/**
 *
 * @param {Partial<PullDevicesMetadataRequest.AsObject>} obj
 * @return {undefined|PullDevicesMetadataRequest}
 */
function pullDevicesMetadataRequestFromObject(obj) {
  if (!obj) return undefined;
  const dst = new PullDevicesMetadataRequest();
  setProperties(dst, obj, 'updatesOnly');
  dst.setReadMask(fieldMaskFromObject(obj.readMask));
  dst.setIncludes(devicesMetadataIncludeFromObject(obj.includes));
  dst.setQuery(deviceQueryFromObject(obj.query));
  return dst;
}

/**
 * @param {Partial<GetDownloadDevicesUrlRequest.AsObject>} obj
 * @return {GetDownloadDevicesUrlRequest|undefined}
 */
function getDownloadDevicesUrlRequestFromObject(obj) {
  if (!obj) return undefined;
  const dst = new GetDownloadDevicesUrlRequest();
  setProperties(dst, obj, 'mediaType', 'filename');
  dst.setQuery(deviceQueryFromObject(obj.query));
  dst.setHistory(periodFromObject(obj.history));
  dst.setTable(getDownloadDevicesUrlRequestTableFromObject(obj.table));
  return dst;
}

/**
 * @param {Partial<GetDownloadDevicesUrlRequest.Table.AsObject>} obj
 * @return {GetDownloadDevicesUrlRequest.Table|undefined}
 */
function getDownloadDevicesUrlRequestTableFromObject(obj) {
  if (!obj) return undefined;
  const dst = new GetDownloadDevicesUrlRequest.Table();
  if (obj.excludeColsList) {
    for (const col of obj.excludeColsList) {
      dst.addExcludeCols(getDownloadDevicesUrlRequestTableColumnFromObject(col));
    }
  }
  if (obj.includeColsList) {
    for (const col of obj.includeColsList) {
      dst.addIncludeCols(getDownloadDevicesUrlRequestTableColumnFromObject(col));
    }
  }
  return dst;
}

/**
 * @param {Partial<GetDownloadDevicesUrlRequest.Table.Column.AsObject>} obj
 * @return {GetDownloadDevicesUrlRequest.Table.Column|undefined}
 */
function getDownloadDevicesUrlRequestTableColumnFromObject(obj) {
  if (!obj) return undefined;
  const dst = new GetDownloadDevicesUrlRequest.Table.Column();
  setProperties(dst, obj, 'name', 'title');
  return dst;
}
