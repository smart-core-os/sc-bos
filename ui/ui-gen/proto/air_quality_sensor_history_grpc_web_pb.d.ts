import * as grpcWeb from 'grpc-web';

import * as air_quality_sensor_history_pb from './air_quality_sensor_history_pb'; // proto import: "air_quality_sensor_history.proto"


export class AirQualitySensorHistoryClient {
  constructor (hostname: string,
               credentials?: null | { [index: string]: string; },
               options?: null | { [index: string]: any; });

  listAirQualityHistory(
    request: air_quality_sensor_history_pb.ListAirQualityHistoryRequest,
    metadata: grpcWeb.Metadata | undefined,
    callback: (err: grpcWeb.RpcError,
               response: air_quality_sensor_history_pb.ListAirQualityHistoryResponse) => void
  ): grpcWeb.ClientReadableStream<air_quality_sensor_history_pb.ListAirQualityHistoryResponse>;

}

export class AirQualitySensorHistoryPromiseClient {
  constructor (hostname: string,
               credentials?: null | { [index: string]: string; },
               options?: null | { [index: string]: any; });

  listAirQualityHistory(
    request: air_quality_sensor_history_pb.ListAirQualityHistoryRequest,
    metadata?: grpcWeb.Metadata
  ): Promise<air_quality_sensor_history_pb.ListAirQualityHistoryResponse>;

}

