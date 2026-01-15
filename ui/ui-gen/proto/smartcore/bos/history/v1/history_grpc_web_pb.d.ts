import * as grpcWeb from 'grpc-web';

import * as smartcore_bos_history_v1_history_pb from '../../../../smartcore/bos/history/v1/history_pb'; // proto import: "smartcore/bos/history/v1/history.proto"


export class HistoryAdminApiClient {
  constructor (hostname: string,
               credentials?: null | { [index: string]: string; },
               options?: null | { [index: string]: any; });

  createHistoryRecord(
    request: smartcore_bos_history_v1_history_pb.CreateHistoryRecordRequest,
    metadata: grpcWeb.Metadata | undefined,
    callback: (err: grpcWeb.RpcError,
               response: smartcore_bos_history_v1_history_pb.HistoryRecord) => void
  ): grpcWeb.ClientReadableStream<smartcore_bos_history_v1_history_pb.HistoryRecord>;

  listHistoryRecords(
    request: smartcore_bos_history_v1_history_pb.ListHistoryRecordsRequest,
    metadata: grpcWeb.Metadata | undefined,
    callback: (err: grpcWeb.RpcError,
               response: smartcore_bos_history_v1_history_pb.ListHistoryRecordsResponse) => void
  ): grpcWeb.ClientReadableStream<smartcore_bos_history_v1_history_pb.ListHistoryRecordsResponse>;

}

export class HistoryAdminApiPromiseClient {
  constructor (hostname: string,
               credentials?: null | { [index: string]: string; },
               options?: null | { [index: string]: any; });

  createHistoryRecord(
    request: smartcore_bos_history_v1_history_pb.CreateHistoryRecordRequest,
    metadata?: grpcWeb.Metadata
  ): Promise<smartcore_bos_history_v1_history_pb.HistoryRecord>;

  listHistoryRecords(
    request: smartcore_bos_history_v1_history_pb.ListHistoryRecordsRequest,
    metadata?: grpcWeb.Metadata
  ): Promise<smartcore_bos_history_v1_history_pb.ListHistoryRecordsResponse>;

}

