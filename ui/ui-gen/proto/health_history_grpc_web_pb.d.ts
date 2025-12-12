import * as grpcWeb from 'grpc-web';

import * as health_history_pb from './health_history_pb'; // proto import: "health_history.proto"


export class HealthHistoryClient {
  constructor (hostname: string,
               credentials?: null | { [index: string]: string; },
               options?: null | { [index: string]: any; });

  listHealthCheckHistory(
    request: health_history_pb.ListHealthCheckHistoryRequest,
    metadata: grpcWeb.Metadata | undefined,
    callback: (err: grpcWeb.RpcError,
               response: health_history_pb.ListHealthCheckHistoryResponse) => void
  ): grpcWeb.ClientReadableStream<health_history_pb.ListHealthCheckHistoryResponse>;

}

export class HealthHistoryPromiseClient {
  constructor (hostname: string,
               credentials?: null | { [index: string]: string; },
               options?: null | { [index: string]: any; });

  listHealthCheckHistory(
    request: health_history_pb.ListHealthCheckHistoryRequest,
    metadata?: grpcWeb.Metadata
  ): Promise<health_history_pb.ListHealthCheckHistoryResponse>;

}

