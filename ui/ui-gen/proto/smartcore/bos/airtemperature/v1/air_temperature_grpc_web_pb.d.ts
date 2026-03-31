import * as grpcWeb from 'grpc-web';

import * as smartcore_bos_airtemperature_v1_air_temperature_pb from '../../../../smartcore/bos/airtemperature/v1/air_temperature_pb'; // proto import: "smartcore/bos/airtemperature/v1/air_temperature.proto"


export class AirTemperatureApiClient {
  constructor (hostname: string,
               credentials?: null | { [index: string]: string; },
               options?: null | { [index: string]: any; });

  getAirTemperature(
    request: smartcore_bos_airtemperature_v1_air_temperature_pb.GetAirTemperatureRequest,
    metadata: grpcWeb.Metadata | undefined,
    callback: (err: grpcWeb.RpcError,
               response: smartcore_bos_airtemperature_v1_air_temperature_pb.AirTemperature) => void
  ): grpcWeb.ClientReadableStream<smartcore_bos_airtemperature_v1_air_temperature_pb.AirTemperature>;

  updateAirTemperature(
    request: smartcore_bos_airtemperature_v1_air_temperature_pb.UpdateAirTemperatureRequest,
    metadata: grpcWeb.Metadata | undefined,
    callback: (err: grpcWeb.RpcError,
               response: smartcore_bos_airtemperature_v1_air_temperature_pb.AirTemperature) => void
  ): grpcWeb.ClientReadableStream<smartcore_bos_airtemperature_v1_air_temperature_pb.AirTemperature>;

  pullAirTemperature(
    request: smartcore_bos_airtemperature_v1_air_temperature_pb.PullAirTemperatureRequest,
    metadata?: grpcWeb.Metadata
  ): grpcWeb.ClientReadableStream<smartcore_bos_airtemperature_v1_air_temperature_pb.PullAirTemperatureResponse>;

}

export class AirTemperatureInfoClient {
  constructor (hostname: string,
               credentials?: null | { [index: string]: string; },
               options?: null | { [index: string]: any; });

  describeAirTemperature(
    request: smartcore_bos_airtemperature_v1_air_temperature_pb.DescribeAirTemperatureRequest,
    metadata: grpcWeb.Metadata | undefined,
    callback: (err: grpcWeb.RpcError,
               response: smartcore_bos_airtemperature_v1_air_temperature_pb.AirTemperatureSupport) => void
  ): grpcWeb.ClientReadableStream<smartcore_bos_airtemperature_v1_air_temperature_pb.AirTemperatureSupport>;

}

export class AirTemperatureApiPromiseClient {
  constructor (hostname: string,
               credentials?: null | { [index: string]: string; },
               options?: null | { [index: string]: any; });

  getAirTemperature(
    request: smartcore_bos_airtemperature_v1_air_temperature_pb.GetAirTemperatureRequest,
    metadata?: grpcWeb.Metadata
  ): Promise<smartcore_bos_airtemperature_v1_air_temperature_pb.AirTemperature>;

  updateAirTemperature(
    request: smartcore_bos_airtemperature_v1_air_temperature_pb.UpdateAirTemperatureRequest,
    metadata?: grpcWeb.Metadata
  ): Promise<smartcore_bos_airtemperature_v1_air_temperature_pb.AirTemperature>;

  pullAirTemperature(
    request: smartcore_bos_airtemperature_v1_air_temperature_pb.PullAirTemperatureRequest,
    metadata?: grpcWeb.Metadata
  ): grpcWeb.ClientReadableStream<smartcore_bos_airtemperature_v1_air_temperature_pb.PullAirTemperatureResponse>;

}

export class AirTemperatureInfoPromiseClient {
  constructor (hostname: string,
               credentials?: null | { [index: string]: string; },
               options?: null | { [index: string]: any; });

  describeAirTemperature(
    request: smartcore_bos_airtemperature_v1_air_temperature_pb.DescribeAirTemperatureRequest,
    metadata?: grpcWeb.Metadata
  ): Promise<smartcore_bos_airtemperature_v1_air_temperature_pb.AirTemperatureSupport>;

}

