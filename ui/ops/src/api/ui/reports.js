import {fieldMaskFromObject, setProperties} from '@/api/convpb.js';
import {clientOptions} from '@/api/grpcweb.js';
import {trackAction} from '@/api/resource.js';
import {ReportApiPromiseClient} from '@smart-core-os/sc-bos-ui-gen/proto/smartcore/bos/report/v1/report_grpc_web_pb';
import {GetDownloadReportUrlRequest, ListReportsRequest} from '@smart-core-os/sc-bos-ui-gen/proto/smartcore/bos/report/v1/report_pb';

/**
 * @param {Partial<GetDownloadReportUrlRequest.AsObject>} request
 * @param {ActionTracker<DownloadReportUrl.AsObject>} [tracker]
 * @return {Promise<DownloadReportUrl.AsObject>}
 */
export function getDownloadReportUrl(request, tracker) {
  return trackAction('Reports.getDownloadReportUrl', tracker ?? {}, endpoint => {
    const api = apiClient(endpoint);
    return api.getDownloadReportUrl(getDownloadReportUrlRequestFromObject(request));
  });
}

/**
 * @param {string} endpoint
 * @return {ReportApiPromiseClient}
 */
function apiClient(endpoint) {
  return new ReportApiPromiseClient(endpoint, null, clientOptions());
}

/**
 * @param {Partial<GetDownloadReportUrlRequest.AsObject>} obj
 * @return {GetDownloadReportUrlRequest|undefined}
 */
function getDownloadReportUrlRequestFromObject(obj) {
  if (!obj) return undefined;
  const dst = new GetDownloadReportUrlRequest();
  setProperties(dst, obj, 'name', 'id');
  return dst;
}

/**
 *
 * @param {Partial<ListReportsRequest.AsObject>} request
 * @param {ActionTracker<ListReportsResponse.AsObject>} tracker
 * @return {Promise<ListReportsResponse.AsObject>}
 */
export function listReports(request, tracker) {
  return trackAction('Reports.listReports', tracker ?? {}, endpoint => {
    const api = apiClient(endpoint);
    return api.listReports(listReportsRequestFromObject(request));
  });
}

/**
 * @param {Partial<ListReportsRequest.AsObject>} obj
 * @return {ListReportsRequest|undefined}
 */
function listReportsRequestFromObject(obj) {
  if (!obj) return undefined;
  const dst = new ListReportsRequest();
  setProperties(dst, obj, 'name', 'pageToken', 'pageSize');
  dst.setReadMask(fieldMaskFromObject(obj.readMask));
  return dst;
}