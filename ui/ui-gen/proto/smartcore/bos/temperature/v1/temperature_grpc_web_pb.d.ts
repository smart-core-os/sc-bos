import * as grpcWeb from 'grpc-web';

import * as smartcore_bos_temperature_v1_temperature_pb from '../../../../smartcore/bos/temperature/v1/temperature_pb'; // proto import: "smartcore/bos/temperature/v1/temperature.proto"


export class TemperatureApiClient {
  constructor (hostname: string,
               credentials?: null | { [index: string]: string; },
               options?: null | { [index: string]: any; });

  getTemperature(
    request: smartcore_bos_temperature_v1_temperature_pb.GetTemperatureRequest,
    metadata: grpcWeb.Metadata | undefined,
    callback: (err: grpcWeb.RpcError,
               response: smartcore_bos_temperature_v1_temperature_pb.Temperature) => void
  ): grpcWeb.ClientReadableStream<smartcore_bos_temperature_v1_temperature_pb.Temperature>;

  pullTemperature(
    request: smartcore_bos_temperature_v1_temperature_pb.PullTemperatureRequest,
    metadata?: grpcWeb.Metadata
  ): grpcWeb.ClientReadableStream<smartcore_bos_temperature_v1_temperature_pb.PullTemperatureResponse>;

  updateTemperature(
    request: smartcore_bos_temperature_v1_temperature_pb.UpdateTemperatureRequest,
    metadata: grpcWeb.Metadata | undefined,
    callback: (err: grpcWeb.RpcError,
               response: smartcore_bos_temperature_v1_temperature_pb.Temperature) => void
  ): grpcWeb.ClientReadableStream<smartcore_bos_temperature_v1_temperature_pb.Temperature>;

}

export class TemperatureApiPromiseClient {
  constructor (hostname: string,
               credentials?: null | { [index: string]: string; },
               options?: null | { [index: string]: any; });

  getTemperature(
    request: smartcore_bos_temperature_v1_temperature_pb.GetTemperatureRequest,
    metadata?: grpcWeb.Metadata
  ): Promise<smartcore_bos_temperature_v1_temperature_pb.Temperature>;

  pullTemperature(
    request: smartcore_bos_temperature_v1_temperature_pb.PullTemperatureRequest,
    metadata?: grpcWeb.Metadata
  ): grpcWeb.ClientReadableStream<smartcore_bos_temperature_v1_temperature_pb.PullTemperatureResponse>;

  updateTemperature(
    request: smartcore_bos_temperature_v1_temperature_pb.UpdateTemperatureRequest,
    metadata?: grpcWeb.Metadata
  ): Promise<smartcore_bos_temperature_v1_temperature_pb.Temperature>;

}

