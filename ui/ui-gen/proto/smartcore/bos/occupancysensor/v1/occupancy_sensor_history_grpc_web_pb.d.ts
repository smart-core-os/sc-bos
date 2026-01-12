import * as grpcWeb from 'grpc-web';

import * as smartcore_bos_occupancysensor_v1_occupancy_sensor_history_pb from '../../../../smartcore/bos/occupancysensor/v1/occupancy_sensor_history_pb'; // proto import: "smartcore/bos/occupancysensor/v1/occupancy_sensor_history.proto"


export class OccupancySensorHistoryClient {
  constructor (hostname: string,
               credentials?: null | { [index: string]: string; },
               options?: null | { [index: string]: any; });

  listOccupancyHistory(
    request: smartcore_bos_occupancysensor_v1_occupancy_sensor_history_pb.ListOccupancyHistoryRequest,
    metadata: grpcWeb.Metadata | undefined,
    callback: (err: grpcWeb.RpcError,
               response: smartcore_bos_occupancysensor_v1_occupancy_sensor_history_pb.ListOccupancyHistoryResponse) => void
  ): grpcWeb.ClientReadableStream<smartcore_bos_occupancysensor_v1_occupancy_sensor_history_pb.ListOccupancyHistoryResponse>;

}

export class OccupancySensorHistoryPromiseClient {
  constructor (hostname: string,
               credentials?: null | { [index: string]: string; },
               options?: null | { [index: string]: any; });

  listOccupancyHistory(
    request: smartcore_bos_occupancysensor_v1_occupancy_sensor_history_pb.ListOccupancyHistoryRequest,
    metadata?: grpcWeb.Metadata
  ): Promise<smartcore_bos_occupancysensor_v1_occupancy_sensor_history_pb.ListOccupancyHistoryResponse>;

}

