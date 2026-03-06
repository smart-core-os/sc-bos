import * as grpcWeb from 'grpc-web';

import * as traits_motion_sensor_pb from '../traits/motion_sensor_pb'; // proto import: "traits/motion_sensor.proto"


export class MotionSensorApiClient {
  constructor (hostname: string,
               credentials?: null | { [index: string]: string; },
               options?: null | { [index: string]: any; });

  getMotionDetection(
    request: traits_motion_sensor_pb.GetMotionDetectionRequest,
    metadata: grpcWeb.Metadata | undefined,
    callback: (err: grpcWeb.RpcError,
               response: traits_motion_sensor_pb.MotionDetection) => void
  ): grpcWeb.ClientReadableStream<traits_motion_sensor_pb.MotionDetection>;

  pullMotionDetections(
    request: traits_motion_sensor_pb.PullMotionDetectionRequest,
    metadata?: grpcWeb.Metadata
  ): grpcWeb.ClientReadableStream<traits_motion_sensor_pb.PullMotionDetectionResponse>;

}

export class MotionSensorSensorInfoClient {
  constructor (hostname: string,
               credentials?: null | { [index: string]: string; },
               options?: null | { [index: string]: any; });

  describeMotionDetection(
    request: traits_motion_sensor_pb.DescribeMotionDetectionRequest,
    metadata: grpcWeb.Metadata | undefined,
    callback: (err: grpcWeb.RpcError,
               response: traits_motion_sensor_pb.MotionDetectionSupport) => void
  ): grpcWeb.ClientReadableStream<traits_motion_sensor_pb.MotionDetectionSupport>;

}

export class MotionSensorApiPromiseClient {
  constructor (hostname: string,
               credentials?: null | { [index: string]: string; },
               options?: null | { [index: string]: any; });

  getMotionDetection(
    request: traits_motion_sensor_pb.GetMotionDetectionRequest,
    metadata?: grpcWeb.Metadata
  ): Promise<traits_motion_sensor_pb.MotionDetection>;

  pullMotionDetections(
    request: traits_motion_sensor_pb.PullMotionDetectionRequest,
    metadata?: grpcWeb.Metadata
  ): grpcWeb.ClientReadableStream<traits_motion_sensor_pb.PullMotionDetectionResponse>;

}

export class MotionSensorSensorInfoPromiseClient {
  constructor (hostname: string,
               credentials?: null | { [index: string]: string; },
               options?: null | { [index: string]: any; });

  describeMotionDetection(
    request: traits_motion_sensor_pb.DescribeMotionDetectionRequest,
    metadata?: grpcWeb.Metadata
  ): Promise<traits_motion_sensor_pb.MotionDetectionSupport>;

}

