import * as grpcWeb from 'grpc-web';

import * as traits_mode_pb from '../traits/mode_pb'; // proto import: "traits/mode.proto"


export class ModeApiClient {
  constructor (hostname: string,
               credentials?: null | { [index: string]: string; },
               options?: null | { [index: string]: any; });

  getModeValues(
    request: traits_mode_pb.GetModeValuesRequest,
    metadata: grpcWeb.Metadata | undefined,
    callback: (err: grpcWeb.RpcError,
               response: traits_mode_pb.ModeValues) => void
  ): grpcWeb.ClientReadableStream<traits_mode_pb.ModeValues>;

  updateModeValues(
    request: traits_mode_pb.UpdateModeValuesRequest,
    metadata: grpcWeb.Metadata | undefined,
    callback: (err: grpcWeb.RpcError,
               response: traits_mode_pb.ModeValues) => void
  ): grpcWeb.ClientReadableStream<traits_mode_pb.ModeValues>;

  pullModeValues(
    request: traits_mode_pb.PullModeValuesRequest,
    metadata?: grpcWeb.Metadata
  ): grpcWeb.ClientReadableStream<traits_mode_pb.PullModeValuesResponse>;

}

export class ModeInfoClient {
  constructor (hostname: string,
               credentials?: null | { [index: string]: string; },
               options?: null | { [index: string]: any; });

  describeModes(
    request: traits_mode_pb.DescribeModesRequest,
    metadata: grpcWeb.Metadata | undefined,
    callback: (err: grpcWeb.RpcError,
               response: traits_mode_pb.ModesSupport) => void
  ): grpcWeb.ClientReadableStream<traits_mode_pb.ModesSupport>;

}

export class ModeApiPromiseClient {
  constructor (hostname: string,
               credentials?: null | { [index: string]: string; },
               options?: null | { [index: string]: any; });

  getModeValues(
    request: traits_mode_pb.GetModeValuesRequest,
    metadata?: grpcWeb.Metadata
  ): Promise<traits_mode_pb.ModeValues>;

  updateModeValues(
    request: traits_mode_pb.UpdateModeValuesRequest,
    metadata?: grpcWeb.Metadata
  ): Promise<traits_mode_pb.ModeValues>;

  pullModeValues(
    request: traits_mode_pb.PullModeValuesRequest,
    metadata?: grpcWeb.Metadata
  ): grpcWeb.ClientReadableStream<traits_mode_pb.PullModeValuesResponse>;

}

export class ModeInfoPromiseClient {
  constructor (hostname: string,
               credentials?: null | { [index: string]: string; },
               options?: null | { [index: string]: any; });

  describeModes(
    request: traits_mode_pb.DescribeModesRequest,
    metadata?: grpcWeb.Metadata
  ): Promise<traits_mode_pb.ModesSupport>;

}

