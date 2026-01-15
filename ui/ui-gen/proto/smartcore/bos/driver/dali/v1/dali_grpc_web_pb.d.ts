import * as grpcWeb from 'grpc-web';

import * as smartcore_bos_driver_dali_v1_dali_pb from '../../../../../smartcore/bos/driver/dali/v1/dali_pb'; // proto import: "smartcore/bos/driver/dali/v1/dali.proto"


export class DaliApiClient {
  constructor (hostname: string,
               credentials?: null | { [index: string]: string; },
               options?: null | { [index: string]: any; });

  addToGroup(
    request: smartcore_bos_driver_dali_v1_dali_pb.AddToGroupRequest,
    metadata: grpcWeb.Metadata | undefined,
    callback: (err: grpcWeb.RpcError,
               response: smartcore_bos_driver_dali_v1_dali_pb.AddToGroupResponse) => void
  ): grpcWeb.ClientReadableStream<smartcore_bos_driver_dali_v1_dali_pb.AddToGroupResponse>;

  removeFromGroup(
    request: smartcore_bos_driver_dali_v1_dali_pb.RemoveFromGroupRequest,
    metadata: grpcWeb.Metadata | undefined,
    callback: (err: grpcWeb.RpcError,
               response: smartcore_bos_driver_dali_v1_dali_pb.RemoveFromGroupResponse) => void
  ): grpcWeb.ClientReadableStream<smartcore_bos_driver_dali_v1_dali_pb.RemoveFromGroupResponse>;

  getGroupMembership(
    request: smartcore_bos_driver_dali_v1_dali_pb.GetGroupMembershipRequest,
    metadata: grpcWeb.Metadata | undefined,
    callback: (err: grpcWeb.RpcError,
               response: smartcore_bos_driver_dali_v1_dali_pb.GetGroupMembershipResponse) => void
  ): grpcWeb.ClientReadableStream<smartcore_bos_driver_dali_v1_dali_pb.GetGroupMembershipResponse>;

  getControlGearStatus(
    request: smartcore_bos_driver_dali_v1_dali_pb.GetControlGearStatusRequest,
    metadata: grpcWeb.Metadata | undefined,
    callback: (err: grpcWeb.RpcError,
               response: smartcore_bos_driver_dali_v1_dali_pb.ControlGearStatus) => void
  ): grpcWeb.ClientReadableStream<smartcore_bos_driver_dali_v1_dali_pb.ControlGearStatus>;

  getEmergencyStatus(
    request: smartcore_bos_driver_dali_v1_dali_pb.GetEmergencyStatusRequest,
    metadata: grpcWeb.Metadata | undefined,
    callback: (err: grpcWeb.RpcError,
               response: smartcore_bos_driver_dali_v1_dali_pb.EmergencyStatus) => void
  ): grpcWeb.ClientReadableStream<smartcore_bos_driver_dali_v1_dali_pb.EmergencyStatus>;

  identify(
    request: smartcore_bos_driver_dali_v1_dali_pb.IdentifyRequest,
    metadata: grpcWeb.Metadata | undefined,
    callback: (err: grpcWeb.RpcError,
               response: smartcore_bos_driver_dali_v1_dali_pb.IdentifyResponse) => void
  ): grpcWeb.ClientReadableStream<smartcore_bos_driver_dali_v1_dali_pb.IdentifyResponse>;

  startTest(
    request: smartcore_bos_driver_dali_v1_dali_pb.StartTestRequest,
    metadata: grpcWeb.Metadata | undefined,
    callback: (err: grpcWeb.RpcError,
               response: smartcore_bos_driver_dali_v1_dali_pb.StartTestResponse) => void
  ): grpcWeb.ClientReadableStream<smartcore_bos_driver_dali_v1_dali_pb.StartTestResponse>;

