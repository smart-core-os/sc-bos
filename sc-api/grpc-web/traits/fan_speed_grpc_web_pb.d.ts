import * as grpcWeb from 'grpc-web';

import * as traits_fan_speed_pb from '../traits/fan_speed_pb'; // proto import: "traits/fan_speed.proto"


export class FanSpeedApiClient {
  constructor (hostname: string,
               credentials?: null | { [index: string]: string; },
               options?: null | { [index: string]: any; });

  getFanSpeed(
    request: traits_fan_speed_pb.GetFanSpeedRequest,
    metadata: grpcWeb.Metadata | undefined,
    callback: (err: grpcWeb.RpcError,
               response: traits_fan_speed_pb.FanSpeed) => void
  ): grpcWeb.ClientReadableStream<traits_fan_speed_pb.FanSpeed>;

  updateFanSpeed(
    request: traits_fan_speed_pb.UpdateFanSpeedRequest,
    metadata: grpcWeb.Metadata | undefined,
    callback: (err: grpcWeb.RpcError,
               response: traits_fan_speed_pb.FanSpeed) => void
  ): grpcWeb.ClientReadableStream<traits_fan_speed_pb.FanSpeed>;

  pullFanSpeed(
    request: traits_fan_speed_pb.PullFanSpeedRequest,
    metadata?: grpcWeb.Metadata
  ): grpcWeb.ClientReadableStream<traits_fan_speed_pb.PullFanSpeedResponse>;

  reverseFanSpeedDirection(
    request: traits_fan_speed_pb.ReverseFanSpeedDirectionRequest,
    metadata: grpcWeb.Metadata | undefined,
    callback: (err: grpcWeb.RpcError,
               response: traits_fan_speed_pb.FanSpeed) => void
  ): grpcWeb.ClientReadableStream<traits_fan_speed_pb.FanSpeed>;

}

export class FanSpeedInfoClient {
  constructor (hostname: string,
               credentials?: null | { [index: string]: string; },
               options?: null | { [index: string]: any; });

  describeFanSpeed(
    request: traits_fan_speed_pb.DescribeFanSpeedRequest,
    metadata: grpcWeb.Metadata | undefined,
    callback: (err: grpcWeb.RpcError,
               response: traits_fan_speed_pb.FanSpeedSupport) => void
  ): grpcWeb.ClientReadableStream<traits_fan_speed_pb.FanSpeedSupport>;

}

export class FanSpeedApiPromiseClient {
  constructor (hostname: string,
               credentials?: null | { [index: string]: string; },
               options?: null | { [index: string]: any; });

  getFanSpeed(
    request: traits_fan_speed_pb.GetFanSpeedRequest,
    metadata?: grpcWeb.Metadata
  ): Promise<traits_fan_speed_pb.FanSpeed>;

  updateFanSpeed(
    request: traits_fan_speed_pb.UpdateFanSpeedRequest,
    metadata?: grpcWeb.Metadata
  ): Promise<traits_fan_speed_pb.FanSpeed>;

  pullFanSpeed(
    request: traits_fan_speed_pb.PullFanSpeedRequest,
    metadata?: grpcWeb.Metadata
  ): grpcWeb.ClientReadableStream<traits_fan_speed_pb.PullFanSpeedResponse>;

  reverseFanSpeedDirection(
    request: traits_fan_speed_pb.ReverseFanSpeedDirectionRequest,
    metadata?: grpcWeb.Metadata
  ): Promise<traits_fan_speed_pb.FanSpeed>;

}

export class FanSpeedInfoPromiseClient {
  constructor (hostname: string,
               credentials?: null | { [index: string]: string; },
               options?: null | { [index: string]: any; });

  describeFanSpeed(
    request: traits_fan_speed_pb.DescribeFanSpeedRequest,
    metadata?: grpcWeb.Metadata
  ): Promise<traits_fan_speed_pb.FanSpeedSupport>;

}

