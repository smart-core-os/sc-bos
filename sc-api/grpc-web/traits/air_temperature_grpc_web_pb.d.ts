import * as grpcWeb from 'grpc-web';

import * as traits_air_temperature_pb from '../traits/air_temperature_pb'; // proto import: "traits/air_temperature.proto"


export class AirTemperatureApiClient {
  constructor (hostname: string,
               credentials?: null | { [index: string]: string; },
               options?: null | { [index: string]: any; });

  getAirTemperature(
    request: traits_air_temperature_pb.GetAirTemperatureRequest,
    metadata: grpcWeb.Metadata | undefined,
    callback: (err: grpcWeb.RpcError,
               response: traits_air_temperature_pb.AirTemperature) => void
  ): grpcWeb.ClientReadableStream<traits_air_temperature_pb.AirTemperature>;

  updateAirTemperature(
    request: traits_air_temperature_pb.UpdateAirTemperatureRequest,
    metadata: grpcWeb.Metadata | undefined,
    callback: (err: grpcWeb.RpcError,
               response: traits_air_temperature_pb.AirTemperature) => void
  ): grpcWeb.ClientReadableStream<traits_air_temperature_pb.AirTemperature>;

  pullAirTemperature(
    request: traits_air_temperature_pb.PullAirTemperatureRequest,
    metadata?: grpcWeb.Metadata
  ): grpcWeb.ClientReadableStream<traits_air_temperature_pb.PullAirTemperatureResponse>;

}

export class AirTemperatureInfoClient {
  constructor (hostname: string,
               credentials?: null | { [index: string]: string; },
               options?: null | { [index: string]: any; });

  describeAirTemperature(
    request: traits_air_temperature_pb.DescribeAirTemperatureRequest,
    metadata: grpcWeb.Metadata | undefined,
    callback: (err: grpcWeb.RpcError,
               response: traits_air_temperature_pb.AirTemperatureSupport) => void
  ): grpcWeb.ClientReadableStream<traits_air_temperature_pb.AirTemperatureSupport>;

}

export class AirTemperatureApiPromiseClient {
  constructor (hostname: string,
               credentials?: null | { [index: string]: string; },
               options?: null | { [index: string]: any; });

  getAirTemperature(
    request: traits_air_temperature_pb.GetAirTemperatureRequest,
    metadata?: grpcWeb.Metadata
  ): Promise<traits_air_temperature_pb.AirTemperature>;

  updateAirTemperature(
    request: traits_air_temperature_pb.UpdateAirTemperatureRequest,
    metadata?: grpcWeb.Metadata
  ): Promise<traits_air_temperature_pb.AirTemperature>;

  pullAirTemperature(
    request: traits_air_temperature_pb.PullAirTemperatureRequest,
    metadata?: grpcWeb.Metadata
  ): grpcWeb.ClientReadableStream<traits_air_temperature_pb.PullAirTemperatureResponse>;

}

export class AirTemperatureInfoPromiseClient {
  constructor (hostname: string,
               credentials?: null | { [index: string]: string; },
               options?: null | { [index: string]: any; });

  describeAirTemperature(
    request: traits_air_temperature_pb.DescribeAirTemperatureRequest,
    metadata?: grpcWeb.Metadata
  ): Promise<traits_air_temperature_pb.AirTemperatureSupport>;

}

