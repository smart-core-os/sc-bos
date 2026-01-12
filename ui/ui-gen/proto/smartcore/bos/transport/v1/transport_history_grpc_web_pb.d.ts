import * as grpcWeb from 'grpc-web';

import * as smartcore_bos_transport_v1_transport_history_pb from '../../../../smartcore/bos/transport/v1/transport_history_pb'; // proto import: "smartcore/bos/transport/v1/transport_history.proto"


export class TransportHistoryClient {
  constructor (hostname: string,
               credentials?: null | { [index: string]: string; },
               options?: null | { [index: string]: any; });

  listTransportHistory(
    request: smartcore_bos_transport_v1_transport_history_pb.ListTransportHistoryRequest,
    metadata: grpcWeb.Metadata | undefined,
    callback: (err: grpcWeb.RpcError,
               response: smartcore_bos_transport_v1_transport_history_pb.ListTransportHistoryResponse) => void
  ): grpcWeb.ClientReadableStream<smartcore_bos_transport_v1_transport_history_pb.ListTransportHistoryResponse>;

}

export class TransportHistoryPromiseClient {
  constructor (hostname: string,
               credentials?: null | { [index: string]: string; },
               options?: null | { [index: string]: any; });

  listTransportHistory(
    request: smartcore_bos_transport_v1_transport_history_pb.ListTransportHistoryRequest,
    metadata?: grpcWeb.Metadata
  ): Promise<smartcore_bos_transport_v1_transport_history_pb.ListTransportHistoryResponse>;

}

