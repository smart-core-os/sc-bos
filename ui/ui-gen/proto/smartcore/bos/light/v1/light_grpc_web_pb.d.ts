import * as grpcWeb from 'grpc-web';

import * as smartcore_bos_light_v1_light_pb from '../../../../smartcore/bos/light/v1/light_pb'; // proto import: "smartcore/bos/light/v1/light.proto"


export class LightApiClient {
  constructor (hostname: string,
               credentials?: null | { [index: string]: string; },
               options?: null | { [index: string]: any; });

  updateBrightness(
    request: smartcore_bos_light_v1_light_pb.UpdateBrightnessRequest,
    metadata: grpcWeb.Metadata | undefined,
    callback: (err: grpcWeb.RpcError,
               response: smartcore_bos_light_v1_light_pb.Brightness) => void
  ): grpcWeb.ClientReadableStream<smartcore_bos_light_v1_light_pb.Brightness>;

  getBrightness(
    request: smartcore_bos_light_v1_light_pb.GetBrightnessRequest,
    metadata: grpcWeb.Metadata | undefined,
    callback: (err: grpcWeb.RpcError,
               response: smartcore_bos_light_v1_light_pb.Brightness) => void
  ): grpcWeb.ClientReadableStream<smartcore_bos_light_v1_light_pb.Brightness>;

  pullBrightness(
    request: smartcore_bos_light_v1_light_pb.PullBrightnessRequest,
    metadata?: grpcWeb.Metadata
  ): grpcWeb.ClientReadableStream<smartcore_bos_light_v1_light_pb.PullBrightnessResponse>;

}

export class LightInfoClient {
  constructor (hostname: string,
               credentials?: null | { [index: string]: string; },
               options?: null | { [index: string]: any; });

  describeBrightness(
    request: smartcore_bos_light_v1_light_pb.DescribeBrightnessRequest,
    metadata: grpcWeb.Metadata | undefined,
    callback: (err: grpcWeb.RpcError,
               response: smartcore_bos_light_v1_light_pb.BrightnessSupport) => void
  ): grpcWeb.ClientReadableStream<smartcore_bos_light_v1_light_pb.BrightnessSupport>;

}

export class LightApiPromiseClient {
  constructor (hostname: string,
               credentials?: null | { [index: string]: string; },
               options?: null | { [index: string]: any; });

  updateBrightness(
    request: smartcore_bos_light_v1_light_pb.UpdateBrightnessRequest,
    metadata?: grpcWeb.Metadata
  ): Promise<smartcore_bos_light_v1_light_pb.Brightness>;

  getBrightness(
    request: smartcore_bos_light_v1_light_pb.GetBrightnessRequest,
    metadata?: grpcWeb.Metadata
  ): Promise<smartcore_bos_light_v1_light_pb.Brightness>;

  pullBrightness(
    request: smartcore_bos_light_v1_light_pb.PullBrightnessRequest,
    metadata?: grpcWeb.Metadata
  ): grpcWeb.ClientReadableStream<smartcore_bos_light_v1_light_pb.PullBrightnessResponse>;

}

export class LightInfoPromiseClient {
  constructor (hostname: string,
               credentials?: null | { [index: string]: string; },
               options?: null | { [index: string]: any; });

  describeBrightness(
    request: smartcore_bos_light_v1_light_pb.DescribeBrightnessRequest,
    metadata?: grpcWeb.Metadata
  ): Promise<smartcore_bos_light_v1_light_pb.BrightnessSupport>;

}

