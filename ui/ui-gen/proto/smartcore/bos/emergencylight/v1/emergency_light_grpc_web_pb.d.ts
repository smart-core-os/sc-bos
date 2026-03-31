import * as grpcWeb from 'grpc-web';

import * as smartcore_bos_emergencylight_v1_emergency_light_pb from '../../../../smartcore/bos/emergencylight/v1/emergency_light_pb'; // proto import: "smartcore/bos/emergencylight/v1/emergency_light.proto"


export class EmergencyLightApiClient {
  constructor (hostname: string,
               credentials?: null | { [index: string]: string; },
               options?: null | { [index: string]: any; });

  startFunctionTest(
    request: smartcore_bos_emergencylight_v1_emergency_light_pb.StartEmergencyTestRequest,
    metadata: grpcWeb.Metadata | undefined,
    callback: (err: grpcWeb.RpcError,
               response: smartcore_bos_emergencylight_v1_emergency_light_pb.StartEmergencyTestResponse) => void
  ): grpcWeb.ClientReadableStream<smartcore_bos_emergencylight_v1_emergency_light_pb.StartEmergencyTestResponse>;

  startDurationTest(
    request: smartcore_bos_emergencylight_v1_emergency_light_pb.StartEmergencyTestRequest,
    metadata: grpcWeb.Metadata | undefined,
    callback: (err: grpcWeb.RpcError,
               response: smartcore_bos_emergencylight_v1_emergency_light_pb.StartEmergencyTestResponse) => void
  ): grpcWeb.ClientReadableStream<smartcore_bos_emergencylight_v1_emergency_light_pb.StartEmergencyTestResponse>;

  stopEmergencyTest(
    request: smartcore_bos_emergencylight_v1_emergency_light_pb.StopEmergencyTestsRequest,
    metadata: grpcWeb.Metadata | undefined,
    callback: (err: grpcWeb.RpcError,
               response: smartcore_bos_emergencylight_v1_emergency_light_pb.StopEmergencyTestsResponse) => void
  ): grpcWeb.ClientReadableStream<smartcore_bos_emergencylight_v1_emergency_light_pb.StopEmergencyTestsResponse>;

  getTestResultSet(
    request: smartcore_bos_emergencylight_v1_emergency_light_pb.GetTestResultSetRequest,
    metadata: grpcWeb.Metadata | undefined,
    callback: (err: grpcWeb.RpcError,
               response: smartcore_bos_emergencylight_v1_emergency_light_pb.TestResultSet) => void
  ): grpcWeb.ClientReadableStream<smartcore_bos_emergencylight_v1_emergency_light_pb.TestResultSet>;

  pullTestResultSets(
    request: smartcore_bos_emergencylight_v1_emergency_light_pb.PullTestResultRequest,
    metadata?: grpcWeb.Metadata
  ): grpcWeb.ClientReadableStream<smartcore_bos_emergencylight_v1_emergency_light_pb.PullTestResultsResponse>;

}

export class EmergencyLightApiPromiseClient {
  constructor (hostname: string,
               credentials?: null | { [index: string]: string; },
               options?: null | { [index: string]: any; });

  startFunctionTest(
    request: smartcore_bos_emergencylight_v1_emergency_light_pb.StartEmergencyTestRequest,
    metadata?: grpcWeb.Metadata
  ): Promise<smartcore_bos_emergencylight_v1_emergency_light_pb.StartEmergencyTestResponse>;

  startDurationTest(
    request: smartcore_bos_emergencylight_v1_emergency_light_pb.StartEmergencyTestRequest,
    metadata?: grpcWeb.Metadata
  ): Promise<smartcore_bos_emergencylight_v1_emergency_light_pb.StartEmergencyTestResponse>;

  stopEmergencyTest(
    request: smartcore_bos_emergencylight_v1_emergency_light_pb.StopEmergencyTestsRequest,
    metadata?: grpcWeb.Metadata
  ): Promise<smartcore_bos_emergencylight_v1_emergency_light_pb.StopEmergencyTestsResponse>;

  getTestResultSet(
    request: smartcore_bos_emergencylight_v1_emergency_light_pb.GetTestResultSetRequest,
    metadata?: grpcWeb.Metadata
  ): Promise<smartcore_bos_emergencylight_v1_emergency_light_pb.TestResultSet>;

  pullTestResultSets(
    request: smartcore_bos_emergencylight_v1_emergency_light_pb.PullTestResultRequest,
    metadata?: grpcWeb.Metadata
  ): grpcWeb.ClientReadableStream<smartcore_bos_emergencylight_v1_emergency_light_pb.PullTestResultsResponse>;

}

