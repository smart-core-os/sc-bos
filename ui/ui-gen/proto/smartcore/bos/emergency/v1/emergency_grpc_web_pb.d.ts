import * as grpcWeb from 'grpc-web';

import * as smartcore_bos_emergency_v1_emergency_pb from '../../../../smartcore/bos/emergency/v1/emergency_pb'; // proto import: "smartcore/bos/emergency/v1/emergency.proto"


export class EmergencyApiClient {
  constructor (hostname: string,
               credentials?: null | { [index: string]: string; },
               options?: null | { [index: string]: any; });

  getEmergency(
    request: smartcore_bos_emergency_v1_emergency_pb.GetEmergencyRequest,
    metadata: grpcWeb.Metadata | undefined,
    callback: (err: grpcWeb.RpcError,
               response: smartcore_bos_emergency_v1_emergency_pb.Emergency) => void
  ): grpcWeb.ClientReadableStream<smartcore_bos_emergency_v1_emergency_pb.Emergency>;

  updateEmergency(
    request: smartcore_bos_emergency_v1_emergency_pb.UpdateEmergencyRequest,
    metadata: grpcWeb.Metadata | undefined,
    callback: (err: grpcWeb.RpcError,
               response: smartcore_bos_emergency_v1_emergency_pb.Emergency) => void
  ): grpcWeb.ClientReadableStream<smartcore_bos_emergency_v1_emergency_pb.Emergency>;

  pullEmergency(
    request: smartcore_bos_emergency_v1_emergency_pb.PullEmergencyRequest,
    metadata?: grpcWeb.Metadata
  ): grpcWeb.ClientReadableStream<smartcore_bos_emergency_v1_emergency_pb.PullEmergencyResponse>;

}

export class EmergencyInfoClient {
  constructor (hostname: string,
               credentials?: null | { [index: string]: string; },
               options?: null | { [index: string]: any; });

  describeEmergency(
    request: smartcore_bos_emergency_v1_emergency_pb.DescribeEmergencyRequest,
    metadata: grpcWeb.Metadata | undefined,
    callback: (err: grpcWeb.RpcError,
               response: smartcore_bos_emergency_v1_emergency_pb.EmergencySupport) => void
  ): grpcWeb.ClientReadableStream<smartcore_bos_emergency_v1_emergency_pb.EmergencySupport>;

}

export class EmergencyApiPromiseClient {
  constructor (hostname: string,
               credentials?: null | { [index: string]: string; },
               options?: null | { [index: string]: any; });

  getEmergency(
    request: smartcore_bos_emergency_v1_emergency_pb.GetEmergencyRequest,
    metadata?: grpcWeb.Metadata
  ): Promise<smartcore_bos_emergency_v1_emergency_pb.Emergency>;

  updateEmergency(
    request: smartcore_bos_emergency_v1_emergency_pb.UpdateEmergencyRequest,
    metadata?: grpcWeb.Metadata
  ): Promise<smartcore_bos_emergency_v1_emergency_pb.Emergency>;

  pullEmergency(
    request: smartcore_bos_emergency_v1_emergency_pb.PullEmergencyRequest,
    metadata?: grpcWeb.Metadata
  ): grpcWeb.ClientReadableStream<smartcore_bos_emergency_v1_emergency_pb.PullEmergencyResponse>;

}

export class EmergencyInfoPromiseClient {
  constructor (hostname: string,
               credentials?: null | { [index: string]: string; },
               options?: null | { [index: string]: any; });

  describeEmergency(
    request: smartcore_bos_emergency_v1_emergency_pb.DescribeEmergencyRequest,
    metadata?: grpcWeb.Metadata
  ): Promise<smartcore_bos_emergency_v1_emergency_pb.EmergencySupport>;

}

