import * as grpcWeb from 'grpc-web';

import * as smartcore_bos_color_v1_color_pb from '../../../../smartcore/bos/color/v1/color_pb'; // proto import: "smartcore/bos/color/v1/color.proto"


export class ColorApiClient {
  constructor (hostname: string,
               credentials?: null | { [index: string]: string; },
               options?: null | { [index: string]: any; });

  getColor(
    request: smartcore_bos_color_v1_color_pb.GetColorRequest,
    metadata: grpcWeb.Metadata | undefined,
    callback: (err: grpcWeb.RpcError,
               response: smartcore_bos_color_v1_color_pb.Color) => void
  ): grpcWeb.ClientReadableStream<smartcore_bos_color_v1_color_pb.Color>;

  updateColor(
    request: smartcore_bos_color_v1_color_pb.UpdateColorRequest,
    metadata: grpcWeb.Metadata | undefined,
    callback: (err: grpcWeb.RpcError,
               response: smartcore_bos_color_v1_color_pb.Color) => void
  ): grpcWeb.ClientReadableStream<smartcore_bos_color_v1_color_pb.Color>;

  pullColor(
    request: smartcore_bos_color_v1_color_pb.PullColorRequest,
    metadata?: grpcWeb.Metadata
  ): grpcWeb.ClientReadableStream<smartcore_bos_color_v1_color_pb.PullColorResponse>;

}

export class ColorInfoClient {
  constructor (hostname: string,
               credentials?: null | { [index: string]: string; },
               options?: null | { [index: string]: any; });

  describeColor(
    request: smartcore_bos_color_v1_color_pb.DescribeColorRequest,
    metadata: grpcWeb.Metadata | undefined,
    callback: (err: grpcWeb.RpcError,
               response: smartcore_bos_color_v1_color_pb.ColorSupport) => void
  ): grpcWeb.ClientReadableStream<smartcore_bos_color_v1_color_pb.ColorSupport>;

}

export class ColorApiPromiseClient {
  constructor (hostname: string,
               credentials?: null | { [index: string]: string; },
               options?: null | { [index: string]: any; });

  getColor(
    request: smartcore_bos_color_v1_color_pb.GetColorRequest,
    metadata?: grpcWeb.Metadata
  ): Promise<smartcore_bos_color_v1_color_pb.Color>;

  updateColor(
    request: smartcore_bos_color_v1_color_pb.UpdateColorRequest,
    metadata?: grpcWeb.Metadata
  ): Promise<smartcore_bos_color_v1_color_pb.Color>;

  pullColor(
    request: smartcore_bos_color_v1_color_pb.PullColorRequest,
    metadata?: grpcWeb.Metadata
  ): grpcWeb.ClientReadableStream<smartcore_bos_color_v1_color_pb.PullColorResponse>;

}

export class ColorInfoPromiseClient {
  constructor (hostname: string,
               credentials?: null | { [index: string]: string; },
               options?: null | { [index: string]: any; });

  describeColor(
    request: smartcore_bos_color_v1_color_pb.DescribeColorRequest,
    metadata?: grpcWeb.Metadata
  ): Promise<smartcore_bos_color_v1_color_pb.ColorSupport>;

}

