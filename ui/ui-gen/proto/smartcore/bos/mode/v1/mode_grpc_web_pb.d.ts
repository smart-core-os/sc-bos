import * as grpcWeb from 'grpc-web';

import * as smartcore_bos_mode_v1_mode_pb from '../../../../smartcore/bos/mode/v1/mode_pb'; // proto import: "smartcore/bos/mode/v1/mode.proto"


export class ModeApiClient {
  constructor (hostname: string,
               credentials?: null | { [index: string]: string; },
               options?: null | { [index: string]: any; });

  getModeValues(
    request: smartcore_bos_mode_v1_mode_pb.GetModeValuesRequest,
    metadata: grpcWeb.Metadata | undefined,
    callback: (err: grpcWeb.RpcError,
               response: smartcore_bos_mode_v1_mode_pb.ModeValues) => void
  ): grpcWeb.ClientReadableStream<smartcore_bos_mode_v1_mode_pb.ModeValues>;

  updateModeValues(
    request: smartcore_bos_mode_v1_mode_pb.UpdateModeValuesRequest,
    metadata: grpcWeb.Metadata | undefined,
    callback: (err: grpcWeb.RpcError,
               response: smartcore_bos_mode_v1_mode_pb.ModeValues) => void
  ): grpcWeb.ClientReadableStream<smartcore_bos_mode_v1_mode_pb.ModeValues>;

  pullModeValues(
    request: smartcore_bos_mode_v1_mode_pb.PullModeValuesRequest,
    metadata?: grpcWeb.Metadata
  ): grpcWeb.ClientReadableStream<smartcore_bos_mode_v1_mode_pb.PullModeValuesResponse>;

}

export class ModeInfoClient {
  constructor (hostname: string,
               credentials?: null | { [index: string]: string; },
               options?: null | { [index: string]: any; });

  describeModes(
    request: smartcore_bos_mode_v1_mode_pb.DescribeModesRequest,
    metadata: grpcWeb.Metadata | undefined,
    callback: (err: grpcWeb.RpcError,
               response: smartcore_bos_mode_v1_mode_pb.ModesSupport) => void
  ): grpcWeb.ClientReadableStream<smartcore_bos_mode_v1_mode_pb.ModesSupport>;

}

export class ModeApiPromiseClient {
  constructor (hostname: string,
               credentials?: null | { [index: string]: string; },
               options?: null | { [index: string]: any; });

  getModeValues(
    request: smartcore_bos_mode_v1_mode_pb.GetModeValuesRequest,
    metadata?: grpcWeb.Metadata
  ): Promise<smartcore_bos_mode_v1_mode_pb.ModeValues>;

  updateModeValues(
    request: smartcore_bos_mode_v1_mode_pb.UpdateModeValuesRequest,
    metadata?: grpcWeb.Metadata
  ): Promise<smartcore_bos_mode_v1_mode_pb.ModeValues>;

  pullModeValues(
    request: smartcore_bos_mode_v1_mode_pb.PullModeValuesRequest,
    metadata?: grpcWeb.Metadata
  ): grpcWeb.ClientReadableStream<smartcore_bos_mode_v1_mode_pb.PullModeValuesResponse>;

}

export class ModeInfoPromiseClient {
  constructor (hostname: string,
               credentials?: null | { [index: string]: string; },
               options?: null | { [index: string]: any; });

  describeModes(
    request: smartcore_bos_mode_v1_mode_pb.DescribeModesRequest,
    metadata?: grpcWeb.Metadata
  ): Promise<smartcore_bos_mode_v1_mode_pb.ModesSupport>;

}

