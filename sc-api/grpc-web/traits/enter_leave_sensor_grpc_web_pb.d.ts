import * as grpcWeb from 'grpc-web';

import * as traits_enter_leave_sensor_pb from '../traits/enter_leave_sensor_pb'; // proto import: "traits/enter_leave_sensor.proto"


export class EnterLeaveSensorApiClient {
  constructor (hostname: string,
               credentials?: null | { [index: string]: string; },
               options?: null | { [index: string]: any; });

  pullEnterLeaveEvents(
    request: traits_enter_leave_sensor_pb.PullEnterLeaveEventsRequest,
    metadata?: grpcWeb.Metadata
  ): grpcWeb.ClientReadableStream<traits_enter_leave_sensor_pb.PullEnterLeaveEventsResponse>;

  getEnterLeaveEvent(
    request: traits_enter_leave_sensor_pb.GetEnterLeaveEventRequest,
    metadata: grpcWeb.Metadata | undefined,
    callback: (err: grpcWeb.RpcError,
               response: traits_enter_leave_sensor_pb.EnterLeaveEvent) => void
  ): grpcWeb.ClientReadableStream<traits_enter_leave_sensor_pb.EnterLeaveEvent>;

  resetEnterLeaveTotals(
    request: traits_enter_leave_sensor_pb.ResetEnterLeaveTotalsRequest,
    metadata: grpcWeb.Metadata | undefined,
    callback: (err: grpcWeb.RpcError,
               response: traits_enter_leave_sensor_pb.ResetEnterLeaveTotalsResponse) => void
  ): grpcWeb.ClientReadableStream<traits_enter_leave_sensor_pb.ResetEnterLeaveTotalsResponse>;

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
    request: traits_enter_leave_sensor_pb.PullEnterLeaveEventsRequest,
    metadata?: grpcWeb.Metadata
  ): grpcWeb.ClientReadableStream<traits_enter_leave_sensor_pb.PullEnterLeaveEventsResponse>;

  getEnterLeaveEvent(
    request: traits_enter_leave_sensor_pb.GetEnterLeaveEventRequest,
    metadata?: grpcWeb.Metadata
  ): Promise<traits_enter_leave_sensor_pb.EnterLeaveEvent>;

  resetEnterLeaveTotals(
    request: traits_enter_leave_sensor_pb.ResetEnterLeaveTotalsRequest,
    metadata?: grpcWeb.Metadata
  ): Promise<traits_enter_leave_sensor_pb.ResetEnterLeaveTotalsResponse>;

}

export class EnterLeaveSensorInfoPromiseClient {
  constructor (hostname: string,
               credentials?: null | { [index: string]: string; },
               options?: null | { [index: string]: any; });

}

