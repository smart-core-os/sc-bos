import {setProperties} from '@/api/convpb';
import {clientOptions} from '@/api/grpcweb';
import {trackAction} from '@/api/resource';
import {DaliApiPromiseClient} from '@smart-core-os/sc-bos-ui-gen/proto/smartcore/bos/driver/dali/v1/dali_grpc_web_pb';
import {StartTestRequest} from '@smart-core-os/sc-bos-ui-gen/proto/smartcore/bos/driver/dali/v1/dali_pb';
import {LightingTestApiPromiseClient} from '@smart-core-os/sc-bos-ui-gen/proto/smartcore/bos/lightingtest/v1/lighting_test_grpc_web_pb';
import {GetReportCSVRequest, ListLightHealthRequest} from '@smart-core-os/sc-bos-ui-gen/proto/smartcore/bos/lightingtest/v1/lighting_test_pb';

/**
 *
 * @param {ActionTracker<ReportCSV.AsObject>} [tracker]
 * @return {Promise<ReportCSV.AsObject>}
 */
export function getReportCSV(tracker) {
  return trackAction('LightingTest.getReportCSV', tracker ?? {}, endpoint => {
    const api = client(endpoint);
    const req = new GetReportCSVRequest();
    req.setIncludeHeader(true);
    return api.getReportCSV(req);
  });
}

/**
 *
 * @param {Partial<ListLightHealthRequest.AsObject>} request
 * @param {ActionTracker<ListLightHealthResponse.AsObject>} [tracker]
 * @return {Promise<ListLightHealthResponse.AsObject>}
 */
export function listLightHealth(request, tracker) {
  return trackAction('LightingTest.listLightHealth', tracker ?? {}, endpoint => {
    const api = client(endpoint);
    return api.listLightHealth(listLightHealthRequestFromObject(request));
  });
}

/**
 *
 * @param {Partial<StartTestRequest.AsObject>} request
 * @param {ActionTracker<StartTestRequest.AsObject>} [tracker]
 * @return {Promise<StartTestResponse.AsObject>}
 */
export function runTest(request, tracker) {
  return trackAction('Dali.StartTest', tracker ?? {}, endpoint => {
    const api = new DaliApiPromiseClient(endpoint, null, clientOptions());
    return api.startTest(startTestRequestFromObject(request));
  });
}

/**
 *
 * @param {number} faultId
 * @return {string}
 */
export function faultToString(faultId) {
  switch (faultId) {
    case 1:
      return 'DURATION_TEST_FAILED';
    case 2:
      return 'FUNCTION_TEST_FAILED';
    case 3:
      return 'BATTERY_FAULT';
    case 4:
      return 'LAMP_FAULT';
    case 5:
      return 'COMMUNICATION_FAILURE';
    case 6:
      return 'OTHER_FAULT';
    default:
      return 'FAULT_UNSPECIFIED';
  }
}

/**
 *
 * @param {string} endpoint
 * @return {LightingTestApiPromiseClient}
 */
function client(endpoint) {
  return new LightingTestApiPromiseClient(endpoint, null, clientOptions());
}

/**
 * @param {Partial<StartTestRequest.AsObject>} obj
 * @return {StartTestRequest}
 */
function startTestRequestFromObject(obj) {
  if (!obj) return undefined;

  const req = new StartTestRequest();
  setProperties(req, obj, 'name', 'test');
  return req;
}

/**
 * @param {Partial<ListLightHealthRequest.AsObject>} obj
 * @return {undefined|ListLightHealthRequest}
 */
function listLightHealthRequestFromObject(obj) {
  if (!obj) return undefined;

  const dst = new ListLightHealthRequest();
  setProperties(dst, obj, 'pageSize', 'pageToken');
  return dst;
}
