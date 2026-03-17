import * as grpcWeb from 'grpc-web';

import * as traits_count_pb from '../traits/count_pb'; // proto import: "traits/count.proto"


export class CountApiClient {
  constructor (hostname: string,
               credentials?: null | { [index: string]: string; },
               options?: null | { [index: string]: any; });

  getCount(
    request: traits_count_pb.GetCountRequest,
    metadata: grpcWeb.Metadata | undefined,
    callback: (err: grpcWeb.RpcError,
               response: traits_count_pb.Count) => void
  ): grpcWeb.ClientReadableStream<traits_count_pb.Count>;

  resetCount(
    request: traits_count_pb.ResetCountRequest,
    metadata: grpcWeb.Metadata | undefined,
    callback: (err: grpcWeb.RpcError,
               response: traits_count_pb.Count) => void
  ): grpcWeb.ClientReadableStream<traits_count_pb.Count>;

  updateCount(
    request: traits_count_pb.UpdateCountRequest,
    metadata: grpcWeb.Metadata | undefined,
    callback: (err: grpcWeb.RpcError,
               response: traits_count_pb.Count) => void
  ): grpcWeb.ClientReadableStream<traits_count_pb.Count>;

  pullCounts(
    request: traits_count_pb.PullCountsRequest,
    metadata?: grpcWeb.Metadata
  ): grpcWeb.ClientReadableStream<traits_count_pb.PullCountsResponse>;

}

export class CountInfoClient {
  constructor (hostname: string,
               credentials?: null | { [index: string]: string; },
               options?: null | { [index: string]: any; });

  describeCount(
    request: traits_count_pb.DescribeCountRequest,
    metadata: grpcWeb.Metadata | undefined,
    callback: (err: grpcWeb.RpcError,
               response: traits_count_pb.CountSupport) => void
  ): grpcWeb.ClientReadableStream<traits_count_pb.CountSupport>;

}

export class CountApiPromiseClient {
  constructor (hostname: string,
               credentials?: null | { [index: string]: string; },
               options?: null | { [index: string]: any; });

  getCount(
    request: traits_count_pb.GetCountRequest,
    metadata?: grpcWeb.Metadata
  ): Promise<traits_count_pb.Count>;

  resetCount(
    request: traits_count_pb.ResetCountRequest,
    metadata?: grpcWeb.Metadata
  ): Promise<traits_count_pb.Count>;

  updateCount(
    request: traits_count_pb.UpdateCountRequest,
    metadata?: grpcWeb.Metadata
  ): Promise<traits_count_pb.Count>;

  pullCounts(
    request: traits_count_pb.PullCountsRequest,
    metadata?: grpcWeb.Metadata
  ): grpcWeb.ClientReadableStream<traits_count_pb.PullCountsResponse>;

}

export class CountInfoPromiseClient {
  constructor (hostname: string,
               credentials?: null | { [index: string]: string; },
               options?: null | { [index: string]: any; });

  describeCount(
    request: traits_count_pb.DescribeCountRequest,
    metadata?: grpcWeb.Metadata
  ): Promise<traits_count_pb.CountSupport>;

}