  stopTest(
    request: smartcore_bos_driver_dali_v1_dali_pb.StopTestRequest,
    metadata: grpcWeb.Metadata | undefined,
    callback: (err: grpcWeb.RpcError,
               response: smartcore_bos_driver_dali_v1_dali_pb.StopTestResponse) => void
  ): grpcWeb.ClientReadableStream<smartcore_bos_driver_dali_v1_dali_pb.StopTestResponse>;

  getTestResult(
    request: smartcore_bos_driver_dali_v1_dali_pb.GetTestResultRequest,
    metadata: grpcWeb.Metadata | undefined,
    callback: (err: grpcWeb.RpcError,
               response: smartcore_bos_driver_dali_v1_dali_pb.TestResult) => void
  ): grpcWeb.ClientReadableStream<smartcore_bos_driver_dali_v1_dali_pb.TestResult>;

  deleteTestResult(
    request: smartcore_bos_driver_dali_v1_dali_pb.DeleteTestResultRequest,
    metadata: grpcWeb.Metadata | undefined,
    callback: (err: grpcWeb.RpcError,
               response: smartcore_bos_driver_dali_v1_dali_pb.TestResult) => void
  ): grpcWeb.ClientReadableStream<smartcore_bos_driver_dali_v1_dali_pb.TestResult>;

}

export class DaliApiPromiseClient {
  constructor (hostname: string,
               credentials?: null | { [index: string]: string; },
               options?: null | { [index: string]: any; });

  addToGroup(
    request: smartcore_bos_driver_dali_v1_dali_pb.AddToGroupRequest,
    metadata?: grpcWeb.Metadata
  ): Promise<smartcore_bos_driver_dali_v1_dali_pb.AddToGroupResponse>;

  removeFromGroup(
    request: smartcore_bos_driver_dali_v1_dali_pb.RemoveFromGroupRequest,
    metadata?: grpcWeb.Metadata
  ): Promise<smartcore_bos_driver_dali_v1_dali_pb.RemoveFromGroupResponse>;

  getGroupMembership(
    request: smartcore_bos_driver_dali_v1_dali_pb.GetGroupMembershipRequest,
    metadata?: grpcWeb.Metadata
  ): Promise<smartcore_bos_driver_dali_v1_dali_pb.GetGroupMembershipResponse>;

  getControlGearStatus(
    request: smartcore_bos_driver_dali_v1_dali_pb.GetControlGearStatusRequest,
    metadata?: grpcWeb.Metadata
  ): Promise<smartcore_bos_driver_dali_v1_dali_pb.ControlGearStatus>;

  getEmergencyStatus(
    request: smartcore_bos_driver_dali_v1_dali_pb.GetEmergencyStatusRequest,
    metadata?: grpcWeb.Metadata
  ): Promise<smartcore_bos_driver_dali_v1_dali_pb.EmergencyStatus>;

  identify(
    request: smartcore_bos_driver_dali_v1_dali_pb.IdentifyRequest,
    metadata?: grpcWeb.Metadata
  ): Promise<smartcore_bos_driver_dali_v1_dali_pb.IdentifyResponse>;

  startTest(
    request: smartcore_bos_driver_dali_v1_dali_pb.StartTestRequest,
    metadata?: grpcWeb.Metadata
  ): Promise<smartcore_bos_driver_dali_v1_dali_pb.StartTestResponse>;

  stopTest(
    request: smartcore_bos_driver_dali_v1_dali_pb.StopTestRequest,
    metadata?: grpcWeb.Metadata
  ): Promise<smartcore_bos_driver_dali_v1_dali_pb.StopTestResponse>;

  getTestResult(
    request: smartcore_bos_driver_dali_v1_dali_pb.GetTestResultRequest,
    metadata?: grpcWeb.Metadata
  ): Promise<smartcore_bos_driver_dali_v1_dali_pb.TestResult>;

  deleteTestResult(
    request: smartcore_bos_driver_dali_v1_dali_pb.DeleteTestResultRequest,
    metadata?: grpcWeb.Metadata
  ): Promise<smartcore_bos_driver_dali_v1_dali_pb.TestResult>;

}

