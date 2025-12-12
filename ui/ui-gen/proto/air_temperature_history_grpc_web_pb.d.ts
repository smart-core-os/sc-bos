import * as grpcWeb from 'grpc-web';

import * as air_temperature_history_pb from './air_temperature_history_pb'; // proto import: "air_temperature_history.proto"


export class AirTemperatureHistoryClient {
  constructor (hostname: string,
               credentials?: null | { [index: string]: string; },
               options?: null | { [index: string]: any; });

  listAirTemperatureHistory(
    request: air_temperature_history_pb.ListAirTemperatureHistoryRequest,
    metadata: grpcWeb.Metadata | undefined,
    callback: (err: grpcWeb.RpcError,
               response: air_temperature_history_pb.ListAirTemperatureHistoryResponse) => void
  ): grpcWeb.ClientReadableStream<air_temperature_history_pb.ListAirTemperatureHistoryResponse>;

}

export class AirTemperatureHistoryPromiseClient {
  constructor (hostname: string,
               credentials?: null | { [index: string]: string; },
               options?: null | { [index: string]: any; });

  listAirTemperatureHistory(
    request: air_temperature_history_pb.ListAirTemperatureHistoryRequest,
    metadata?: grpcWeb.Metadata
  ): Promise<air_temperature_history_pb.ListAirTemperatureHistoryResponse>;

}

