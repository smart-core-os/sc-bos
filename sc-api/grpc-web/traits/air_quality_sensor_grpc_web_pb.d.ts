import * as grpcWeb from 'grpc-web';

import * as traits_air_quality_sensor_pb from '../traits/air_quality_sensor_pb'; // proto import: "traits/air_quality_sensor.proto"


export class AirQualitySensorApiClient {
  constructor (hostname: string,
               credentials?: null | { [index: string]: string; },
               options?: null | { [index: string]: any; });

  getAirQuality(
    request: traits_air_quality_sensor_pb.GetAirQualityRequest,
    metadata: grpcWeb.Metadata | undefined,
    callback: (err: grpcWeb.RpcError,
               response: traits_air_quality_sensor_pb.AirQuality) => void
  ): grpcWeb.ClientReadableStream<traits_air_quality_sensor_pb.AirQuality>;

  pullAirQuality(
    request: traits_air_quality_sensor_pb.PullAirQualityRequest,
    metadata?: grpcWeb.Metadata
  ): grpcWeb.ClientReadableStream<traits_air_quality_sensor_pb.PullAirQualityResponse>;

}

export class AirQualitySensorInfoClient {
  constructor (hostname: string,
               credentials?: null | { [index: string]: string; },
               options?: null | { [index: string]: any; });

  describeAirQuality(
    request: traits_air_quality_sensor_pb.DescribeAirQualityRequest,
    metadata: grpcWeb.Metadata | undefined,
    callback: (err: grpcWeb.RpcError,
               response: traits_air_quality_sensor_pb.AirQualitySupport) => void
  ): grpcWeb.ClientReadableStream<traits_air_quality_sensor_pb.AirQualitySupport>;

}

export class AirQualitySensorApiPromiseClient {
  constructor (hostname: string,
               credentials?: null | { [index: string]: string; },
               options?: null | { [index: string]: any; });

  getAirQuality(
    request: traits_air_quality_sensor_pb.GetAirQualityRequest,
    metadata?: grpcWeb.Metadata
  ): Promise<traits_air_quality_sensor_pb.AirQuality>;

  pullAirQuality(
    request: traits_air_quality_sensor_pb.PullAirQualityRequest,
    metadata?: grpcWeb.Metadata
  ): grpcWeb.ClientReadableStream<traits_air_quality_sensor_pb.PullAirQualityResponse>;

}

export class AirQualitySensorInfoPromiseClient {
  constructor (hostname: string,
               credentials?: null | { [index: string]: string; },
               options?: null | { [index: string]: any; });

  describeAirQuality(
    request: traits_air_quality_sensor_pb.DescribeAirQualityRequest,
    metadata?: grpcWeb.Metadata
  ): Promise<traits_air_quality_sensor_pb.AirQualitySupport>;

}

