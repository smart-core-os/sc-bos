import * as grpcWeb from 'grpc-web';

import * as occupancy_sensor_history_pb from './occupancy_sensor_history_pb'; // proto import: "occupancy_sensor_history.proto"


export class OccupancySensorHistoryClient {
  constructor (hostname: string,
               credentials?: null | { [index: string]: string; },
               options?: null | { [index: string]: any; });

  listOccupancyHistory(
    request: occupancy_sensor_history_pb.ListOccupancyHistoryRequest,
    metadata: grpcWeb.Metadata | undefined,
    callback: (err: grpcWeb.RpcError,
               response: occupancy_sensor_history_pb.ListOccupancyHistoryResponse) => void
  ): grpcWeb.ClientReadableStream<occupancy_sensor_history_pb.ListOccupancyHistoryResponse>;

}

export class OccupancySensorHistoryPromiseClient {
  constructor (hostname: string,
               credentials?: null | { [index: string]: string; },
               options?: null | { [index: string]: any; });

  listOccupancyHistory(
    request: occupancy_sensor_history_pb.ListOccupancyHistoryRequest,
    metadata?: grpcWeb.Metadata
  ): Promise<occupancy_sensor_history_pb.ListOccupancyHistoryResponse>;

}

