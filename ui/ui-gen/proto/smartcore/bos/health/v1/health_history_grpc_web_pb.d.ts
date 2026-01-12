import * as grpcWeb from 'grpc-web';

import * as smartcore_bos_health_v1_health_history_pb from '../../../../smartcore/bos/health/v1/health_history_pb'; // proto import: "smartcore/bos/health/v1/health_history.proto"


export class HealthHistoryClient {
  constructor (hostname: string,
               credentials?: null | { [index: string]: string; },
               options?: null | { [index: string]: any; });

  listHealthCheckHistory(
    request: smartcore_bos_health_v1_health_history_pb.ListHealthCheckHistoryRequest,
    metadata: grpcWeb.Metadata | undefined,
    callback: (err: grpcWeb.RpcError,
               response: smartcore_bos_health_v1_health_history_pb.ListHealthCheckHistoryResponse) => void
  ): grpcWeb.ClientReadableStream<smartcore_bos_health_v1_health_history_pb.ListHealthCheckHistoryResponse>;

}

export class HealthHistoryPromiseClient {
  constructor (hostname: string,
               credentials?: null | { [index: string]: string; },
               options?: null | { [index: string]: any; });

  listHealthCheckHistory(
    request: smartcore_bos_health_v1_health_history_pb.ListHealthCheckHistoryRequest,
    metadata?: grpcWeb.Metadata
  ): Promise<smartcore_bos_health_v1_health_history_pb.ListHealthCheckHistoryResponse>;

}

