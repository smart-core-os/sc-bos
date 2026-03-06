import * as grpcWeb from 'grpc-web';

import * as traits_metadata_pb from '../traits/metadata_pb'; // proto import: "traits/metadata.proto"


export class MetadataApiClient {
  constructor (hostname: string,
               credentials?: null | { [index: string]: string; },
               options?: null | { [index: string]: any; });

  getMetadata(
    request: traits_metadata_pb.GetMetadataRequest,
    metadata: grpcWeb.Metadata | undefined,
    callback: (err: grpcWeb.RpcError,
               response: traits_metadata_pb.Metadata) => void
  ): grpcWeb.ClientReadableStream<traits_metadata_pb.Metadata>;

  pullMetadata(
    request: traits_metadata_pb.PullMetadataRequest,
    metadata?: grpcWeb.Metadata
  ): grpcWeb.ClientReadableStream<traits_metadata_pb.PullMetadataResponse>;

}

export class MetadataInfoClient {
  constructor (hostname: string,
               credentials?: null | { [index: string]: string; },
               options?: null | { [index: string]: any; });

}

export class MetadataApiPromiseClient {
  constructor (hostname: string,
               credentials?: null | { [index: string]: string; },
               options?: null | { [index: string]: any; });

  getMetadata(
    request: traits_metadata_pb.GetMetadataRequest,
    metadata?: grpcWeb.Metadata
  ): Promise<traits_metadata_pb.Metadata>;

  pullMetadata(
    request: traits_metadata_pb.PullMetadataRequest,
    metadata?: grpcWeb.Metadata
  ): grpcWeb.ClientReadableStream<traits_metadata_pb.PullMetadataResponse>;

}

export class MetadataInfoPromiseClient {
  constructor (hostname: string,
               credentials?: null | { [index: string]: string; },
               options?: null | { [index: string]: any; });

}

