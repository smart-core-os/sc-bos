import * as grpcWeb from 'grpc-web';

import * as traits_light_pb from '../traits/light_pb'; // proto import: "traits/light.proto"


export class LightApiClient {
  constructor (hostname: string,
               credentials?: null | { [index: string]: string; },
               options?: null | { [index: string]: any; });

  updateBrightness(
    request: traits_light_pb.UpdateBrightnessRequest,
    metadata: grpcWeb.Metadata | undefined,
    callback: (err: grpcWeb.RpcError,
               response: traits_light_pb.Brightness) => void
  ): grpcWeb.ClientReadableStream<traits_light_pb.Brightness>;

  getBrightness(
    request: traits_light_pb.GetBrightnessRequest,
    metadata: grpcWeb.Metadata | undefined,
    callback: (err: grpcWeb.RpcError,
               response: traits_light_pb.Brightness) => void
  ): grpcWeb.ClientReadableStream<traits_light_pb.Brightness>;

  pullBrightness(
    request: traits_light_pb.PullBrightnessRequest,
    metadata?: grpcWeb.Metadata
  ): grpcWeb.ClientReadableStream<traits_light_pb.PullBrightnessResponse>;

}

export class LightInfoClient {
  constructor (hostname: string,
               credentials?: null | { [index: string]: string; },
               options?: null | { [index: string]: any; });

  describeBrightness(
    request: traits_light_pb.DescribeBrightnessRequest,
    metadata: grpcWeb.Metadata | undefined,
    callback: (err: grpcWeb.RpcError,
               response: traits_light_pb.BrightnessSupport) => void
  ): grpcWeb.ClientReadableStream<traits_light_pb.BrightnessSupport>;

}

export class LightApiPromiseClient {
  constructor (hostname: string,
               credentials?: null | { [index: string]: string; },
               options?: null | { [index: string]: any; });

  updateBrightness(
    request: traits_light_pb.UpdateBrightnessRequest,
    metadata?: grpcWeb.Metadata
  ): Promise<traits_light_pb.Brightness>;

  getBrightness(
    request: traits_light_pb.GetBrightnessRequest,
    metadata?: grpcWeb.Metadata
  ): Promise<traits_light_pb.Brightness>;

  pullBrightness(
    request: traits_light_pb.PullBrightnessRequest,
    metadata?: grpcWeb.Metadata
  ): grpcWeb.ClientReadableStream<traits_light_pb.PullBrightnessResponse>;

}

export class LightInfoPromiseClient {
  constructor (hostname: string,
               credentials?: null | { [index: string]: string; },
               options?: null | { [index: string]: any; });

  describeBrightness(
    request: traits_light_pb.DescribeBrightnessRequest,
    metadata?: grpcWeb.Metadata
  ): Promise<traits_light_pb.BrightnessSupport>;

}

