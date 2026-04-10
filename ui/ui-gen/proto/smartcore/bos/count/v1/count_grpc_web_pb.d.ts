import * as grpcWeb from 'grpc-web';

import * as smartcore_bos_count_v1_count_pb from '../../../../smartcore/bos/count/v1/count_pb'; // proto import: "smartcore/bos/count/v1/count.proto"


export class CountApiClient {
  constructor (hostname: string,
               credentials?: null | { [index: string]: string; },
               options?: null | { [index: string]: any; });

  getCount(
    request: smartcore_bos_count_v1_count_pb.GetCountRequest,
    metadata: grpcWeb.Metadata | undefined,
    callback: (err: grpcWeb.RpcError,
               response: smartcore_bos_count_v1_count_pb.Count) => void
  ): grpcWeb.ClientReadableStream<smartcore_bos_count_v1_count_pb.Count>;

  resetCount(
    request: smartcore_bos_count_v1_count_pb.ResetCountRequest,
    metadata: grpcWeb.Metadata | undefined,
    callback: (err: grpcWeb.RpcError,
               response: smartcore_bos_count_v1_count_pb.Count) => void
  ): grpcWeb.ClientReadableStream<smartcore_bos_count_v1_count_pb.Count>;

  updateCount(
    request: smartcore_bos_count_v1_count_pb.UpdateCountRequest,
    metadata: grpcWeb.Metadata | undefined,
    callback: (err: grpcWeb.RpcError,
               response: smartcore_bos_count_v1_count_pb.Count) => void
  ): grpcWeb.ClientReadableStream<smartcore_bos_count_v1_count_pb.Count>;

  pullCounts(
    request: smartcore_bos_count_v1_count_pb.PullCountsRequest,
    metadata?: grpcWeb.Metadata
  ): grpcWeb.ClientReadableStream<smartcore_bos_count_v1_count_pb.PullCountsResponse>;

}

export class CountInfoClient {
  constructor (hostname: string,
               credentials?: null | { [index: string]: string; },
               options?: null | { [index: string]: any; });

  describeCount(
    request: smartcore_bos_count_v1_count_pb.DescribeCountRequest,
    metadata: grpcWeb.Metadata | undefined,
    callback: (err: grpcWeb.RpcError,
               response: smartcore_bos_count_v1_count_pb.CountSupport) => void
  ): grpcWeb.ClientReadableStream<smartcore_bos_count_v1_count_pb.CountSupport>;

}

export class CountApiPromiseClient {
  constructor (hostname: string,
               credentials?: null | { [index: string]: string; },
               options?: null | { [index: string]: any; });

  getCount(
    request: smartcore_bos_count_v1_count_pb.GetCountRequest,
    metadata?: grpcWeb.Metadata
  ): Promise<smartcore_bos_count_v1_count_pb.Count>;

  resetCount(
    request: smartcore_bos_count_v1_count_pb.ResetCountRequest,
    metadata?: grpcWeb.Metadata
  ): Promise<smartcore_bos_count_v1_count_pb.Count>;

  updateCount(
    request: smartcore_bos_count_v1_count_pb.UpdateCountRequest,
    metadata?: grpcWeb.Metadata
  ): Promise<smartcore_bos_count_v1_count_pb.Count>;

  pullCounts(
    request: smartcore_bos_count_v1_count_pb.PullCountsRequest,
    metadata?: grpcWeb.Metadata
  ): grpcWeb.ClientReadableStream<smartcore_bos_count_v1_count_pb.PullCountsResponse>;

}

export class CountInfoPromiseClient {
  constructor (hostname: string,
               credentials?: null | { [index: string]: string; },
               options?: null | { [index: string]: any; });

  describeCount(
    request: smartcore_bos_count_v1_count_pb.DescribeCountRequest,
    metadata?: grpcWeb.Metadata
  ): Promise<smartcore_bos_count_v1_count_pb.CountSupport>;

}

