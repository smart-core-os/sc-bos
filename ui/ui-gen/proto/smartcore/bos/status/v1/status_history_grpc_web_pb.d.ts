import * as grpcWeb from 'grpc-web';

import * as smartcore_bos_status_v1_status_history_pb from '../../../../smartcore/bos/status/v1/status_history_pb'; // proto import: "smartcore/bos/status/v1/status_history.proto"


export class StatusHistoryClient {
  constructor (hostname: string,
               credentials?: null | { [index: string]: string; },
               options?: null | { [index: string]: any; });

  listCurrentStatusHistory(
    request: smartcore_bos_status_v1_status_history_pb.ListCurrentStatusHistoryRequest,
    metadata: grpcWeb.Metadata | undefined,
    callback: (err: grpcWeb.RpcError,
               response: smartcore_bos_status_v1_status_history_pb.ListCurrentStatusHistoryResponse) => void
  ): grpcWeb.ClientReadableStream<smartcore_bos_status_v1_status_history_pb.ListCurrentStatusHistoryResponse>;

}

export class StatusHistoryPromiseClient {
  constructor (hostname: string,
               credentials?: null | { [index: string]: string; },
               options?: null | { [index: string]: any; });

  listCurrentStatusHistory(
    request: smartcore_bos_status_v1_status_history_pb.ListCurrentStatusHistoryRequest,
    metadata?: grpcWeb.Metadata
  ): Promise<smartcore_bos_status_v1_status_history_pb.ListCurrentStatusHistoryResponse>;

}

