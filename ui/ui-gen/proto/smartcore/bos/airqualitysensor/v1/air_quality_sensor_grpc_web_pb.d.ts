import * as grpcWeb from 'grpc-web';

import * as smartcore_bos_airqualitysensor_v1_air_quality_sensor_pb from '../../../../smartcore/bos/airqualitysensor/v1/air_quality_sensor_pb'; // proto import: "smartcore/bos/airqualitysensor/v1/air_quality_sensor.proto"


export class AirQualitySensorApiClient {
  constructor (hostname: string,
               credentials?: null | { [index: string]: string; },
               options?: null | { [index: string]: any; });

  getAirQuality(
    request: smartcore_bos_airqualitysensor_v1_air_quality_sensor_pb.GetAirQualityRequest,
    metadata: grpcWeb.Metadata | undefined,
    callback: (err: grpcWeb.RpcError,
               response: smartcore_bos_airqualitysensor_v1_air_quality_sensor_pb.AirQuality) => void
  ): grpcWeb.ClientReadableStream<smartcore_bos_airqualitysensor_v1_air_quality_sensor_pb.AirQuality>;

  pullAirQuality(
    request: smartcore_bos_airqualitysensor_v1_air_quality_sensor_pb.PullAirQualityRequest,
    metadata?: grpcWeb.Metadata
  ): grpcWeb.ClientReadableStream<smartcore_bos_airqualitysensor_v1_air_quality_sensor_pb.PullAirQualityResponse>;

}

export class AirQualitySensorInfoClient {
  constructor (hostname: string,
               credentials?: null | { [index: string]: string; },
               options?: null | { [index: string]: any; });

  describeAirQuality(
    request: smartcore_bos_airqualitysensor_v1_air_quality_sensor_pb.DescribeAirQualityRequest,
    metadata: grpcWeb.Metadata | undefined,
    callback: (err: grpcWeb.RpcError,
               response: smartcore_bos_airqualitysensor_v1_air_quality_sensor_pb.AirQualitySupport) => void
  ): grpcWeb.ClientReadableStream<smartcore_bos_airqualitysensor_v1_air_quality_sensor_pb.AirQualitySupport>;

}

export class AirQualitySensorApiPromiseClient {
  constructor (hostname: string,
               credentials?: null | { [index: string]: string; },
               options?: null | { [index: string]: any; });

  getAirQuality(
    request: smartcore_bos_airqualitysensor_v1_air_quality_sensor_pb.GetAirQualityRequest,
    metadata?: grpcWeb.Metadata
  ): Promise<smartcore_bos_airqualitysensor_v1_air_quality_sensor_pb.AirQuality>;

  pullAirQuality(
    request: smartcore_bos_airqualitysensor_v1_air_quality_sensor_pb.PullAirQualityRequest,
    metadata?: grpcWeb.Metadata
  ): grpcWeb.ClientReadableStream<smartcore_bos_airqualitysensor_v1_air_quality_sensor_pb.PullAirQualityResponse>;

}

export class AirQualitySensorInfoPromiseClient {
  constructor (hostname: string,
               credentials?: null | { [index: string]: string; },
               options?: null | { [index: string]: any; });

  describeAirQuality(
    request: smartcore_bos_airqualitysensor_v1_air_quality_sensor_pb.DescribeAirQualityRequest,
    metadata?: grpcWeb.Metadata
  ): Promise<smartcore_bos_airqualitysensor_v1_air_quality_sensor_pb.AirQualitySupport>;

}

