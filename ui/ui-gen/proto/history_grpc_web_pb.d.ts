import * as grpcWeb from 'grpc-web';

import * as history_pb from './history_pb'; // proto import: "history.proto"


export class HistoryAdminApiClient {
  constructor (hostname: string,
               credentials?: null | { [index: string]: string; },
               options?: null | { [index: string]: any; });

  createHistoryRecord(
    request: history_pb.CreateHistoryRecordRequest,
    metadata: grpcWeb.Metadata | undefined,
    callback: (err: grpcWeb.RpcError,
               response: history_pb.HistoryRecord) => void
  ): grpcWeb.ClientReadableStream<history_pb.HistoryRecord>;

  listHistoryRecords(
    request: history_pb.ListHistoryRecordsRequest,
    metadata: grpcWeb.Metadata | undefined,
    callback: (err: grpcWeb.RpcError,
               response: history_pb.ListHistoryRecordsResponse) => void
  ): grpcWeb.ClientReadableStream<history_pb.ListHistoryRecordsResponse>;

}

export class HistoryAdminApiPromiseClient {
  constructor (hostname: string,
               credentials?: null | { [index: string]: string; },
               options?: null | { [index: string]: any; });

  createHistoryRecord(
    request: history_pb.CreateHistoryRecordRequest,
    metadata?: grpcWeb.Metadata
  ): Promise<history_pb.HistoryRecord>;

  listHistoryRecords(
    request: history_pb.ListHistoryRecordsRequest,
    metadata?: grpcWeb.Metadata
  ): Promise<history_pb.ListHistoryRecordsResponse>;

}

