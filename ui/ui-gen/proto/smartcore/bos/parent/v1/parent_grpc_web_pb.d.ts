import * as grpcWeb from 'grpc-web';

import * as smartcore_bos_parent_v1_parent_pb from '../../../../smartcore/bos/parent/v1/parent_pb'; // proto import: "smartcore/bos/parent/v1/parent.proto"


export class ParentApiClient {
  constructor (hostname: string,
               credentials?: null | { [index: string]: string; },
               options?: null | { [index: string]: any; });

  listChildren(
    request: smartcore_bos_parent_v1_parent_pb.ListChildrenRequest,
    metadata: grpcWeb.Metadata | undefined,
    callback: (err: grpcWeb.RpcError,
               response: smartcore_bos_parent_v1_parent_pb.ListChildrenResponse) => void
  ): grpcWeb.ClientReadableStream<smartcore_bos_parent_v1_parent_pb.ListChildrenResponse>;

  pullChildren(
    request: smartcore_bos_parent_v1_parent_pb.PullChildrenRequest,
    metadata?: grpcWeb.Metadata
  ): grpcWeb.ClientReadableStream<smartcore_bos_parent_v1_parent_pb.PullChildrenResponse>;

}

export class ParentInfoClient {
  constructor (hostname: string,
               credentials?: null | { [index: string]: string; },
               options?: null | { [index: string]: any; });

}

export class ParentApiPromiseClient {
  constructor (hostname: string,
               credentials?: null | { [index: string]: string; },
               options?: null | { [index: string]: any; });

  listChildren(
    request: smartcore_bos_parent_v1_parent_pb.ListChildrenRequest,
    metadata?: grpcWeb.Metadata
  ): Promise<smartcore_bos_parent_v1_parent_pb.ListChildrenResponse>;

  pullChildren(
    request: smartcore_bos_parent_v1_parent_pb.PullChildrenRequest,
    metadata?: grpcWeb.Metadata
  ): grpcWeb.ClientReadableStream<smartcore_bos_parent_v1_parent_pb.PullChildrenResponse>;

}

export class ParentInfoPromiseClient {
  constructor (hostname: string,
               credentials?: null | { [index: string]: string; },
               options?: null | { [index: string]: any; });

}

