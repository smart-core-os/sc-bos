import * as grpcWeb from 'grpc-web';

import * as smartcore_bos_allocation_v1_allocation_pb from '../../../../smartcore/bos/allocation/v1/allocation_pb'; // proto import: "smartcore/bos/allocation/v1/allocation.proto"


export class AllocationApiClient {
  constructor (hostname: string,
               credentials?: null | { [index: string]: string; },
               options?: null | { [index: string]: any; });

  getAllocation(
    request: smartcore_bos_allocation_v1_allocation_pb.GetAllocationRequest,
    metadata: grpcWeb.Metadata | undefined,
    callback: (err: grpcWeb.RpcError,
               response: smartcore_bos_allocation_v1_allocation_pb.Allocation) => void
  ): grpcWeb.ClientReadableStream<smartcore_bos_allocation_v1_allocation_pb.Allocation>;

  updateAllocation(
    request: smartcore_bos_allocation_v1_allocation_pb.UpdateAllocationRequest,
    metadata: grpcWeb.Metadata | undefined,
    callback: (err: grpcWeb.RpcError,
               response: smartcore_bos_allocation_v1_allocation_pb.Allocation) => void
  ): grpcWeb.ClientReadableStream<smartcore_bos_allocation_v1_allocation_pb.Allocation>;

  pullAllocation(
    request: smartcore_bos_allocation_v1_allocation_pb.PullAllocationRequest,
    metadata?: grpcWeb.Metadata
  ): grpcWeb.ClientReadableStream<smartcore_bos_allocation_v1_allocation_pb.PullAllocationResponse>;

}

export class AllocationApiPromiseClient {
  constructor (hostname: string,
               credentials?: null | { [index: string]: string; },
               options?: null | { [index: string]: any; });

  getAllocation(
    request: smartcore_bos_allocation_v1_allocation_pb.GetAllocationRequest,
    metadata?: grpcWeb.Metadata
  ): Promise<smartcore_bos_allocation_v1_allocation_pb.Allocation>;

  updateAllocation(
    request: smartcore_bos_allocation_v1_allocation_pb.UpdateAllocationRequest,
    metadata?: grpcWeb.Metadata
  ): Promise<smartcore_bos_allocation_v1_allocation_pb.Allocation>;

  pullAllocation(
    request: smartcore_bos_allocation_v1_allocation_pb.PullAllocationRequest,
    metadata?: grpcWeb.Metadata
  ): grpcWeb.ClientReadableStream<smartcore_bos_allocation_v1_allocation_pb.PullAllocationResponse>;

}

