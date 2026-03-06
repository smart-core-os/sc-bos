import * as grpcWeb from 'grpc-web';

import * as traits_open_close_pb from '../traits/open_close_pb'; // proto import: "traits/open_close.proto"


export class OpenCloseApiClient {
  constructor (hostname: string,
               credentials?: null | { [index: string]: string; },
               options?: null | { [index: string]: any; });

  getPositions(
    request: traits_open_close_pb.GetOpenClosePositionsRequest,
    metadata: grpcWeb.Metadata | undefined,
    callback: (err: grpcWeb.RpcError,
               response: traits_open_close_pb.OpenClosePositions) => void
  ): grpcWeb.ClientReadableStream<traits_open_close_pb.OpenClosePositions>;

  updatePositions(
    request: traits_open_close_pb.UpdateOpenClosePositionsRequest,
    metadata: grpcWeb.Metadata | undefined,
    callback: (err: grpcWeb.RpcError,
               response: traits_open_close_pb.OpenClosePositions) => void
  ): grpcWeb.ClientReadableStream<traits_open_close_pb.OpenClosePositions>;

  stop(
    request: traits_open_close_pb.StopOpenCloseRequest,
    metadata: grpcWeb.Metadata | undefined,
    callback: (err: grpcWeb.RpcError,
               response: traits_open_close_pb.OpenClosePositions) => void
  ): grpcWeb.ClientReadableStream<traits_open_close_pb.OpenClosePositions>;

  pullPositions(
    request: traits_open_close_pb.PullOpenClosePositionsRequest,
    metadata?: grpcWeb.Metadata
  ): grpcWeb.ClientReadableStream<traits_open_close_pb.PullOpenClosePositionsResponse>;

}

export class OpenCloseInfoClient {
  constructor (hostname: string,
               credentials?: null | { [index: string]: string; },
               options?: null | { [index: string]: any; });

  describePositions(
    request: traits_open_close_pb.DescribePositionsRequest,
    metadata: grpcWeb.Metadata | undefined,
    callback: (err: grpcWeb.RpcError,
               response: traits_open_close_pb.PositionsSupport) => void
  ): grpcWeb.ClientReadableStream<traits_open_close_pb.PositionsSupport>;

}

export class OpenCloseApiPromiseClient {
  constructor (hostname: string,
               credentials?: null | { [index: string]: string; },
               options?: null | { [index: string]: any; });

  getPositions(
    request: traits_open_close_pb.GetOpenClosePositionsRequest,
    metadata?: grpcWeb.Metadata
  ): Promise<traits_open_close_pb.OpenClosePositions>;

  updatePositions(
    request: traits_open_close_pb.UpdateOpenClosePositionsRequest,
    metadata?: grpcWeb.Metadata
  ): Promise<traits_open_close_pb.OpenClosePositions>;

  stop(
    request: traits_open_close_pb.StopOpenCloseRequest,
    metadata?: grpcWeb.Metadata
  ): Promise<traits_open_close_pb.OpenClosePositions>;

  pullPositions(
    request: traits_open_close_pb.PullOpenClosePositionsRequest,
    metadata?: grpcWeb.Metadata
  ): grpcWeb.ClientReadableStream<traits_open_close_pb.PullOpenClosePositionsResponse>;

}

export class OpenCloseInfoPromiseClient {
  constructor (hostname: string,
               credentials?: null | { [index: string]: string; },
               options?: null | { [index: string]: any; });

  describePositions(
    request: traits_open_close_pb.DescribePositionsRequest,
    metadata?: grpcWeb.Metadata
  ): Promise<traits_open_close_pb.PositionsSupport>;

}

