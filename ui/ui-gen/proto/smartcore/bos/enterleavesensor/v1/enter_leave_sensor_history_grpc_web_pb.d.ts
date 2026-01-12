import * as grpcWeb from 'grpc-web';

import * as smartcore_bos_enterleavesensor_v1_enter_leave_sensor_history_pb from '../../../../smartcore/bos/enterleavesensor/v1/enter_leave_sensor_history_pb'; // proto import: "smartcore/bos/enterleavesensor/v1/enter_leave_sensor_history.proto"


export class EnterLeaveSensorHistoryClient {
  constructor (hostname: string,
               credentials?: null | { [index: string]: string; },
               options?: null | { [index: string]: any; });

  listEnterLeaveSensorHistory(
    request: smartcore_bos_enterleavesensor_v1_enter_leave_sensor_history_pb.ListEnterLeaveHistoryRequest,
    metadata: grpcWeb.Metadata | undefined,
    callback: (err: grpcWeb.RpcError,
               response: smartcore_bos_enterleavesensor_v1_enter_leave_sensor_history_pb.ListEnterLeaveHistoryResponse) => void
  ): grpcWeb.ClientReadableStream<smartcore_bos_enterleavesensor_v1_enter_leave_sensor_history_pb.ListEnterLeaveHistoryResponse>;

}

export class EnterLeaveSensorHistoryPromiseClient {
  constructor (hostname: string,
               credentials?: null | { [index: string]: string; },
               options?: null | { [index: string]: any; });

  listEnterLeaveSensorHistory(
    request: smartcore_bos_enterleavesensor_v1_enter_leave_sensor_history_pb.ListEnterLeaveHistoryRequest,
    metadata?: grpcWeb.Metadata
  ): Promise<smartcore_bos_enterleavesensor_v1_enter_leave_sensor_history_pb.ListEnterLeaveHistoryResponse>;

}

