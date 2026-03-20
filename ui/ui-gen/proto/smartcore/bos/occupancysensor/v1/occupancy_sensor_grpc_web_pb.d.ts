import * as grpcWeb from 'grpc-web';

import * as smartcore_bos_occupancysensor_v1_occupancy_sensor_pb from '../../../../smartcore/bos/occupancysensor/v1/occupancy_sensor_pb'; // proto import: "smartcore/bos/occupancysensor/v1/occupancy_sensor.proto"


export class OccupancySensorApiClient {
  constructor (hostname: string,
               credentials?: null | { [index: string]: string; },
               options?: null | { [index: string]: any; });

  getOccupancy(
    request: smartcore_bos_occupancysensor_v1_occupancy_sensor_pb.GetOccupancyRequest,
    metadata: grpcWeb.Metadata | undefined,
    callback: (err: grpcWeb.RpcError,
               response: smartcore_bos_occupancysensor_v1_occupancy_sensor_pb.Occupancy) => void
  ): grpcWeb.ClientReadableStream<smartcore_bos_occupancysensor_v1_occupancy_sensor_pb.Occupancy>;

  pullOccupancy(
    request: smartcore_bos_occupancysensor_v1_occupancy_sensor_pb.PullOccupancyRequest,
    metadata?: grpcWeb.Metadata
  ): grpcWeb.ClientReadableStream<smartcore_bos_occupancysensor_v1_occupancy_sensor_pb.PullOccupancyResponse>;

}

export class OccupancySensorInfoClient {
  constructor (hostname: string,
               credentials?: null | { [index: string]: string; },
               options?: null | { [index: string]: any; });

  describeOccupancy(
    request: smartcore_bos_occupancysensor_v1_occupancy_sensor_pb.DescribeOccupancyRequest,
    metadata: grpcWeb.Metadata | undefined,
    callback: (err: grpcWeb.RpcError,
               response: smartcore_bos_occupancysensor_v1_occupancy_sensor_pb.OccupancySupport) => void
  ): grpcWeb.ClientReadableStream<smartcore_bos_occupancysensor_v1_occupancy_sensor_pb.OccupancySupport>;

}

export class OccupancySensorApiPromiseClient {
  constructor (hostname: string,
               credentials?: null | { [index: string]: string; },
               options?: null | { [index: string]: any; });

  getOccupancy(
    request: smartcore_bos_occupancysensor_v1_occupancy_sensor_pb.GetOccupancyRequest,
    metadata?: grpcWeb.Metadata
  ): Promise<smartcore_bos_occupancysensor_v1_occupancy_sensor_pb.Occupancy>;

  pullOccupancy(
    request: smartcore_bos_occupancysensor_v1_occupancy_sensor_pb.PullOccupancyRequest,
    metadata?: grpcWeb.Metadata
  ): grpcWeb.ClientReadableStream<smartcore_bos_occupancysensor_v1_occupancy_sensor_pb.PullOccupancyResponse>;

}

export class OccupancySensorInfoPromiseClient {
  constructor (hostname: string,
               credentials?: null | { [index: string]: string; },
               options?: null | { [index: string]: any; });

  describeOccupancy(
    request: smartcore_bos_occupancysensor_v1_occupancy_sensor_pb.DescribeOccupancyRequest,
    metadata?: grpcWeb.Metadata
  ): Promise<smartcore_bos_occupancysensor_v1_occupancy_sensor_pb.OccupancySupport>;

}

