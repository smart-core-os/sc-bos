import * as grpcWeb from 'grpc-web';

import * as smartcore_bos_airqualitysensor_v1_air_quality_sensor_history_pb from '../../../../smartcore/bos/airqualitysensor/v1/air_quality_sensor_history_pb'; // proto import: "smartcore/bos/airqualitysensor/v1/air_quality_sensor_history.proto"


export class AirQualitySensorHistoryClient {
  constructor (hostname: string,
               credentials?: null | { [index: string]: string; },
               options?: null | { [index: string]: any; });

  listAirQualityHistory(
    request: smartcore_bos_airqualitysensor_v1_air_quality_sensor_history_pb.ListAirQualityHistoryRequest,
    metadata: grpcWeb.Metadata | undefined,
    callback: (err: grpcWeb.RpcError,
               response: smartcore_bos_airqualitysensor_v1_air_quality_sensor_history_pb.ListAirQualityHistoryResponse) => void
  ): grpcWeb.ClientReadableStream<smartcore_bos_airqualitysensor_v1_air_quality_sensor_history_pb.ListAirQualityHistoryResponse>;

}

export class AirQualitySensorHistoryPromiseClient {
  constructor (hostname: string,
               credentials?: null | { [index: string]: string; },
               options?: null | { [index: string]: any; });

  listAirQualityHistory(
    request: smartcore_bos_airqualitysensor_v1_air_quality_sensor_history_pb.ListAirQualityHistoryRequest,
    metadata?: grpcWeb.Metadata
  ): Promise<smartcore_bos_airqualitysensor_v1_air_quality_sensor_history_pb.ListAirQualityHistoryResponse>;

}

