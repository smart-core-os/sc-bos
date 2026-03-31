import * as grpcWeb from 'grpc-web';

import * as smartcore_bos_extendretract_v1_extend_retract_pb from '../../../../smartcore/bos/extendretract/v1/extend_retract_pb'; // proto import: "smartcore/bos/extendretract/v1/extend_retract.proto"


export class ExtendRetractApiClient {
  constructor (hostname: string,
               credentials?: null | { [index: string]: string; },
               options?: null | { [index: string]: any; });

  getExtension(
    request: smartcore_bos_extendretract_v1_extend_retract_pb.GetExtensionRequest,
    metadata: grpcWeb.Metadata | undefined,
    callback: (err: grpcWeb.RpcError,
               response: smartcore_bos_extendretract_v1_extend_retract_pb.Extension) => void
  ): grpcWeb.ClientReadableStream<smartcore_bos_extendretract_v1_extend_retract_pb.Extension>;

  updateExtension(
    request: smartcore_bos_extendretract_v1_extend_retract_pb.UpdateExtensionRequest,
    metadata: grpcWeb.Metadata | undefined,
    callback: (err: grpcWeb.RpcError,
               response: smartcore_bos_extendretract_v1_extend_retract_pb.Extension) => void
  ): grpcWeb.ClientReadableStream<smartcore_bos_extendretract_v1_extend_retract_pb.Extension>;

  stop(
    request: smartcore_bos_extendretract_v1_extend_retract_pb.ExtendRetractStopRequest,
    metadata: grpcWeb.Metadata | undefined,
    callback: (err: grpcWeb.RpcError,
               response: smartcore_bos_extendretract_v1_extend_retract_pb.Extension) => void
  ): grpcWeb.ClientReadableStream<smartcore_bos_extendretract_v1_extend_retract_pb.Extension>;

  createExtensionPreset(
    request: smartcore_bos_extendretract_v1_extend_retract_pb.CreateExtensionPresetRequest,
    metadata: grpcWeb.Metadata | undefined,
    callback: (err: grpcWeb.RpcError,
               response: smartcore_bos_extendretract_v1_extend_retract_pb.ExtensionPreset) => void
  ): grpcWeb.ClientReadableStream<smartcore_bos_extendretract_v1_extend_retract_pb.ExtensionPreset>;

  pullExtensions(
    request: smartcore_bos_extendretract_v1_extend_retract_pb.PullExtensionsRequest,
    metadata?: grpcWeb.Metadata
  ): grpcWeb.ClientReadableStream<smartcore_bos_extendretract_v1_extend_retract_pb.PullExtensionsResponse>;

}

export class ExtendRetractInfoClient {
  constructor (hostname: string,
               credentials?: null | { [index: string]: string; },
               options?: null | { [index: string]: any; });

  describeExtension(
    request: smartcore_bos_extendretract_v1_extend_retract_pb.DescribeExtensionRequest,
    metadata: grpcWeb.Metadata | undefined,
    callback: (err: grpcWeb.RpcError,
               response: smartcore_bos_extendretract_v1_extend_retract_pb.ExtensionSupport) => void
  ): grpcWeb.ClientReadableStream<smartcore_bos_extendretract_v1_extend_retract_pb.ExtensionSupport>;

}

export class ExtendRetractApiPromiseClient {
  constructor (hostname: string,
               credentials?: null | { [index: string]: string; },
               options?: null | { [index: string]: any; });

  getExtension(
    request: smartcore_bos_extendretract_v1_extend_retract_pb.GetExtensionRequest,
    metadata?: grpcWeb.Metadata
  ): Promise<smartcore_bos_extendretract_v1_extend_retract_pb.Extension>;

  updateExtension(
    request: smartcore_bos_extendretract_v1_extend_retract_pb.UpdateExtensionRequest,
    metadata?: grpcWeb.Metadata
  ): Promise<smartcore_bos_extendretract_v1_extend_retract_pb.Extension>;

  stop(
    request: smartcore_bos_extendretract_v1_extend_retract_pb.ExtendRetractStopRequest,
    metadata?: grpcWeb.Metadata
  ): Promise<smartcore_bos_extendretract_v1_extend_retract_pb.Extension>;

  createExtensionPreset(
    request: smartcore_bos_extendretract_v1_extend_retract_pb.CreateExtensionPresetRequest,
    metadata?: grpcWeb.Metadata
  ): Promise<smartcore_bos_extendretract_v1_extend_retract_pb.ExtensionPreset>;

  pullExtensions(
    request: smartcore_bos_extendretract_v1_extend_retract_pb.PullExtensionsRequest,
    metadata?: grpcWeb.Metadata
  ): grpcWeb.ClientReadableStream<smartcore_bos_extendretract_v1_extend_retract_pb.PullExtensionsResponse>;

}

export class ExtendRetractInfoPromiseClient {
  constructor (hostname: string,
               credentials?: null | { [index: string]: string; },
               options?: null | { [index: string]: any; });

  describeExtension(
    request: smartcore_bos_extendretract_v1_extend_retract_pb.DescribeExtensionRequest,
    metadata?: grpcWeb.Metadata
  ): Promise<smartcore_bos_extendretract_v1_extend_retract_pb.ExtensionSupport>;

}

