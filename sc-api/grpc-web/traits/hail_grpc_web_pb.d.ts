import * as grpcWeb from 'grpc-web';

import * as traits_hail_pb from '../traits/hail_pb'; // proto import: "traits/hail.proto"


export class HailApiClient {
  constructor (hostname: string,
               credentials?: null | { [index: string]: string; },
               options?: null | { [index: string]: any; });

  createHail(
    request: traits_hail_pb.CreateHailRequest,
    metadata: grpcWeb.Metadata | undefined,
    callback: (err: grpcWeb.RpcError,
               response: traits_hail_pb.Hail) => void
  ): grpcWeb.ClientReadableStream<traits_hail_pb.Hail>;

  getHail(
    request: traits_hail_pb.GetHailRequest,
    metadata: grpcWeb.Metadata | undefined,
    callback: (err: grpcWeb.RpcError,
               response: traits_hail_pb.Hail) => void
  ): grpcWeb.ClientReadableStream<traits_hail_pb.Hail>;

  updateHail(
    request: traits_hail_pb.UpdateHailRequest,
    metadata: grpcWeb.Metadata | undefined,
    callback: (err: grpcWeb.RpcError,
               response: traits_hail_pb.Hail) => void
  ): grpcWeb.ClientReadableStream<traits_hail_pb.Hail>;

  deleteHail(
    request: traits_hail_pb.DeleteHailRequest,
    metadata: grpcWeb.Metadata | undefined,
    callback: (err: grpcWeb.RpcError,
               response: traits_hail_pb.DeleteHailResponse) => void
  ): grpcWeb.ClientReadableStream<traits_hail_pb.DeleteHailResponse>;

  pullHail(
    request: traits_hail_pb.PullHailRequest,
    metadata?: grpcWeb.Metadata
  ): grpcWeb.ClientReadableStream<traits_hail_pb.PullHailResponse>;

  listHails(
    request: traits_hail_pb.ListHailsRequest,
    metadata: grpcWeb.Metadata | undefined,
    callback: (err: grpcWeb.RpcError,
               response: traits_hail_pb.ListHailsResponse) => void
  ): grpcWeb.ClientReadableStream<traits_hail_pb.ListHailsResponse>;

  pullHails(
    request: traits_hail_pb.PullHailsRequest,
    metadata?: grpcWeb.Metadata
  ): grpcWeb.ClientReadableStream<traits_hail_pb.PullHailsResponse>;

}

export class HailInfoClient {
  constructor (hostname: string,
               credentials?: null | { [index: string]: string; },
               options?: null | { [index: string]: any; });

  describeHail(
    request: traits_hail_pb.DescribeHailRequest,
    metadata: grpcWeb.Metadata | undefined,
    callback: (err: grpcWeb.RpcError,
               response: traits_hail_pb.HailSupport) => void
  ): grpcWeb.ClientReadableStream<traits_hail_pb.HailSupport>;

}

export class HailApiPromiseClient {
  constructor (hostname: string,
               credentials?: null | { [index: string]: string; },
               options?: null | { [index: string]: any; });

  createHail(
    request: traits_hail_pb.CreateHailRequest,
    metadata?: grpcWeb.Metadata
  ): Promise<traits_hail_pb.Hail>;

  getHail(
    request: traits_hail_pb.GetHailRequest,
    metadata?: grpcWeb.Metadata
  ): Promise<traits_hail_pb.Hail>;

  updateHail(
    request: traits_hail_pb.UpdateHailRequest,
    metadata?: grpcWeb.Metadata
  ): Promise<traits_hail_pb.Hail>;

  deleteHail(
    request: traits_hail_pb.DeleteHailRequest,
    metadata?: grpcWeb.Metadata
  ): Promise<traits_hail_pb.DeleteHailResponse>;

  pullHail(
    request: traits_hail_pb.PullHailRequest,
    metadata?: grpcWeb.Metadata
  ): grpcWeb.ClientReadableStream<traits_hail_pb.PullHailResponse>;

  listHails(
    request: traits_hail_pb.ListHailsRequest,
    metadata?: grpcWeb.Metadata
  ): Promise<traits_hail_pb.ListHailsResponse>;

  pullHails(
    request: traits_hail_pb.PullHailsRequest,
    metadata?: grpcWeb.Metadata
  ): grpcWeb.ClientReadableStream<traits_hail_pb.PullHailsResponse>;

}

export class HailInfoPromiseClient {
  constructor (hostname: string,
               credentials?: null | { [index: string]: string; },
               options?: null | { [index: string]: any; });

  describeHail(
    request: traits_hail_pb.DescribeHailRequest,
    metadata?: grpcWeb.Metadata
  ): Promise<traits_hail_pb.HailSupport>;

}

