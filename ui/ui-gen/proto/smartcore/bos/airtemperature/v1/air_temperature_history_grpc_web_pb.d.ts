import * as grpcWeb from 'grpc-web';

import * as smartcore_bos_airtemperature_v1_air_temperature_history_pb from '../../../../smartcore/bos/airtemperature/v1/air_temperature_history_pb'; // proto import: "smartcore/bos/airtemperature/v1/air_temperature_history.proto"


export class AirTemperatureHistoryClient {
  constructor (hostname: string,
               credentials?: null | { [index: string]: string; },
               options?: null | { [index: string]: any; });

  listAirTemperatureHistory(
    request: smartcore_bos_airtemperature_v1_air_temperature_history_pb.ListAirTemperatureHistoryRequest,
    metadata: grpcWeb.Metadata | undefined,
    callback: (err: grpcWeb.RpcError,
               response: smartcore_bos_airtemperature_v1_air_temperature_history_pb.ListAirTemperatureHistoryResponse) => void
  ): grpcWeb.ClientReadableStream<smartcore_bos_airtemperature_v1_air_temperature_history_pb.ListAirTemperatureHistoryResponse>;

}

export class AirTemperatureHistoryPromiseClient {
  constructor (hostname: string,
               credentials?: null | { [index: string]: string; },
               options?: null | { [index: string]: any; });

  listAirTemperatureHistory(
    request: smartcore_bos_airtemperature_v1_air_temperature_history_pb.ListAirTemperatureHistoryRequest,
    metadata?: grpcWeb.Metadata
  ): Promise<smartcore_bos_airtemperature_v1_air_temperature_history_pb.ListAirTemperatureHistoryResponse>;

}

