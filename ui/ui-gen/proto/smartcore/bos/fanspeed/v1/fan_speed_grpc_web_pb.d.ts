import * as grpcWeb from 'grpc-web';

import * as smartcore_bos_fanspeed_v1_fan_speed_pb from '../../../../smartcore/bos/fanspeed/v1/fan_speed_pb'; // proto import: "smartcore/bos/fanspeed/v1/fan_speed.proto"


export class FanSpeedApiClient {
  constructor (hostname: string,
               credentials?: null | { [index: string]: string; },
               options?: null | { [index: string]: any; });

  getFanSpeed(
    request: smartcore_bos_fanspeed_v1_fan_speed_pb.GetFanSpeedRequest,
    metadata: grpcWeb.Metadata | undefined,
    callback: (err: grpcWeb.RpcError,
               response: smartcore_bos_fanspeed_v1_fan_speed_pb.FanSpeed) => void
  ): grpcWeb.ClientReadableStream<smartcore_bos_fanspeed_v1_fan_speed_pb.FanSpeed>;

  updateFanSpeed(
    request: smartcore_bos_fanspeed_v1_fan_speed_pb.UpdateFanSpeedRequest,
    metadata: grpcWeb.Metadata | undefined,
    callback: (err: grpcWeb.RpcError,
               response: smartcore_bos_fanspeed_v1_fan_speed_pb.FanSpeed) => void
  ): grpcWeb.ClientReadableStream<smartcore_bos_fanspeed_v1_fan_speed_pb.FanSpeed>;

  pullFanSpeed(
    request: smartcore_bos_fanspeed_v1_fan_speed_pb.PullFanSpeedRequest,
    metadata?: grpcWeb.Metadata
  ): grpcWeb.ClientReadableStream<smartcore_bos_fanspeed_v1_fan_speed_pb.PullFanSpeedResponse>;

  reverseFanSpeedDirection(
    request: smartcore_bos_fanspeed_v1_fan_speed_pb.ReverseFanSpeedDirectionRequest,
    metadata: grpcWeb.Metadata | undefined,
    callback: (err: grpcWeb.RpcError,
               response: smartcore_bos_fanspeed_v1_fan_speed_pb.FanSpeed) => void
  ): grpcWeb.ClientReadableStream<smartcore_bos_fanspeed_v1_fan_speed_pb.FanSpeed>;

}

export class FanSpeedInfoClient {
  constructor (hostname: string,
               credentials?: null | { [index: string]: string; },
               options?: null | { [index: string]: any; });

  describeFanSpeed(
    request: smartcore_bos_fanspeed_v1_fan_speed_pb.DescribeFanSpeedRequest,
    metadata: grpcWeb.Metadata | undefined,
    callback: (err: grpcWeb.RpcError,
               response: smartcore_bos_fanspeed_v1_fan_speed_pb.FanSpeedSupport) => void
  ): grpcWeb.ClientReadableStream<smartcore_bos_fanspeed_v1_fan_speed_pb.FanSpeedSupport>;

}

export class FanSpeedApiPromiseClient {
  constructor (hostname: string,
               credentials?: null | { [index: string]: string; },
               options?: null | { [index: string]: any; });

  getFanSpeed(
    request: smartcore_bos_fanspeed_v1_fan_speed_pb.GetFanSpeedRequest,
    metadata?: grpcWeb.Metadata
  ): Promise<smartcore_bos_fanspeed_v1_fan_speed_pb.FanSpeed>;

  updateFanSpeed(
    request: smartcore_bos_fanspeed_v1_fan_speed_pb.UpdateFanSpeedRequest,
    metadata?: grpcWeb.Metadata
  ): Promise<smartcore_bos_fanspeed_v1_fan_speed_pb.FanSpeed>;

  pullFanSpeed(
    request: smartcore_bos_fanspeed_v1_fan_speed_pb.PullFanSpeedRequest,
    metadata?: grpcWeb.Metadata
  ): grpcWeb.ClientReadableStream<smartcore_bos_fanspeed_v1_fan_speed_pb.PullFanSpeedResponse>;

  reverseFanSpeedDirection(
    request: smartcore_bos_fanspeed_v1_fan_speed_pb.ReverseFanSpeedDirectionRequest,
    metadata?: grpcWeb.Metadata
  ): Promise<smartcore_bos_fanspeed_v1_fan_speed_pb.FanSpeed>;

}

export class FanSpeedInfoPromiseClient {
  constructor (hostname: string,
               credentials?: null | { [index: string]: string; },
               options?: null | { [index: string]: any; });

  describeFanSpeed(
    request: smartcore_bos_fanspeed_v1_fan_speed_pb.DescribeFanSpeedRequest,
    metadata?: grpcWeb.Metadata
  ): Promise<smartcore_bos_fanspeed_v1_fan_speed_pb.FanSpeedSupport>;

}

