import * as grpcWeb from 'grpc-web';

import * as smartcore_bos_enterleavesensor_v1_enter_leave_sensor_pb from '../../../../smartcore/bos/enterleavesensor/v1/enter_leave_sensor_pb'; // proto import: "smartcore/bos/enterleavesensor/v1/enter_leave_sensor.proto"


export class EnterLeaveSensorApiClient {
  constructor (hostname: string,
               credentials?: null | { [index: string]: string; },
               options?: null | { [index: string]: any; });

  pullEnterLeaveEvents(
    request: smartcore_bos_enterleavesensor_v1_enter_leave_sensor_pb.PullEnterLeaveEventsRequest,
    metadata?: grpcWeb.Metadata
  ): grpcWeb.ClientReadableStream<smartcore_bos_enterleavesensor_v1_enter_leave_sensor_pb.PullEnterLeaveEventsResponse>;

  getEnterLeaveEvent(
    request: smartcore_bos_enterleavesensor_v1_enter_leave_sensor_pb.GetEnterLeaveEventRequest,
    metadata: grpcWeb.Metadata | undefined,
    callback: (err: grpcWeb.RpcError,
               response: smartcore_bos_enterleavesensor_v1_enter_leave_sensor_pb.EnterLeaveEvent) => void
  ): grpcWeb.ClientReadableStream<smartcore_bos_enterleavesensor_v1_enter_leave_sensor_pb.EnterLeaveEvent>;

  resetEnterLeaveTotals(
    request: smartcore_bos_enterleavesensor_v1_enter_leave_sensor_pb.ResetEnterLeaveTotalsRequest,
    metadata: grpcWeb.Metadata | undefined,
    callback: (err: grpcWeb.RpcError,
               response: smartcore_bos_enterleavesensor_v1_enter_leave_sensor_pb.ResetEnterLeaveTotalsResponse) => void
  ): grpcWeb.ClientReadableStream<smartcore_bos_enterleavesensor_v1_enter_leave_sensor_pb.ResetEnterLeaveTotalsResponse>;

}

export class EnterLeaveSensorInfoClient {
  constructor (hostname: string,
               credentials?: null | { [index: string]: string; },
               options?: null | { [index: string]: any; });

}

export class EnterLeaveSensorApiPromiseClient {
  constructor (hostname: string,
               credentials?: null | { [index: string]: string; },
               options?: null | { [index: string]: any; });

  pullEnterLeaveEvents(
    request: smartcore_bos_enterleavesensor_v1_enter_leave_sensor_pb.PullEnterLeaveEventsRequest,
    metadata?: grpcWeb.Metadata
  ): grpcWeb.ClientReadableStream<smartcore_bos_enterleavesensor_v1_enter_leave_sensor_pb.PullEnterLeaveEventsResponse>;

  getEnterLeaveEvent(
    request: smartcore_bos_enterleavesensor_v1_enter_leave_sensor_pb.GetEnterLeaveEventRequest,
    metadata?: grpcWeb.Metadata
  ): Promise<smartcore_bos_enterleavesensor_v1_enter_leave_sensor_pb.EnterLeaveEvent>;

  resetEnterLeaveTotals(
    request: smartcore_bos_enterleavesensor_v1_enter_leave_sensor_pb.ResetEnterLeaveTotalsRequest,
    metadata?: grpcWeb.Metadata
  ): Promise<smartcore_bos_enterleavesensor_v1_enter_leave_sensor_pb.ResetEnterLeaveTotalsResponse>;

}

export class EnterLeaveSensorInfoPromiseClient {
  constructor (hostname: string,
               credentials?: null | { [index: string]: string; },
               options?: null | { [index: string]: any; });

}

