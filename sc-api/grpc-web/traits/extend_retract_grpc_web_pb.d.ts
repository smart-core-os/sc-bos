import * as grpcWeb from 'grpc-web';

import * as traits_extend_retract_pb from '../traits/extend_retract_pb'; // proto import: "traits/extend_retract.proto"


export class ExtendRetractApiClient {
  constructor (hostname: string,
               credentials?: null | { [index: string]: string; },
               options?: null | { [index: string]: any; });

  getExtension(
    request: traits_extend_retract_pb.GetExtensionRequest,
    metadata: grpcWeb.Metadata | undefined,
    callback: (err: grpcWeb.RpcError,
               response: traits_extend_retract_pb.Extension) => void
  ): grpcWeb.ClientReadableStream<traits_extend_retract_pb.Extension>;

  updateExtension(
    request: traits_extend_retract_pb.UpdateExtensionRequest,
    metadata: grpcWeb.Metadata | undefined,
    callback: (err: grpcWeb.RpcError,
               response: traits_extend_retract_pb.Extension) => void
  ): grpcWeb.ClientReadableStream<traits_extend_retract_pb.Extension>;

  stop(
    request: traits_extend_retract_pb.ExtendRetractStopRequest,
    metadata: grpcWeb.Metadata | undefined,
    callback: (err: grpcWeb.RpcError,
               response: traits_extend_retract_pb.Extension) => void
  ): grpcWeb.ClientReadableStream<traits_extend_retract_pb.Extension>;

  createExtensionPreset(
    request: traits_extend_retract_pb.CreateExtensionPresetRequest,
    metadata: grpcWeb.Metadata | undefined,
    callback: (err: grpcWeb.RpcError,
               response: traits_extend_retract_pb.ExtensionPreset) => void
  ): grpcWeb.ClientReadableStream<traits_extend_retract_pb.ExtensionPreset>;

  pullExtensions(
    request: traits_extend_retract_pb.PullExtensionsRequest,
    metadata?: grpcWeb.Metadata
  ): grpcWeb.ClientReadableStream<traits_extend_retract_pb.PullExtensionsResponse>;

}

export class ExtendRetractInfoClient {
  constructor (hostname: string,
               credentials?: null | { [index: string]: string; },
               options?: null | { [index: string]: any; });

  describeExtension(
    request: traits_extend_retract_pb.DescribeExtensionRequest,
    metadata: grpcWeb.Metadata | undefined,
    callback: (err: grpcWeb.RpcError,
               response: traits_extend_retract_pb.ExtensionSupport) => void
  ): grpcWeb.ClientReadableStream<traits_extend_retract_pb.ExtensionSupport>;

}

export class ExtendRetractApiPromiseClient {
  constructor (hostname: string,
               credentials?: null | { [index: string]: string; },
               options?: null | { [index: string]: any; });

  getExtension(
    request: traits_extend_retract_pb.GetExtensionRequest,
    metadata?: grpcWeb.Metadata
  ): Promise<traits_extend_retract_pb.Extension>;

  updateExtension(
    request: traits_extend_retract_pb.UpdateExtensionRequest,
    metadata?: grpcWeb.Metadata
  ): Promise<traits_extend_retract_pb.Extension>;

  stop(
    request: traits_extend_retract_pb.ExtendRetractStopRequest,
    metadata?: grpcWeb.Metadata
  ): Promise<traits_extend_retract_pb.Extension>;

  createExtensionPreset(
    request: traits_extend_retract_pb.CreateExtensionPresetRequest,
    metadata?: grpcWeb.Metadata
  ): Promise<traits_extend_retract_pb.ExtensionPreset>;

  pullExtensions(
    request: traits_extend_retract_pb.PullExtensionsRequest,
    metadata?: grpcWeb.Metadata
  ): grpcWeb.ClientReadableStream<traits_extend_retract_pb.PullExtensionsResponse>;

}

export class ExtendRetractInfoPromiseClient {
  constructor (hostname: string,
               credentials?: null | { [index: string]: string; },
               options?: null | { [index: string]: any; });

  describeExtension(
    request: traits_extend_retract_pb.DescribeExtensionRequest,
    metadata?: grpcWeb.Metadata
  ): Promise<traits_extend_retract_pb.ExtensionSupport>;

}

