import * as grpcWeb from 'grpc-web';

import * as traits_emergency_pb from '../traits/emergency_pb'; // proto import: "traits/emergency.proto"


export class EmergencyApiClient {
  constructor (hostname: string,
               credentials?: null | { [index: string]: string; },
               options?: null | { [index: string]: any; });

  getEmergency(
    request: traits_emergency_pb.GetEmergencyRequest,
    metadata: grpcWeb.Metadata | undefined,
    callback: (err: grpcWeb.RpcError,
               response: traits_emergency_pb.Emergency) => void
  ): grpcWeb.ClientReadableStream<traits_emergency_pb.Emergency>;

  updateEmergency(
    request: traits_emergency_pb.UpdateEmergencyRequest,
    metadata: grpcWeb.Metadata | undefined,
    callback: (err: grpcWeb.RpcError,
               response: traits_emergency_pb.Emergency) => void
  ): grpcWeb.ClientReadableStream<traits_emergency_pb.Emergency>;

  pullEmergency(
    request: traits_emergency_pb.PullEmergencyRequest,
    metadata?: grpcWeb.Metadata
  ): grpcWeb.ClientReadableStream<traits_emergency_pb.PullEmergencyResponse>;

}

export class EmergencyInfoClient {
  constructor (hostname: string,
               credentials?: null | { [index: string]: string; },
               options?: null | { [index: string]: any; });

  describeEmergency(
    request: traits_emergency_pb.DescribeEmergencyRequest,
    metadata: grpcWeb.Metadata | undefined,
    callback: (err: grpcWeb.RpcError,
               response: traits_emergency_pb.EmergencySupport) => void
  ): grpcWeb.ClientReadableStream<traits_emergency_pb.EmergencySupport>;

}

export class EmergencyApiPromiseClient {
  constructor (hostname: string,
               credentials?: null | { [index: string]: string; },
               options?: null | { [index: string]: any; });

  getEmergency(
    request: traits_emergency_pb.GetEmergencyRequest,
    metadata?: grpcWeb.Metadata
  ): Promise<traits_emergency_pb.Emergency>;

  updateEmergency(
    request: traits_emergency_pb.UpdateEmergencyRequest,
    metadata?: grpcWeb.Metadata
  ): Promise<traits_emergency_pb.Emergency>;

  pullEmergency(
    request: traits_emergency_pb.PullEmergencyRequest,
    metadata?: grpcWeb.Metadata
  ): grpcWeb.ClientReadableStream<traits_emergency_pb.PullEmergencyResponse>;

}

export class EmergencyInfoPromiseClient {
  constructor (hostname: string,
               credentials?: null | { [index: string]: string; },
               options?: null | { [index: string]: any; });

  describeEmergency(
    request: traits_emergency_pb.DescribeEmergencyRequest,
    metadata?: grpcWeb.Metadata
  ): Promise<traits_emergency_pb.EmergencySupport>;

}

