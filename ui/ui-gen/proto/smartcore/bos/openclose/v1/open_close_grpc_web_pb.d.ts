import * as grpcWeb from 'grpc-web';

import * as smartcore_bos_openclose_v1_open_close_pb from '../../../../smartcore/bos/openclose/v1/open_close_pb'; // proto import: "smartcore/bos/openclose/v1/open_close.proto"


export class OpenCloseApiClient {
  constructor (hostname: string,
               credentials?: null | { [index: string]: string; },
               options?: null | { [index: string]: any; });

  getPositions(
    request: smartcore_bos_openclose_v1_open_close_pb.GetOpenClosePositionsRequest,
    metadata: grpcWeb.Metadata | undefined,
    callback: (err: grpcWeb.RpcError,
               response: smartcore_bos_openclose_v1_open_close_pb.OpenClosePositions) => void
  ): grpcWeb.ClientReadableStream<smartcore_bos_openclose_v1_open_close_pb.OpenClosePositions>;

  updatePositions(
    request: smartcore_bos_openclose_v1_open_close_pb.UpdateOpenClosePositionsRequest,
    metadata: grpcWeb.Metadata | undefined,
    callback: (err: grpcWeb.RpcError,
               response: smartcore_bos_openclose_v1_open_close_pb.OpenClosePositions) => void
  ): grpcWeb.ClientReadableStream<smartcore_bos_openclose_v1_open_close_pb.OpenClosePositions>;

  stop(
    request: smartcore_bos_openclose_v1_open_close_pb.StopOpenCloseRequest,
    metadata: grpcWeb.Metadata | undefined,
    callback: (err: grpcWeb.RpcError,
               response: smartcore_bos_openclose_v1_open_close_pb.OpenClosePositions) => void
  ): grpcWeb.ClientReadableStream<smartcore_bos_openclose_v1_open_close_pb.OpenClosePositions>;

  pullPositions(
    request: smartcore_bos_openclose_v1_open_close_pb.PullOpenClosePositionsRequest,
    metadata?: grpcWeb.Metadata
  ): grpcWeb.ClientReadableStream<smartcore_bos_openclose_v1_open_close_pb.PullOpenClosePositionsResponse>;

}

export class OpenCloseInfoClient {
  constructor (hostname: string,
               credentials?: null | { [index: string]: string; },
               options?: null | { [index: string]: any; });

  describePositions(
    request: smartcore_bos_openclose_v1_open_close_pb.DescribePositionsRequest,
    metadata: grpcWeb.Metadata | undefined,
    callback: (err: grpcWeb.RpcError,
               response: smartcore_bos_openclose_v1_open_close_pb.PositionsSupport) => void
  ): grpcWeb.ClientReadableStream<smartcore_bos_openclose_v1_open_close_pb.PositionsSupport>;

}

export class OpenCloseApiPromiseClient {
  constructor (hostname: string,
               credentials?: null | { [index: string]: string; },
               options?: null | { [index: string]: any; });

  getPositions(
    request: smartcore_bos_openclose_v1_open_close_pb.GetOpenClosePositionsRequest,
    metadata?: grpcWeb.Metadata
  ): Promise<smartcore_bos_openclose_v1_open_close_pb.OpenClosePositions>;

  updatePositions(
    request: smartcore_bos_openclose_v1_open_close_pb.UpdateOpenClosePositionsRequest,
    metadata?: grpcWeb.Metadata
  ): Promise<smartcore_bos_openclose_v1_open_close_pb.OpenClosePositions>;

  stop(
    request: smartcore_bos_openclose_v1_open_close_pb.StopOpenCloseRequest,
    metadata?: grpcWeb.Metadata
  ): Promise<smartcore_bos_openclose_v1_open_close_pb.OpenClosePositions>;

  pullPositions(
    request: smartcore_bos_openclose_v1_open_close_pb.PullOpenClosePositionsRequest,
    metadata?: grpcWeb.Metadata
  ): grpcWeb.ClientReadableStream<smartcore_bos_openclose_v1_open_close_pb.PullOpenClosePositionsResponse>;

}

export class OpenCloseInfoPromiseClient {
  constructor (hostname: string,
               credentials?: null | { [index: string]: string; },
               options?: null | { [index: string]: any; });

  describePositions(
    request: smartcore_bos_openclose_v1_open_close_pb.DescribePositionsRequest,
    metadata?: grpcWeb.Metadata
  ): Promise<smartcore_bos_openclose_v1_open_close_pb.PositionsSupport>;

}

