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

  pullAllocation(
    request: allocation_pb.PullAllocationRequest,
    metadata?: grpcWeb.Metadata
  ): grpcWeb.ClientReadableStream<allocation_pb.PullAllocationResponse>;

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

  pullAllocation(
    request: allocation_pb.PullAllocationRequest,
    metadata?: grpcWeb.Metadata
  ): grpcWeb.ClientReadableStream<allocation_pb.PullAllocationResponse>;

}

