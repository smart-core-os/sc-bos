import * as grpcWeb from 'grpc-web';

import * as smartcore_bos_allocation_v1_allocation_history_pb from '../../../../smartcore/bos/allocation/v1/allocation_history_pb'; // proto import: "smartcore/bos/allocation/v1/allocation_history.proto"


export class AllocationHistoryClient {
  constructor (hostname: string,
               credentials?: null | { [index: string]: string; },
               options?: null | { [index: string]: any; });

  listAllocationHistory(
    request: smartcore_bos_allocation_v1_allocation_history_pb.ListAllocationHistoryRequest,
    metadata: grpcWeb.Metadata | undefined,
    callback: (err: grpcWeb.RpcError,
               response: smartcore_bos_allocation_v1_allocation_history_pb.ListAllocationHistoryResponse) => void
  ): grpcWeb.ClientReadableStream<smartcore_bos_allocation_v1_allocation_history_pb.ListAllocationHistoryResponse>;

}

export class AllocationHistoryPromiseClient {
  constructor (hostname: string,
               credentials?: null | { [index: string]: string; },
               options?: null | { [index: string]: any; });

  listAllocationHistory(
    request: smartcore_bos_allocation_v1_allocation_history_pb.ListAllocationHistoryRequest,
    metadata?: grpcWeb.Metadata
  ): Promise<smartcore_bos_allocation_v1_allocation_history_pb.ListAllocationHistoryResponse>;

}

