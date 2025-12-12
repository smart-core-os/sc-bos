import * as grpcWeb from 'grpc-web';

import * as transport_history_pb from './transport_history_pb'; // proto import: "transport_history.proto"


export class TransportHistoryClient {
  constructor (hostname: string,
               credentials?: null | { [index: string]: string; },
               options?: null | { [index: string]: any; });

  listTransportHistory(
    request: transport_history_pb.ListTransportHistoryRequest,
    metadata: grpcWeb.Metadata | undefined,
    callback: (err: grpcWeb.RpcError,
               response: transport_history_pb.ListTransportHistoryResponse) => void
  ): grpcWeb.ClientReadableStream<transport_history_pb.ListTransportHistoryResponse>;

}

export class TransportHistoryPromiseClient {
  constructor (hostname: string,
               credentials?: null | { [index: string]: string; },
               options?: null | { [index: string]: any; });

  listTransportHistory(
    request: transport_history_pb.ListTransportHistoryRequest,
    metadata?: grpcWeb.Metadata
  ): Promise<transport_history_pb.ListTransportHistoryResponse>;

}

