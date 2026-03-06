import * as grpcWeb from 'grpc-web';

import * as traits_temperature_pb from '../traits/temperature_pb'; // proto import: "traits/temperature.proto"


export class TemperatureApiClient {
  constructor (hostname: string,
               credentials?: null | { [index: string]: string; },
               options?: null | { [index: string]: any; });

  getTemperature(
    request: traits_temperature_pb.GetTemperatureRequest,
    metadata: grpcWeb.Metadata | undefined,
    callback: (err: grpcWeb.RpcError,
               response: traits_temperature_pb.Temperature) => void
  ): grpcWeb.ClientReadableStream<traits_temperature_pb.Temperature>;

  pullTemperature(
    request: traits_temperature_pb.PullTemperatureRequest,
    metadata?: grpcWeb.Metadata
  ): grpcWeb.ClientReadableStream<traits_temperature_pb.PullTemperatureResponse>;

  updateTemperature(
    request: traits_temperature_pb.UpdateTemperatureRequest,
    metadata: grpcWeb.Metadata | undefined,
    callback: (err: grpcWeb.RpcError,
               response: traits_temperature_pb.Temperature) => void
  ): grpcWeb.ClientReadableStream<traits_temperature_pb.Temperature>;

}

export class TemperatureApiPromiseClient {
  constructor (hostname: string,
               credentials?: null | { [index: string]: string; },
               options?: null | { [index: string]: any; });

  getTemperature(
    request: traits_temperature_pb.GetTemperatureRequest,
    metadata?: grpcWeb.Metadata
  ): Promise<traits_temperature_pb.Temperature>;

  pullTemperature(
    request: traits_temperature_pb.PullTemperatureRequest,
    metadata?: grpcWeb.Metadata
  ): grpcWeb.ClientReadableStream<traits_temperature_pb.PullTemperatureResponse>;

  updateTemperature(
    request: traits_temperature_pb.UpdateTemperatureRequest,
    metadata?: grpcWeb.Metadata
  ): Promise<traits_temperature_pb.Temperature>;

}

