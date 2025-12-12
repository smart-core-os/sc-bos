import * as grpcWeb from 'grpc-web';

import * as status_history_pb from './status_history_pb'; // proto import: "status_history.proto"


export class StatusHistoryClient {
  constructor (hostname: string,
               credentials?: null | { [index: string]: string; },
               options?: null | { [index: string]: any; });

  listCurrentStatusHistory(
    request: status_history_pb.ListCurrentStatusHistoryRequest,
    metadata: grpcWeb.Metadata | undefined,
    callback: (err: grpcWeb.RpcError,
               response: status_history_pb.ListCurrentStatusHistoryResponse) => void
  ): grpcWeb.ClientReadableStream<status_history_pb.ListCurrentStatusHistoryResponse>;

}

export class StatusHistoryPromiseClient {
  constructor (hostname: string,
               credentials?: null | { [index: string]: string; },
               options?: null | { [index: string]: any; });

  listCurrentStatusHistory(
    request: status_history_pb.ListCurrentStatusHistoryRequest,
    metadata?: grpcWeb.Metadata
  ): Promise<status_history_pb.ListCurrentStatusHistoryResponse>;

}

