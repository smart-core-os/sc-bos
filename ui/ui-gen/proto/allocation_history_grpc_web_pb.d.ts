import * as grpcWeb from 'grpc-web';

import * as allocation_history_pb from './allocation_history_pb'; // proto import: "allocation_history.proto"


export class AllocationHistoryClient {
  constructor (hostname: string,
               credentials?: null | { [index: string]: string; },
               options?: null | { [index: string]: any; });

  listAllocationHistory(
    request: allocation_history_pb.ListAllocationHistoryRequest,
    metadata: grpcWeb.Metadata | undefined,
    callback: (err: grpcWeb.RpcError,
               response: allocation_history_pb.ListAllocationHistoryResponse) => void
  ): grpcWeb.ClientReadableStream<allocation_history_pb.ListAllocationHistoryResponse>;

}

export class AllocationHistoryPromiseClient {
  constructor (hostname: string,
               credentials?: null | { [index: string]: string; },
               options?: null | { [index: string]: any; });

  listAllocationHistory(
    request: allocation_history_pb.ListAllocationHistoryRequest,
    metadata?: grpcWeb.Metadata
  ): Promise<allocation_history_pb.ListAllocationHistoryResponse>;

}

