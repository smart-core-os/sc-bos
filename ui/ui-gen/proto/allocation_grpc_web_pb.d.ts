import * as grpcWeb from 'grpc-web';

import * as allocation_pb from './allocation_pb'; // proto import: "allocation.proto"


export class AllocationApiClient {
  constructor (hostname: string,
               credentials?: null | { [index: string]: string; },
               options?: null | { [index: string]: any; });

  getAllocation(
    request: allocation_pb.GetAllocationRequest,
    metadata: grpcWeb.Metadata | undefined,
    callback: (err: grpcWeb.RpcError,
               response: allocation_pb.Allocation) => void
  ): grpcWeb.ClientReadableStream<allocation_pb.Allocation>;

  updateAllocation(
    request: allocation_pb.UpdateAllocationRequest,
    metadata: grpcWeb.Metadata | undefined,
    callback: (err: grpcWeb.RpcError,
               response: allocation_pb.Allocation) => void
  ): grpcWeb.ClientReadableStream<allocation_pb.Allocation>;

  pullAllocations(
    request: allocation_pb.PullAllocationsRequest,
    metadata?: grpcWeb.Metadata
  ): grpcWeb.ClientReadableStream<allocation_pb.PullAllocationsResponse>;

  listAllocatableResources(
    request: allocation_pb.ListAllocatableResourcesRequest,
    metadata: grpcWeb.Metadata | undefined,
    callback: (err: grpcWeb.RpcError,
               response: allocation_pb.ListAllocatableResourcesResponse) => void
  ): grpcWeb.ClientReadableStream<allocation_pb.ListAllocatableResourcesResponse>;

}

export class AllocationHistoryClient {
  constructor (hostname: string,
               credentials?: null | { [index: string]: string; },
               options?: null | { [index: string]: any; });

  listAllocationHistory(
    request: allocation_pb.ListAllocationHistoryRequest,
    metadata: grpcWeb.Metadata | undefined,
    callback: (err: grpcWeb.RpcError,
               response: allocation_pb.ListAllocationHistoryResponse) => void
  ): grpcWeb.ClientReadableStream<allocation_pb.ListAllocationHistoryResponse>;

}

export class AllocationApiPromiseClient {
  constructor (hostname: string,
               credentials?: null | { [index: string]: string; },
               options?: null | { [index: string]: any; });

  getAllocation(
    request: allocation_pb.GetAllocationRequest,
    metadata?: grpcWeb.Metadata
  ): Promise<allocation_pb.Allocation>;

  updateAllocation(
    request: allocation_pb.UpdateAllocationRequest,
    metadata?: grpcWeb.Metadata
  ): Promise<allocation_pb.Allocation>;

  pullAllocations(
    request: allocation_pb.PullAllocationsRequest,
    metadata?: grpcWeb.Metadata
  ): grpcWeb.ClientReadableStream<allocation_pb.PullAllocationsResponse>;

  listAllocatableResources(
    request: allocation_pb.ListAllocatableResourcesRequest,
    metadata?: grpcWeb.Metadata
  ): Promise<allocation_pb.ListAllocatableResourcesResponse>;

}

export class AllocationHistoryPromiseClient {
  constructor (hostname: string,
               credentials?: null | { [index: string]: string; },
               options?: null | { [index: string]: any; });

  listAllocationHistory(
    request: allocation_pb.ListAllocationHistoryRequest,
    metadata?: grpcWeb.Metadata
  ): Promise<allocation_pb.ListAllocationHistoryResponse>;

}

