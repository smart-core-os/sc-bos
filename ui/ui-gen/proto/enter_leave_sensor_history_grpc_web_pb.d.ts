import * as grpcWeb from 'grpc-web';

import * as enter_leave_sensor_history_pb from './enter_leave_sensor_history_pb'; // proto import: "enter_leave_sensor_history.proto"


export class EnterLeaveHistoryClient {
  constructor (hostname: string,
               credentials?: null | { [index: string]: string; },
               options?: null | { [index: string]: any; });

  listEnterLeaveSensorHistory(
    request: enter_leave_sensor_history_pb.ListEnterLeaveHistoryRequest,
    metadata: grpcWeb.Metadata | undefined,
    callback: (err: grpcWeb.RpcError,
               response: enter_leave_sensor_history_pb.ListEnterLeaveHistoryResponse) => void
  ): grpcWeb.ClientReadableStream<enter_leave_sensor_history_pb.ListEnterLeaveHistoryResponse>;

}

export class EnterLeaveHistoryPromiseClient {
  constructor (hostname: string,
               credentials?: null | { [index: string]: string; },
               options?: null | { [index: string]: any; });

  listEnterLeaveSensorHistory(
    request: enter_leave_sensor_history_pb.ListEnterLeaveHistoryRequest,
    metadata?: grpcWeb.Metadata
  ): Promise<enter_leave_sensor_history_pb.ListEnterLeaveHistoryResponse>;

}

