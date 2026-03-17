import * as grpcWeb from 'grpc-web';

import * as traits_color_pb from '../traits/color_pb'; // proto import: "traits/color.proto"


export class ColorApiClient {
  constructor (hostname: string,
               credentials?: null | { [index: string]: string; },
               options?: null | { [index: string]: any; });

  getColor(
    request: traits_color_pb.GetColorRequest,
    metadata: grpcWeb.Metadata | undefined,
    callback: (err: grpcWeb.RpcError,
               response: traits_color_pb.Color) => void
  ): grpcWeb.ClientReadableStream<traits_color_pb.Color>;

  updateColor(
    request: traits_color_pb.UpdateColorRequest,
    metadata: grpcWeb.Metadata | undefined,
    callback: (err: grpcWeb.RpcError,
               response: traits_color_pb.Color) => void
  ): grpcWeb.ClientReadableStream<traits_color_pb.Color>;

  pullColor(
    request: traits_color_pb.PullColorRequest,
    metadata?: grpcWeb.Metadata
  ): grpcWeb.ClientReadableStream<traits_color_pb.PullColorResponse>;

}

export class ColorInfoClient {
  constructor (hostname: string,
               credentials?: null | { [index: string]: string; },
               options?: null | { [index: string]: any; });

  describeColor(
    request: traits_color_pb.DescribeColorRequest,
    metadata: grpcWeb.Metadata | undefined,
    callback: (err: grpcWeb.RpcError,
               response: traits_color_pb.ColorSupport) => void
  ): grpcWeb.ClientReadableStream<traits_color_pb.ColorSupport>;

}

export class ColorApiPromiseClient {
  constructor (hostname: string,
               credentials?: null | { [index: string]: string; },
               options?: null | { [index: string]: any; });

  getColor(
    request: traits_color_pb.GetColorRequest,
    metadata?: grpcWeb.Metadata
  ): Promise<traits_color_pb.Color>;

  updateColor(
    request: traits_color_pb.UpdateColorRequest,
    metadata?: grpcWeb.Metadata
  ): Promise<traits_color_pb.Color>;

  pullColor(
    request: traits_color_pb.PullColorRequest,
    metadata?: grpcWeb.Metadata
  ): grpcWeb.ClientReadableStream<traits_color_pb.PullColorResponse>;

}

export class ColorInfoPromiseClient {
  constructor (hostname: string,
               credentials?: null | { [index: string]: string; },
               options?: null | { [index: string]: any; });

  describeColor(
    request: traits_color_pb.DescribeColorRequest,
    metadata?: grpcWeb.Metadata
  ): Promise<traits_color_pb.ColorSupport>;

}

