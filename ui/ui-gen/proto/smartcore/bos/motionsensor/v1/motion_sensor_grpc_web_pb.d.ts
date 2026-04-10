import * as grpcWeb from 'grpc-web';

import * as smartcore_bos_motionsensor_v1_motion_sensor_pb from '../../../../smartcore/bos/motionsensor/v1/motion_sensor_pb'; // proto import: "smartcore/bos/motionsensor/v1/motion_sensor.proto"


export class MotionSensorApiClient {
  constructor (hostname: string,
               credentials?: null | { [index: string]: string; },
               options?: null | { [index: string]: any; });

  getMotionDetection(
    request: smartcore_bos_motionsensor_v1_motion_sensor_pb.GetMotionDetectionRequest,
    metadata: grpcWeb.Metadata | undefined,
    callback: (err: grpcWeb.RpcError,
               response: smartcore_bos_motionsensor_v1_motion_sensor_pb.MotionDetection) => void
  ): grpcWeb.ClientReadableStream<smartcore_bos_motionsensor_v1_motion_sensor_pb.MotionDetection>;

  pullMotionDetections(
    request: smartcore_bos_motionsensor_v1_motion_sensor_pb.PullMotionDetectionRequest,
    metadata?: grpcWeb.Metadata
  ): grpcWeb.ClientReadableStream<smartcore_bos_motionsensor_v1_motion_sensor_pb.PullMotionDetectionResponse>;

}

export class MotionSensorInfoClient {
  constructor (hostname: string,
               credentials?: null | { [index: string]: string; },
               options?: null | { [index: string]: any; });

  describeMotionDetection(
    request: smartcore_bos_motionsensor_v1_motion_sensor_pb.DescribeMotionDetectionRequest,
    metadata: grpcWeb.Metadata | undefined,
    callback: (err: grpcWeb.RpcError,
               response: smartcore_bos_motionsensor_v1_motion_sensor_pb.MotionDetectionSupport) => void
  ): grpcWeb.ClientReadableStream<smartcore_bos_motionsensor_v1_motion_sensor_pb.MotionDetectionSupport>;

}

export class MotionSensorApiPromiseClient {
  constructor (hostname: string,
               credentials?: null | { [index: string]: string; },
               options?: null | { [index: string]: any; });

  getMotionDetection(
    request: smartcore_bos_motionsensor_v1_motion_sensor_pb.GetMotionDetectionRequest,
    metadata?: grpcWeb.Metadata
  ): Promise<smartcore_bos_motionsensor_v1_motion_sensor_pb.MotionDetection>;

  pullMotionDetections(
    request: smartcore_bos_motionsensor_v1_motion_sensor_pb.PullMotionDetectionRequest,
    metadata?: grpcWeb.Metadata
  ): grpcWeb.ClientReadableStream<smartcore_bos_motionsensor_v1_motion_sensor_pb.PullMotionDetectionResponse>;

}

export class MotionSensorInfoPromiseClient {
  constructor (hostname: string,
               credentials?: null | { [index: string]: string; },
               options?: null | { [index: string]: any; });

  describeMotionDetection(
    request: smartcore_bos_motionsensor_v1_motion_sensor_pb.DescribeMotionDetectionRequest,
    metadata?: grpcWeb.Metadata
  ): Promise<smartcore_bos_motionsensor_v1_motion_sensor_pb.MotionDetectionSupport>;

}

