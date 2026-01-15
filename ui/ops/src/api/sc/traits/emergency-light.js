import {setProperties} from '@/api/convpb.js';
import {clientOptions} from '@/api/grpcweb.js';
import {trackAction} from '@/api/resource.js';
import {EmergencyLightApiPromiseClient} from '@smart-core-os/sc-bos-ui-gen/proto/smartcore/bos/emergencylight/v1/emergency_light_grpc_web_pb';
import {GetTestResultSetRequest, StartEmergencyTestRequest} from '@smart-core-os/sc-bos-ui-gen/proto/smartcore/bos/emergencylight/v1/emergency_light_pb';

/**
 * @param {string} endpoint
 * @return {EmergencyLightApiPromiseClient}
 */
function apiClient(endpoint) {
 return new EmergencyLightApiPromiseClient(endpoint, null, clientOptions());
}

/**
 * @param {Partial<GetTestResultSetRequest.AsObject>} request
 * @param {ActionTracker<GetTestResultSetRequest.AsObject>} [tracker]
 * @return {Promise<TestResultSet.AsObject>}
 */
export function getTestResultSet(request, tracker) {
 return trackAction('EmergencyLight.TestResultSet', tracker ?? {}, endpoint => {
  const api = apiClient(endpoint);
  return api.getTestResultSet(getLatestTestResultsRequestFromObject(request));
 });
}

/**
 * @param {Partial<GetTestResultSetRequest.AsObject>} obj
 * @return {GetTestResultSetRequest|undefined}
 */
function getLatestTestResultsRequestFromObject(obj) {
 if (!obj) return undefined;
 const dst = new GetTestResultSetRequest();
 setProperties(dst, obj, 'name', 'queryDevice');
 return dst;
}

/**
 * @param {Partial<StartEmergencyTestRequest.AsObject>} request
 * @param {ActionTracker<StartTestRequest.AsObject>} [tracker]
 * @return {Promise<StartEmergencyTestResponse.AsObject>}
 */
export function startFunctionTest(request, tracker) {
 return trackAction('EmergencyLight.StartFunctionTest', tracker ?? {}, endpoint => {
  const api = apiClient(endpoint);
  return api.startFunctionTest(startEmergencyTestRequestFromObject(request));
 });
}

/**
 * @param {Partial<StartEmergencyTestRequest.AsObject>} request
 * @param {ActionTracker<StartTestRequest.AsObject>} [tracker]
 * @return {Promise<StartEmergencyTestResponse.AsObject>}
 */
export function startDurationTest(request, tracker) {
 return trackAction('EmergencyLight.StartDurationTest', tracker ?? {}, endpoint => {
  const api = apiClient(endpoint);
  return api.startDurationTest(startEmergencyTestRequestFromObject(request));
 });
}

/**
 * @param {Partial<StartEmergencyTestRequest.AsObject>} obj
 * @return {StartEmergencyTestRequest|undefined}
 */
function startEmergencyTestRequestFromObject(obj) {
 if (!obj) return undefined;
 const dst = new StartEmergencyTestRequest();
 setProperties(dst, obj, 'name');
 return dst;
}

/**
 * @param {Partial<EmergencyTestResult.Result.AsObject>} result
 * @return {string|undefined}
 */
export function emergencyLightResultToString(result) {
 switch (result) {
  case 0:
   return 'Unspecified';
  case 1:
   return 'Test Result Pending';
  case 2:
   return 'Test Passed';
  case 3:
   return 'Circuit Failure';
  case 4:
   return 'Battery Duration Failure';
  case 5:
   return 'Battery Failure';
  case 6:
   return 'Lamp Failure';
  case 7:
   return 'Test Failed';
  case 8:
   return 'Light Faulty';
  case 9:
   return 'Communication Failure';
  case 10:
   return 'Other Fault';
  case -1:
   return 'Failed to get result';
  default:
   return 'Unspecified';
 }
}