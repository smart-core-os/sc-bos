import * as grpcWeb from 'grpc-web';

import * as smartcore_bos_brightnesssensor_v1_brightness_sensor_pb from '../../../../smartcore/bos/brightnesssensor/v1/brightness_sensor_pb'; // proto import: "smartcore/bos/brightnesssensor/v1/brightness_sensor.proto"


export class BrightnessSensorApiClient {
  constructor (hostname: string,
               credentials?: null | { [index: string]: string; },
               options?: null | { [index: string]: any; });

  getAmbientBrightness(
    request: smartcore_bos_brightnesssensor_v1_brightness_sensor_pb.GetAmbientBrightnessRequest,
    metadata: grpcWeb.Metadata | undefined,
    callback: (err: grpcWeb.RpcError,
               response: smartcore_bos_brightnesssensor_v1_brightness_sensor_pb.AmbientBrightness) => void
  ): grpcWeb.ClientReadableStream<smartcore_bos_brightnesssensor_v1_brightness_sensor_pb.AmbientBrightness>;

  pullAmbientBrightness(
    request: smartcore_bos_brightnesssensor_v1_brightness_sensor_pb.PullAmbientBrightnessRequest,
    metadata?: grpcWeb.Metadata
  ): grpcWeb.ClientReadableStream<smartcore_bos_brightnesssensor_v1_brightness_sensor_pb.PullAmbientBrightnessResponse>;

}

export class BrightnessSensorInfoClient {
  constructor (hostname: string,
               credentials?: null | { [index: string]: string; },
               options?: null | { [index: string]: any; });

  describeAmbientBrightness(
    request: smartcore_bos_brightnesssensor_v1_brightness_sensor_pb.DescribeAmbientBrightnessRequest,
    metadata: grpcWeb.Metadata | undefined,
    callback: (err: grpcWeb.RpcError,
               response: smartcore_bos_brightnesssensor_v1_brightness_sensor_pb.AmbientBrightnessSupport) => void
  ): grpcWeb.ClientReadableStream<smartcore_bos_brightnesssensor_v1_brightness_sensor_pb.AmbientBrightnessSupport>;

}

export class BrightnessSensorApiPromiseClient {
  constructor (hostname: string,
               credentials?: null | { [index: string]: string; },
               options?: null | { [index: string]: any; });

  getAmbientBrightness(
    request: smartcore_bos_brightnesssensor_v1_brightness_sensor_pb.GetAmbientBrightnessRequest,
    metadata?: grpcWeb.Metadata
  ): Promise<smartcore_bos_brightnesssensor_v1_brightness_sensor_pb.AmbientBrightness>;

  pullAmbientBrightness(
    request: smartcore_bos_brightnesssensor_v1_brightness_sensor_pb.PullAmbientBrightnessRequest,
    metadata?: grpcWeb.Metadata
  ): grpcWeb.ClientReadableStream<smartcore_bos_brightnesssensor_v1_brightness_sensor_pb.PullAmbientBrightnessResponse>;

}

export class BrightnessSensorInfoPromiseClient {
  constructor (hostname: string,
               credentials?: null | { [index: string]: string; },
               options?: null | { [index: string]: any; });

  describeAmbientBrightness(
    request: smartcore_bos_brightnesssensor_v1_brightness_sensor_pb.DescribeAmbientBrightnessRequest,
    metadata?: grpcWeb.Metadata
  ): Promise<smartcore_bos_brightnesssensor_v1_brightness_sensor_pb.AmbientBrightnessSupport>;

}

