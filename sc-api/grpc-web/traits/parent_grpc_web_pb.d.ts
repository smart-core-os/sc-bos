import * as grpcWeb from 'grpc-web';

import * as traits_parent_pb from '../traits/parent_pb'; // proto import: "traits/parent.proto"


export class ParentApiClient {
  constructor (hostname: string,
               credentials?: null | { [index: string]: string; },
               options?: null | { [index: string]: any; });

  listChildren(
    request: traits_parent_pb.ListChildrenRequest,
    metadata: grpcWeb.Metadata | undefined,
    callback: (err: grpcWeb.RpcError,
               response: traits_parent_pb.ListChildrenResponse) => void
  ): grpcWeb.ClientReadableStream<traits_parent_pb.ListChildrenResponse>;

  pullChildren(
    request: traits_parent_pb.PullChildrenRequest,
    metadata?: grpcWeb.Metadata
  ): grpcWeb.ClientReadableStream<traits_parent_pb.PullChildrenResponse>;

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
    request: traits_parent_pb.ListChildrenRequest,
    metadata?: grpcWeb.Metadata
  ): Promise<traits_parent_pb.ListChildrenResponse>;

  pullChildren(
    request: traits_parent_pb.PullChildrenRequest,
    metadata?: grpcWeb.Metadata
  ): grpcWeb.ClientReadableStream<traits_parent_pb.PullChildrenResponse>;

}

export class ParentInfoPromiseClient {
  constructor (hostname: string,
               credentials?: null | { [index: string]: string; },
               options?: null | { [index: string]: any; });

}

