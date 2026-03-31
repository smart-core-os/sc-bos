import * as grpcWeb from 'grpc-web';

import * as smartcore_bos_hail_v1_hail_pb from '../../../../smartcore/bos/hail/v1/hail_pb'; // proto import: "smartcore/bos/hail/v1/hail.proto"


export class HailApiClient {
  constructor (hostname: string,
               credentials?: null | { [index: string]: string; },
               options?: null | { [index: string]: any; });

  createHail(
    request: smartcore_bos_hail_v1_hail_pb.CreateHailRequest,
    metadata: grpcWeb.Metadata | undefined,
    callback: (err: grpcWeb.RpcError,
               response: smartcore_bos_hail_v1_hail_pb.Hail) => void
  ): grpcWeb.ClientReadableStream<smartcore_bos_hail_v1_hail_pb.Hail>;

  getHail(
    request: smartcore_bos_hail_v1_hail_pb.GetHailRequest,
    metadata: grpcWeb.Metadata | undefined,
    callback: (err: grpcWeb.RpcError,
               response: smartcore_bos_hail_v1_hail_pb.Hail) => void
  ): grpcWeb.ClientReadableStream<smartcore_bos_hail_v1_hail_pb.Hail>;

  updateHail(
    request: smartcore_bos_hail_v1_hail_pb.UpdateHailRequest,
    metadata: grpcWeb.Metadata | undefined,
    callback: (err: grpcWeb.RpcError,
               response: smartcore_bos_hail_v1_hail_pb.Hail) => void
  ): grpcWeb.ClientReadableStream<smartcore_bos_hail_v1_hail_pb.Hail>;

  deleteHail(
    request: smartcore_bos_hail_v1_hail_pb.DeleteHailRequest,
    metadata: grpcWeb.Metadata | undefined,
    callback: (err: grpcWeb.RpcError,
               response: smartcore_bos_hail_v1_hail_pb.DeleteHailResponse) => void
  ): grpcWeb.ClientReadableStream<smartcore_bos_hail_v1_hail_pb.DeleteHailResponse>;

  pullHail(
    request: smartcore_bos_hail_v1_hail_pb.PullHailRequest,
    metadata?: grpcWeb.Metadata
  ): grpcWeb.ClientReadableStream<smartcore_bos_hail_v1_hail_pb.PullHailResponse>;

  listHails(
    request: smartcore_bos_hail_v1_hail_pb.ListHailsRequest,
    metadata: grpcWeb.Metadata | undefined,
    callback: (err: grpcWeb.RpcError,
               response: smartcore_bos_hail_v1_hail_pb.ListHailsResponse) => void
  ): grpcWeb.ClientReadableStream<smartcore_bos_hail_v1_hail_pb.ListHailsResponse>;

  pullHails(
    request: smartcore_bos_hail_v1_hail_pb.PullHailsRequest,
    metadata?: grpcWeb.Metadata
  ): grpcWeb.ClientReadableStream<smartcore_bos_hail_v1_hail_pb.PullHailsResponse>;

}

export class HailInfoClient {
  constructor (hostname: string,
               credentials?: null | { [index: string]: string; },
               options?: null | { [index: string]: any; });

  describeHail(
    request: smartcore_bos_hail_v1_hail_pb.DescribeHailRequest,
    metadata: grpcWeb.Metadata | undefined,
    callback: (err: grpcWeb.RpcError,
               response: smartcore_bos_hail_v1_hail_pb.HailSupport) => void
  ): grpcWeb.ClientReadableStream<smartcore_bos_hail_v1_hail_pb.HailSupport>;

}

export class HailApiPromiseClient {
  constructor (hostname: string,
               credentials?: null | { [index: string]: string; },
               options?: null | { [index: string]: any; });

  createHail(
    request: smartcore_bos_hail_v1_hail_pb.CreateHailRequest,
    metadata?: grpcWeb.Metadata
  ): Promise<smartcore_bos_hail_v1_hail_pb.Hail>;

  getHail(
    request: smartcore_bos_hail_v1_hail_pb.GetHailRequest,
    metadata?: grpcWeb.Metadata
  ): Promise<smartcore_bos_hail_v1_hail_pb.Hail>;

  updateHail(
    request: smartcore_bos_hail_v1_hail_pb.UpdateHailRequest,
    metadata?: grpcWeb.Metadata
  ): Promise<smartcore_bos_hail_v1_hail_pb.Hail>;

  deleteHail(
    request: smartcore_bos_hail_v1_hail_pb.DeleteHailRequest,
    metadata?: grpcWeb.Metadata
  ): Promise<smartcore_bos_hail_v1_hail_pb.DeleteHailResponse>;

  pullHail(
    request: smartcore_bos_hail_v1_hail_pb.PullHailRequest,
    metadata?: grpcWeb.Metadata
  ): grpcWeb.ClientReadableStream<smartcore_bos_hail_v1_hail_pb.PullHailResponse>;

  listHails(
    request: smartcore_bos_hail_v1_hail_pb.ListHailsRequest,
    metadata?: grpcWeb.Metadata
  ): Promise<smartcore_bos_hail_v1_hail_pb.ListHailsResponse>;

  pullHails(
    request: smartcore_bos_hail_v1_hail_pb.PullHailsRequest,
    metadata?: grpcWeb.Metadata
  ): grpcWeb.ClientReadableStream<smartcore_bos_hail_v1_hail_pb.PullHailsResponse>;

}

export class HailInfoPromiseClient {
  constructor (hostname: string,
               credentials?: null | { [index: string]: string; },
               options?: null | { [index: string]: any; });

  describeHail(
    request: smartcore_bos_hail_v1_hail_pb.DescribeHailRequest,
    metadata?: grpcWeb.Metadata
  ): Promise<smartcore_bos_hail_v1_hail_pb.HailSupport>;

}

